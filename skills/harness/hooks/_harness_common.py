"""Shared utilities for harness hooks.

Consolidates duplicated logic: payload reading, state root discovery,
JSON I/O, lock primitives, task eligibility, and ISO time helpers.

Ported from Codex harness hooks to Claude Code.
"""
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

# ---------------------------------------------------------------------------
# Time helpers
# ---------------------------------------------------------------------------

def utc_now() -> _dt.datetime:
    return _dt.datetime.now(tz=_dt.timezone.utc)


def iso_z(dt: _dt.datetime) -> str:
    dt = dt.astimezone(_dt.timezone.utc).replace(microsecond=0)
    return dt.isoformat().replace("+00:00", "Z")


def parse_iso(ts: Any) -> Optional[_dt.datetime]:
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


# ---------------------------------------------------------------------------
# Hook payload
# ---------------------------------------------------------------------------

def read_hook_payload() -> dict[str, Any]:
    """Read JSON payload from stdin (sent by Claude Code to command hooks)."""
    raw = sys.stdin.read()
    if not raw.strip():
        return {}
    try:
        data = json.loads(raw)
        return data if isinstance(data, dict) else {}
    except Exception:
        return {}


def maybe_log_hook_event(root: Path, payload: dict[str, Any], hook_script: str) -> None:
    """Optionally append a compact hook execution record to HARNESS_HOOK_LOG.

    This is opt-in debugging: when HARNESS_HOOK_LOG is unset, it is a no-op.
    Call this only after the .harness-active guard passes.
    """
    log_path = os.environ.get("HARNESS_HOOK_LOG")
    if not log_path:
        return

    entry: dict[str, Any] = {
        "ts": iso_z(utc_now()),
        "hook_script": hook_script,
        "hook_event_name": payload.get("hook_event_name"),
        "harness_root": str(root),
    }
    for k in (
        "session_id",
        "cwd",
        "source",
        "reason",
        "teammate_name",
        "team_name",
        "agent_id",
        "agent_type",
        "stop_hook_active",
    ):
        if k in payload:
            entry[k] = payload.get(k)

    try:
        Path(log_path).expanduser().parent.mkdir(parents=True, exist_ok=True)
        with Path(log_path).expanduser().open("a", encoding="utf-8") as f:
            f.write(json.dumps(entry, ensure_ascii=False) + "\n")
    except Exception:
        return


# ---------------------------------------------------------------------------
# State root discovery
# ---------------------------------------------------------------------------

def find_harness_root(payload: dict[str, Any]) -> Optional[Path]:
    """Locate the directory containing harness-tasks.json.

    Search order:
    1. HARNESS_STATE_ROOT env var
    2. CLAUDE_PROJECT_DIR env var (+ parents)
    3. payload["cwd"] / os.getcwd() (+ parents)
    """
    env_root = os.environ.get("HARNESS_STATE_ROOT")
    if env_root:
        p = Path(env_root)
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


def is_harness_active(root: Path) -> bool:
    """True when .harness-active marker exists (hooks are live)."""
    return (root / ".harness-active").is_file()


# ---------------------------------------------------------------------------
# JSON I/O
# ---------------------------------------------------------------------------

def load_json(path: Path) -> dict[str, Any]:
    with path.open("r", encoding="utf-8") as f:
        data = json.load(f)
    if not isinstance(data, dict):
        raise ValueError(f"{path.name} must be a JSON object")
    return data


