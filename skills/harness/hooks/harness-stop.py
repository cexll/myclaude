#!/usr/bin/env python3
"""Harness Stop hook — blocks Claude from stopping when eligible tasks remain.

Uses `stop_hook_active` field and a consecutive-block counter to prevent
infinite loops. If the hook blocks N times in a row without any task
completing, it allows the stop with a warning.
"""
from __future__ import annotations

import json
import os
import sys
from pathlib import Path
from typing import Any, Optional

MAX_CONSECUTIVE_BLOCKS = 8  # safety valve


def _read_hook_payload() -> dict[str, Any]:
    raw = sys.stdin.read()
    if not raw.strip():
        return {}
    try:
        data = json.loads(raw)
        return data if isinstance(data, dict) else {}
    except Exception:
        return {"_invalid_json": True}


def _find_harness_root(payload: dict[str, Any]) -> Optional[Path]:
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


def _load_json(path: Path) -> dict[str, Any]:
    with path.open("r", encoding="utf-8") as f:
        data = json.load(f)
    if not isinstance(data, dict):
        raise ValueError(f"{path.name} must be a JSON object")
    return data


def _tail_text(path: Path, max_bytes: int = 200_000) -> str:
    with path.open("rb") as f:
        try:
            f.seek(0, os.SEEK_END)
            size = f.tell()
            f.seek(max(0, size - max_bytes), os.SEEK_SET)
        except Exception:
            f.seek(0, os.SEEK_SET)
        chunk = f.read()
    return chunk.decode("utf-8", errors="replace")


def _priority_rank(v: Any) -> int:
    return {"P0": 0, "P1": 1, "P2": 2}.get(str(v or ""), 9)


def _deps_completed(t: dict[str, Any], completed: set[str]) -> bool:
    deps = t.get("depends_on") or []
    if not isinstance(deps, list):
        return False
    return all(str(d) in completed for d in deps)


def _attempts(t: dict[str, Any]) -> int:
    try:
        return int(t.get("attempts") or 0)
    except Exception:
        return 0


def _max_attempts(t: dict[str, Any]) -> int:
    try:
        v = t.get("max_attempts")
        return int(v) if v is not None else 3
    except Exception:
        return 3


def _pick_next(pending: list[dict[str, Any]], retry: list[dict[str, Any]]) -> Optional[dict[str, Any]]:
    def key(t: dict[str, Any]) -> tuple[int, str]:
        return (_priority_rank(t.get("priority")), str(t.get("id", "")))
    pending.sort(key=key)
    retry.sort(key=key)
    return pending[0] if pending else (retry[0] if retry else None)


def _block_counter_path(root: Path) -> Path:
    return root / ".harness-stop-counter"


def _read_block_counter(root: Path) -> tuple[int, int]:
    """Returns (consecutive_blocks, last_completed_count)."""
    p = _block_counter_path(root)
    try:
        raw = p.read_text("utf-8").strip()
        parts = raw.split(",")
        return int(parts[0]), int(parts[1]) if len(parts) > 1 else 0
    except Exception:
        return 0, 0


def _write_block_counter(root: Path, blocks: int, completed: int) -> None:
    p = _block_counter_path(root)
    tmp = p.with_name(f"{p.name}.tmp.{os.getpid()}")
    try:
        tmp.write_text(f"{blocks},{completed}", encoding="utf-8")
        os.replace(tmp, p)
    except Exception:
        try:
            tmp.unlink(missing_ok=True)
        except Exception:
            pass


def _reset_block_counter(root: Path) -> None:
    p = _block_counter_path(root)
    try:
        p.unlink(missing_ok=True)
    except Exception:
        pass


def _is_harness_active(root: Path) -> bool:
    """Check if harness skill is actively running (marker file exists)."""
    return (root / ".harness-active").is_file()


