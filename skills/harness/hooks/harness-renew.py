#!/usr/bin/env python3
from __future__ import annotations

import datetime as _dt
import hashlib
import json
import os
import shutil
import sys
import time
from pathlib import Path
from typing import Any, Optional


def _utc_now() -> _dt.datetime:
    return _dt.datetime.now(tz=_dt.timezone.utc)


def _iso_z(dt: _dt.datetime) -> str:
    dt = dt.astimezone(_dt.timezone.utc).replace(microsecond=0)
    return dt.isoformat().replace("+00:00", "Z")


def _read_payload() -> dict[str, Any]:
    raw = sys.stdin.read()
    if not raw.strip():
        return {}
    try:
        data = json.loads(raw)
        return data if isinstance(data, dict) else {}
    except Exception:
        return {}


def _find_state_root(payload: dict[str, Any]) -> Optional[Path]:
    state_root = os.environ.get("HARNESS_STATE_ROOT")
    if state_root:
        p = Path(state_root)
        if (p / "harness-tasks.json").is_file():
            try:
                return p.resolve()
            except Exception:
                return p

    candidates: list[Path] = []
    env_dir = os.environ.get("CLAUDE_PROJECT_DIR")
    if env_dir:
        candidates.append(Path(env_dir))

    cwd = payload.get("cwd") or os.getcwd()
    candidates.append(Path(cwd))

    seen: set[str] = set()
    for base in candidates:
        try:
            base = base.resolve()
        except Exception:
            continue
        if str(base) in seen:
            continue
        seen.add(str(base))
        for parent in [base, *list(base.parents)[:8]]:
            if (parent / "harness-tasks.json").is_file():
                return parent
    return None


def _lockdir_for_root(root: Path) -> Path:
    h = hashlib.sha256(str(root).encode("utf-8")).hexdigest()[:16]
    return Path("/tmp") / f"harness-{h}.lock"


def _pid_alive(pid: int) -> bool:
    try:
        os.kill(pid, 0)
        return True
    except Exception:
        return False


def _read_pid(lockdir: Path) -> Optional[int]:
    try:
        raw = (lockdir / "pid").read_text("utf-8").strip()
        return int(raw) if raw else None
    except Exception:
        return None


def _acquire_lock(lockdir: Path, timeout_seconds: float) -> None:
    deadline = time.time() + timeout_seconds
    missing_pid_since: Optional[float] = None
    while True:
        try:
            lockdir.mkdir(mode=0o700)
            (lockdir / "pid").write_text(str(os.getpid()), encoding="utf-8")
            return
        except FileExistsError:
            pid = _read_pid(lockdir)
            if pid is None:
                if missing_pid_since is None:
                    missing_pid_since = time.time()
                if time.time() - missing_pid_since < 1.0:
                    if time.time() >= deadline:
                        raise TimeoutError("lock busy (pid missing)")
                    time.sleep(0.05)
                    continue
            else:
                missing_pid_since = None
                if _pid_alive(pid):
                    if time.time() >= deadline:
                        raise TimeoutError(f"lock busy (pid={pid})")
                    time.sleep(0.05)
                    continue

            stale = lockdir.with_name(f"{lockdir.name}.stale.{os.getpid()}.{int(time.time())}")
            try:
                lockdir.rename(stale)
            except Exception:
                if time.time() >= deadline:
                    raise TimeoutError("lock contention")
                time.sleep(0.05)
                continue
            shutil.rmtree(stale, ignore_errors=True)
            missing_pid_since = None
            continue


def _release_lock(lockdir: Path) -> None:
    shutil.rmtree(lockdir, ignore_errors=True)


def _load_state(path: Path) -> dict[str, Any]:
    with path.open("r", encoding="utf-8") as f:
        data = json.load(f)
    if not isinstance(data, dict):
        raise ValueError("harness-tasks.json must be an object")
    return data


def _atomic_write_json(path: Path, data: dict[str, Any]) -> None:
    bak = path.with_name(f"{path.name}.bak")
    tmp = path.with_name(f"{path.name}.tmp")
    shutil.copy2(path, bak)
    tmp.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    os.replace(tmp, path)


def main() -> int:
    payload = _read_payload()
    root = _find_state_root(payload)
    if root is None:
        print(json.dumps({"renewed": False, "error": "state root not found"}, ensure_ascii=False))
        return 0

    task_id = os.environ.get("HARNESS_TASK_ID") or str(payload.get("task_id") or "").strip()
    if not task_id:
        print(json.dumps({"renewed": False, "error": "missing task_id"}, ensure_ascii=False))
        return 0

    worker_id = os.environ.get("HARNESS_WORKER_ID") or ""
    if not worker_id:
        print(json.dumps({"renewed": False, "error": "missing HARNESS_WORKER_ID"}, ensure_ascii=False))
        return 0
    lease_seconds = int(os.environ.get("HARNESS_LEASE_SECONDS") or "1800")

    tasks_path = root / "harness-tasks.json"
    lockdir = _lockdir_for_root(root)

    timeout_s = float(os.environ.get("HARNESS_LOCK_TIMEOUT_SECONDS") or "5")
    try:
        _acquire_lock(lockdir, timeout_s)
    except Exception as e:
        print(json.dumps({"renewed": False, "error": str(e)}, ensure_ascii=False))
        return 0

    try:
        state = _load_state(tasks_path)
        tasks_raw = state.get("tasks") or []
        if not isinstance(tasks_raw, list):
            raise ValueError("tasks must be a list")
        tasks = [t for t in tasks_raw if isinstance(t, dict)]

        task = next((t for t in tasks if str(t.get("id") or "") == task_id), None)
        if task is None:
            print(json.dumps({"renewed": False, "error": "task not found"}, ensure_ascii=False))
            return 0

        if str(task.get("status") or "") != "in_progress":
            print(json.dumps({"renewed": False, "error": "task not in_progress"}, ensure_ascii=False))
            return 0

        claimed_by = str(task.get("claimed_by") or "")
        if claimed_by and claimed_by != worker_id:
            print(json.dumps({"renewed": False, "error": "task owned by other worker"}, ensure_ascii=False))
            return 0

        now = _utc_now()
        exp = now + _dt.timedelta(seconds=lease_seconds)
        task["lease_expires_at"] = _iso_z(exp)
        task["claimed_by"] = worker_id
        state["tasks"] = tasks
        _atomic_write_json(tasks_path, state)

        print(json.dumps({"renewed": True, "task_id": task_id, "lease_expires_at": task["lease_expires_at"]}, ensure_ascii=False))
        return 0
    except Exception as e:
        print(json.dumps({"renewed": False, "error": str(e)}, ensure_ascii=False))
        return 0
    finally:
        _release_lock(lockdir)


if __name__ == "__main__":
    raise SystemExit(main())