def atomic_write_json(path: Path, data: dict[str, Any]) -> None:
    """Write JSON atomically: backup -> tmp -> rename."""
    bak = path.with_name(f"{path.name}.bak")
    tmp = path.with_name(f"{path.name}.tmp")
    shutil.copy2(path, bak)
    tmp.write_text(
        json.dumps(data, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )
    os.replace(tmp, path)


def tail_text(path: Path, max_bytes: int = 200_000) -> str:
    """Read the last max_bytes of a text file."""
    with path.open("rb") as f:
        try:
            f.seek(0, os.SEEK_END)
            size = f.tell()
            f.seek(max(0, size - max_bytes), os.SEEK_SET)
        except Exception:
            f.seek(0, os.SEEK_SET)
        chunk = f.read()
    return chunk.decode("utf-8", errors="replace")


# ---------------------------------------------------------------------------
# Lock primitives (mkdir-based, POSIX-portable)
# ---------------------------------------------------------------------------

def lockdir_for_root(root: Path) -> Path:
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


def acquire_lock(lockdir: Path, timeout_seconds: float = 5.0) -> None:
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

            stale = lockdir.with_name(
                f"{lockdir.name}.stale.{os.getpid()}.{int(time.time())}"
            )
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


def release_lock(lockdir: Path) -> None:
    shutil.rmtree(lockdir, ignore_errors=True)


# ---------------------------------------------------------------------------
# Task helpers
# ---------------------------------------------------------------------------

def priority_rank(v: Any) -> int:
    return {"P0": 0, "P1": 1, "P2": 2}.get(str(v or ""), 9)


def task_attempts(t: dict[str, Any]) -> int:
    try:
        return int(t.get("attempts") or 0)
    except Exception:
        return 0


def task_max_attempts(t: dict[str, Any]) -> int:
    try:
        v = t.get("max_attempts")
        return int(v) if v is not None else 3
    except Exception:
        return 3


def deps_completed(t: dict[str, Any], completed_ids: set[str]) -> bool:
    deps = t.get("depends_on") or []
    if not isinstance(deps, list):
        return False
    return all(str(d) in completed_ids for d in deps)


def parse_tasks(state: dict[str, Any]) -> list[dict[str, Any]]:
    """Extract validated task list from state dict."""
    tasks_raw = state.get("tasks") or []
    if not isinstance(tasks_raw, list):
        raise ValueError("tasks must be a list")
    return [t for t in tasks_raw if isinstance(t, dict)]


def completed_ids(tasks: list[dict[str, Any]]) -> set[str]:
    return {
        str(t.get("id", ""))
        for t in tasks
        if str(t.get("status", "")) == "completed"
    }


def eligible_tasks(
    tasks: list[dict[str, Any]],
) -> tuple[list[dict[str, Any]], list[dict[str, Any]]]:
    """Return (pending_eligible, retryable) sorted by priority then id."""
    done = completed_ids(tasks)

    pending = [
        t for t in tasks
        if str(t.get("status", "")) == "pending" and deps_completed(t, done)
    ]
    retry = [
        t for t in tasks
        if str(t.get("status", "")) == "failed"
        and task_attempts(t) < task_max_attempts(t)
        and deps_completed(t, done)
    ]

    def key(t: dict[str, Any]) -> tuple[int, str]:
        return (priority_rank(t.get("priority")), str(t.get("id", "")))

    pending.sort(key=key)
    retry.sort(key=key)
    return pending, retry


def pick_next(
    pending: list[dict[str, Any]], retry: list[dict[str, Any]]
) -> Optional[dict[str, Any]]:
    return pending[0] if pending else (retry[0] if retry else None)


def status_counts(tasks: list[dict[str, Any]]) -> dict[str, int]:
    counts: dict[str, int] = {}
    for t in tasks:
        s = str(t.get("status") or "pending")
        counts[s] = counts.get(s, 0) + 1
    return counts


def reap_stale_leases(
    tasks: list[dict[str, Any]], now: _dt.datetime
) -> bool:
    """Reset in_progress tasks with expired leases to failed. Returns True if any changed."""
    changed = False
    for t in tasks:
        if str(t.get("status", "")) != "in_progress":
            continue
        exp = parse_iso(t.get("lease_expires_at"))
        if exp is None or exp > now:
            continue

        t["attempts"] = task_attempts(t) + 1
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


# ---------------------------------------------------------------------------
# Session config helpers
# ---------------------------------------------------------------------------

def get_session_config(state: dict[str, Any]) -> dict[str, Any]:
    cfg = state.get("session_config") or {}
    return cfg if isinstance(cfg, dict) else {}


def is_concurrent(cfg: dict[str, Any]) -> bool:
    return str(cfg.get("concurrency_mode") or "exclusive") == "concurrent"


# ---------------------------------------------------------------------------
# Hook output helpers
# ---------------------------------------------------------------------------

def emit_block(reason: str) -> None:
    """Print a JSON block decision to stdout and exit 0."""
    print(json.dumps({"decision": "block", "reason": reason}, ensure_ascii=False))


def emit_allow(reason: str = "") -> None:
    """Print a JSON allow decision to stdout and exit 0."""
    out: dict[str, Any] = {"decision": "allow"}
    if reason:
        out["reason"] = reason
    print(json.dumps(out, ensure_ascii=False))


def emit_context(context: str) -> None:
    """Inject additional context via hookSpecificOutput."""
    print(json.dumps(
        {"hookSpecificOutput": {"additionalContext": context}},
        ensure_ascii=False,
    ))


def emit_json(data: dict[str, Any]) -> None:
    """Print arbitrary JSON to stdout."""
    print(json.dumps(data, ensure_ascii=False))