def main() -> int:
    payload = _read_hook_payload()

    # Safety: if stop_hook_active is True, Claude is already continuing
    # from a previous Stop hook block. Check if we should allow stop
    # to prevent infinite loops.
    stop_hook_active = payload.get("stop_hook_active", False)

    root = _find_harness_root(payload)
    if root is None:
        return 0  # no harness project, allow stop

    # Guard: only active when harness skill is triggered
    if not _is_harness_active(root):
        return 0

    tasks_path = root / "harness-tasks.json"
    progress_path = root / "harness-progress.txt"
    try:
        state = _load_json(tasks_path)
        tasks_raw = state.get("tasks") or []
        if not isinstance(tasks_raw, list):
            raise ValueError("tasks must be a list")
        tasks = [t for t in tasks_raw if isinstance(t, dict)]
    except Exception as e:
        if stop_hook_active:
            sys.stderr.write(
                "HARNESS: WARN — harness-tasks.json 无法解析且 stop_hook_active=True，"
                "为避免无限循环，本次允许停止。\n"
            )
            return 0
        reason = (
            "HARNESS: 检测到配置损坏，无法解析 harness-tasks.json。\n"
            f"HARNESS: error={e}\n"
            "按 SKILL.md 的 JSON corruption 恢复：优先用 harness-tasks.json.bak 还原；无法还原则停止并要求人工修复。"
        )
        print(json.dumps({"decision": "block", "reason": reason}, ensure_ascii=False))
        return 0

    session_config = state.get("session_config") or {}
    if not isinstance(session_config, dict):
        session_config = {}

    concurrency_mode = str(session_config.get("concurrency_mode") or "exclusive")
    is_concurrent = concurrency_mode == "concurrent"
    worker_id = os.environ.get("HARNESS_WORKER_ID") or None

    # Check session limits
    try:
        session_count = int(state.get("session_count") or 0)
    except Exception:
        session_count = 0
    try:
        max_sessions = int(session_config.get("max_sessions") or 0)
    except Exception:
        max_sessions = 0
    if max_sessions > 0 and session_count >= max_sessions:
        _reset_block_counter(root)
        return 0  # session limit reached, allow stop

    # Check per-session task limit
    try:
        max_tasks_per_session = int(session_config.get("max_tasks_per_session") or 0)
    except Exception:
        max_tasks_per_session = 0
    if not is_concurrent and max_tasks_per_session > 0 and session_count > 0 and progress_path.is_file():
        tail = _tail_text(progress_path)
        tag = f"[SESSION-{session_count}]"
        finished = 0
        for ln in tail.splitlines():
            if tag not in ln:
                continue
            if " Completed [" in ln or (" ERROR [" in ln and "[task-" in ln):
                finished += 1
        if finished >= max_tasks_per_session:
            _reset_block_counter(root)
            return 0  # per-session limit reached, allow stop

    # Compute eligible tasks
    counts: dict[str, int] = {}
    for t in tasks:
        s = str(t.get("status") or "pending")
        counts[s] = counts.get(s, 0) + 1

    completed_ids = {str(t.get("id", "")) for t in tasks if str(t.get("status", "")) == "completed"}
    completed_count = len(completed_ids)

    pending_eligible = [t for t in tasks if str(t.get("status", "")) == "pending" and _deps_completed(t, completed_ids)]
    retryable = [
        t for t in tasks
        if str(t.get("status", "")) == "failed"
        and _attempts(t) < _max_attempts(t)
        and _deps_completed(t, completed_ids)
    ]
    in_progress_any = [t for t in tasks if str(t.get("status", "")) == "in_progress"]
    if is_concurrent and worker_id:
        in_progress_blocking = [
            t for t in in_progress_any
            if str(t.get("claimed_by") or "") == worker_id or not t.get("claimed_by")
        ]
    else:
        in_progress_blocking = in_progress_any

    # If nothing left to do, allow stop
    if not pending_eligible and not retryable and not in_progress_blocking:
        _reset_block_counter(root)
        try:
            (root / ".harness-active").unlink(missing_ok=True)
        except Exception:
            pass
        return 0

    # Safety valve: track consecutive blocks without progress
    prev_blocks, prev_completed = _read_block_counter(root)
    if completed_count > prev_completed:
        # Progress was made, reset counter
        prev_blocks = 0
    consecutive = prev_blocks + 1
    _write_block_counter(root, consecutive, completed_count)

    if stop_hook_active and consecutive > MAX_CONSECUTIVE_BLOCKS:
        # Too many consecutive blocks without progress — allow stop to prevent infinite loop
        _reset_block_counter(root)
        sys.stderr.write(
            f"HARNESS: WARN — Stop hook blocked {consecutive} times without progress. "
            "Allowing stop to prevent infinite loop. Check task definitions and validation commands.\n"
        )
        return 0

    # Block the stop — tasks remain
    next_task = _pick_next(pending_eligible, retryable)
    next_hint = ""
    if next_task is not None:
        tid = str(next_task.get("id") or "")
        title = str(next_task.get("title") or "").strip()
        next_hint = f"next={tid}{(': ' + title) if title else ''}"

    summary = (
        "HARNESS: 未满足停止条件，继续执行。\n"
        + "HARNESS: "
        + " ".join(f"{k}={v}" for k, v in sorted(counts.items()))
        + f" total={len(tasks)}"
        + (f" {next_hint}" if next_hint else "")
    ).strip()

    reason = (
        summary
        + "\n"
        + "请按 SKILL.md 的 Task Selection Algorithm 选择下一个 eligible 任务，并完整执行 Task Execution Cycle："
        "Claim → Checkpoint → Validate → Record outcome → STATS（如需）→ Continue。"
    )

    print(json.dumps({"decision": "block", "reason": reason}, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
