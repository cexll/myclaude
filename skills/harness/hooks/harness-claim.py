#!/usr/bin/env python3
from __future__ import annotations

import datetime as _dt
import hashlib
import json
import os
import shutil
import socket
import sys
import time
from pathlib import Path
from typing import Any, Optional


def _utc_now() -> _dt.datetime:
    return _dt.datetime.now(tz=_dt.timezone.utc)


def _iso_z(dt: _dt.datetime) -> str:
    dt = dt.astimezone(_dt.timezone.utc).replace(microsecond=0)
    return dt.isoformat().replace("+00:00", "Z")


def _parse_iso(ts: Any) -> Optional[_dt.datetime]:
    if not isinstance(ts, str) or not ts.strip():
        return None
    s = ts.strip()
    if s.endswith("Z"):
        s = s[:-1] + "+00:00"
    try:
        dt = _dt.datetime.fromisoformat(s)
    except Exception:
        return None
    if dt.tzinfo is None:
        dt = dt.replace(tzinfo=_dt.timezone.utc)
    return dt.astimezone(_dt.timezone.utc)


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


def _priority_rank(v: Any) -> int:
    return {"P0": 0, "P1": 1, "P2": 2}.get(str(v or ""), 9)


def _eligible_tasks(tasks: list[dict[str, Any]]) -> tuple[list[dict[str, Any]], list[dict[str, Any]]]:
    completed = {str(t.get("id", "")) for t in tasks if str(t.get("status", "")) == "completed"}

    def deps_ok(t: dict[str, Any]) -> bool:
        deps = t.get("depends_on") or []
        if not isinstance(deps, list):
            return False
        return all(str(d) in completed for d in deps)

    def attempts(t: dict[str, Any]) -> int:
        try:
            return int(t.get("attempts") or 0)
        except Exception:
            return 0

    def max_attempts(t: dict[str, Any]) -> int:
        try:
            v = t.get("max_attempts")
            return int(v) if v is not None else 3
        except Exception:
            return 3

    pending = [t for t in tasks if str(t.get("status", "")) == "pending" and deps_ok(t)]
    retry = [
        t
        for t in tasks
        if str(t.get("status", "")) == "failed"
        and attempts(t) < max_attempts(t)
        and deps_ok(t)
    ]

    def key(t: dict[str, Any]) -> tuple[int, str]:
        return (_priority_rank(t.get("priority")), str(t.get("id", "")))

    pending.sort(key=key)
    retry.sort(key=key)
    return pending, retry


def _reap_stale_leases(tasks: list[dict[str, Any]], now: _dt.datetime) -> bool:
    changed = False
    for t in tasks:
        if str(t.get("status", "")) != "in_progress":
            continue
        exp = _parse_iso(t.get("lease_expires_at"))
        if exp is None or exp > now:
            continue

        try:
            t["attempts"] = int(t.get("attempts") or 0) + 1
        except Exception:
            t["attempts"] = 1

        err = f"[SESSION_TIMEOUT] Lease expired (claimed_by={t.get('claimed_by')})"
        log = t.get("error_log")
        if isinstance(log, list):
            log.append(err)
        else:
            t["error_log"] = [err]

        t["status"] = "failed"
        t.pop("claimed_by", None)
        t.pop("lease_expires_at", None)
        t.pop("claimed_at", None)
        changed = True
    return changed


def main() -> int:
    payload = _read_payload()
    root = _find_state_root(payload)
    if root is None:
        print(json.dumps({"claimed": False, "error": "state root not found"}, ensure_ascii=False))
        return 0

    tasks_path = root / "harness-tasks.json"
    lockdir = _lockdir_for_root(root)

    timeout_s = float(os.environ.get("HARNESS_LOCK_TIMEOUT_SECONDS") or "5")
    _acquire_lock(lockdir, timeout_s)
    try:
        state = _load_state(tasks_path)
        session_config = state.get("session_config") or {}
        if not isinstance(session_config, dict):
            session_config = {}
        concurrency_mode = str(session_config.get("concurrency_mode") or "exclusive")
        is_concurrent = concurrency_mode == "concurrent"
        tasks_raw = state.get("tasks") or []
        if not isinstance(tasks_raw, list):
            raise ValueError("tasks must be a list")
        tasks = [t for t in tasks_raw if isinstance(t, dict)]

        now = _utc_now()
        if _reap_stale_leases(tasks, now):
            state["tasks"] = tasks
            _atomic_write_json(tasks_path, state)

        pending, retry = _eligible_tasks(tasks)
        task = pending[0] if pending else (retry[0] if retry else None)
        if task is None:
            print(json.dumps({"claimed": False}, ensure_ascii=False))
            return 0

        worker_id = os.environ.get("HARNESS_WORKER_ID") or ""
        if is_concurrent and not worker_id:
            print(json.dumps({"claimed": False, "error": "missing HARNESS_WORKER_ID"}, ensure_ascii=False))
            return 0
        if not worker_id:
            worker_id = f"{socket.gethostname()}:{os.getpid()}"
        lease_seconds = int(os.environ.get("HARNESS_LEASE_SECONDS") or "1800")
        exp = now + _dt.timedelta(seconds=lease_seconds)

        task["status"] = "in_progress"
        task["claimed_by"] = worker_id
        task["claimed_at"] = _iso_z(now)
        task["lease_expires_at"] = _iso_z(exp)
        state["tasks"] = tasks
        _atomic_write_json(tasks_path, state)

        out = {
            "claimed": True,
            "worker_id": worker_id,
            "task_id": str(task.get("id") or ""),
            "title": str(task.get("title") or ""),
            "lease_expires_at": task["lease_expires_at"],
        }
        print(json.dumps(out, ensure_ascii=False))
        return 0
    finally:
        _release_lock(lockdir)


if __name__ == "__main__":
    raise SystemExit(main())
