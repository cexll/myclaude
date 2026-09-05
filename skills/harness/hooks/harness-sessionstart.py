#!/usr/bin/env python3
from __future__ import annotations

import json
import os
import sys
from pathlib import Path
from typing import Any, Optional


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


def _tail_text(path: Path, max_bytes: int = 8192) -> str:
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


def _pick_next_eligible(tasks: list[dict[str, Any]]) -> Optional[dict[str, Any]]:
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
    return pending[0] if pending else (retry[0] if retry else None)


def _is_harness_active(root: Path) -> bool:
    """Check if harness skill is actively running (marker file exists)."""
    return (root / ".harness-active").is_file()


def main() -> int:
    payload = _read_hook_payload()
    root = _find_harness_root(payload)
    if root is None:
        return 0

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
        context = f"HARNESS: CONFIG error: cannot read {tasks_path.name}: {e}"
        print(json.dumps({"hookSpecificOutput": {"additionalContext": context}}, ensure_ascii=False))
        return 0

    counts: dict[str, int] = {}
    for t in tasks:
        s = str(t.get("status") or "pending")
        counts[s] = counts.get(s, 0) + 1

    next_task = _pick_next_eligible(tasks)
    next_hint = ""
    if next_task is not None:
        tid = str(next_task.get("id") or "")
        title = str(next_task.get("title") or "").strip()
        next_hint = f" next={tid}{(': ' + title) if title else ''}"

    last_stats = ""
    if progress_path.is_file():
        tail = _tail_text(progress_path)
        lines = [ln.strip() for ln in tail.splitlines() if ln.strip()]
        for ln in reversed(lines[-200:]):
            if " STATS " in f" {ln} " or ln.endswith(" STATS"):
                last_stats = ln
                break
        if not last_stats and lines:
            last_stats = lines[-1]
        if len(last_stats) > 220:
            last_stats = last_stats[:217] + "..."

    summary = (
        "HARNESS: "
        + " ".join(f"{k}={v}" for k, v in sorted(counts.items()))
        + f" total={len(tasks)}"
        + next_hint
    ).strip()
    if last_stats:
        summary += f"\nHARNESS: last_log={last_stats}"

    print(json.dumps({"hookSpecificOutput": {"additionalContext": summary}}, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
