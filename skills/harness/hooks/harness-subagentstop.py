#!/usr/bin/env python3
"""Harness SubagentStop hook — blocks subagents from stopping when they
have assigned harness tasks still in progress.

Uses the same decision format as Stop hooks:
  {"decision": "block", "reason": "..."}
"""
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
        return {}


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


def _is_harness_active(root: Path) -> bool:
    """Check if harness skill is actively running (marker file exists)."""
    return (root / ".harness-active").is_file()


def main() -> int:
    payload = _read_hook_payload()

    # Safety: respect stop_hook_active to prevent infinite loops
    if payload.get("stop_hook_active", False):
        return 0

    root = _find_harness_root(payload)
    if root is None:
        return 0  # no harness project, allow stop

    # Guard: only active when harness skill is triggered
    if not _is_harness_active(root):
        return 0

    tasks_path = root / "harness-tasks.json"
    try:
        state = _load_json(tasks_path)
        session_config = state.get("session_config") or {}
        if not isinstance(session_config, dict):
            session_config = {}
        is_concurrent = str(session_config.get("concurrency_mode") or "exclusive") == "concurrent"
        tasks_raw = state.get("tasks") or []
        if not isinstance(tasks_raw, list):
            return 0
        tasks = [t for t in tasks_raw if isinstance(t, dict)]
    except Exception:
        return 0

    in_progress = [t for t in tasks if str(t.get("status", "")) == "in_progress"]
    worker_id = str(os.environ.get("HARNESS_WORKER_ID") or "").strip()
    agent_id = str(payload.get("agent_id") or "").strip()
    teammate_name = str(payload.get("teammate_name") or "").strip()
    identities = {x for x in (worker_id, agent_id, teammate_name) if x}

    if is_concurrent and in_progress and not identities:
        reason = (
            "HARNESS: concurrent 模式缺少 worker identity（HARNESS_WORKER_ID/agent_id）。"
            "为避免误停导致任务悬空，本次阻止停止。"
        )
        print(json.dumps({"decision": "block", "reason": reason}, ensure_ascii=False))
        return 0

    if is_concurrent:
        owned = [
            t for t in in_progress
            if str(t.get("claimed_by") or "") in identities
        ] if identities else []
    else:
        owned = in_progress

    # Only block when this subagent still owns in-progress work.
    if owned:
        tid = str(owned[0].get("id") or "")
        title = str(owned[0].get("title") or "")
        reason = (
            f"HARNESS: 子代理仍有进行中的任务 [{tid}] {title}。"
            "请完成当前任务的验证和记录后再停止。"
        )
        print(json.dumps({"decision": "block", "reason": reason}, ensure_ascii=False))
        return 0

    return 0  # all done, allow stop


if __name__ == "__main__":
    raise SystemExit(main())
