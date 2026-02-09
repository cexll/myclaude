#!/usr/bin/env python3
"""
Get context for current task.

Reads the current task's jsonl files and returns context for specified agent.
Used by inject-context hook to build agent prompts.
"""

import json
import os
import sys
from pathlib import Path

DIR_TASKS = ".claude/do-tasks"
FILE_CURRENT_TASK = ".current-task"
FILE_TASK_JSON = "task.json"


def get_project_root() -> str:
    return os.environ.get("CLAUDE_PROJECT_DIR", os.getcwd())


def get_current_task(project_root: str) -> str | None:
    current_task_file = os.path.join(project_root, DIR_TASKS, FILE_CURRENT_TASK)
    if not os.path.exists(current_task_file):
        return None
    try:
        with open(current_task_file, "r", encoding="utf-8") as f:
            content = f.read().strip()
            return content if content else None
    except Exception:
        return None


def read_file_content(base_path: str, file_path: str) -> str | None:
    full_path = os.path.join(base_path, file_path)
    if os.path.exists(full_path) and os.path.isfile(full_path):
        try:
            with open(full_path, "r", encoding="utf-8") as f:
                return f.read()
        except Exception:
            return None
    return None


def read_jsonl_entries(base_path: str, jsonl_path: str) -> list[tuple[str, str]]:
    full_path = os.path.join(base_path, jsonl_path)
    if not os.path.exists(full_path):
        return []

    results = []
    try:
        with open(full_path, "r", encoding="utf-8") as f:
            for line in f:
                line = line.strip()
                if not line:
                    continue
                try:
                    item = json.loads(line)
                    file_path = item.get("file") or item.get("path")
                    if not file_path:
                        continue
                    content = read_file_content(base_path, file_path)
                    if content:
                        results.append((file_path, content))
                except json.JSONDecodeError:
                    continue
    except Exception:
        pass
    return results


def get_agent_context(project_root: str, task_dir: str, agent_type: str) -> str:
    """Get complete context for specified agent."""
    context_parts = []

    # Read agent-specific jsonl
    agent_jsonl = os.path.join(task_dir, f"{agent_type}.jsonl")
    agent_entries = read_jsonl_entries(project_root, agent_jsonl)

    for file_path, content in agent_entries:
        context_parts.append(f"=== {file_path} ===\n{content}")

    # Read prd.md
    prd_content = read_file_content(project_root, os.path.join(task_dir, "prd.md"))
    if prd_content:
        context_parts.append(f"=== {task_dir}/prd.md (Requirements) ===\n{prd_content}")

    return "\n\n".join(context_parts)


def get_task_info(project_root: str, task_dir: str) -> dict | None:
    """Get task.json data."""
    task_json_path = os.path.join(project_root, task_dir, FILE_TASK_JSON)
    if not os.path.exists(task_json_path):
        return None
    try:
        with open(task_json_path, "r", encoding="utf-8") as f:
            return json.load(f)
    except Exception:
        return None


def main():
    import argparse

    parser = argparse.ArgumentParser(description="Get context for current task")
    parser.add_argument("agent", nargs="?", choices=["implement", "check", "debug"],
                        help="Agent type (optional, returns task info if not specified)")
    parser.add_argument("--json", action="store_true", help="Output as JSON")
    args = parser.parse_args()

    project_root = get_project_root()
    task_dir = get_current_task(project_root)

    if not task_dir:
        if args.json:
            print(json.dumps({"error": "No active task"}))
        else:
            print("No active task.", file=sys.stderr)
        sys.exit(1)

    task_info = get_task_info(project_root, task_dir)

    if not args.agent:
        if args.json:
            print(json.dumps({"task_dir": task_dir, "task_info": task_info}))
        else:
            print(f"Task: {task_dir}")
            if task_info:
                print(f"Title: {task_info.get('title', 'N/A')}")
                print(f"Phase: {task_info.get('current_phase', '?')}/{task_info.get('max_phases', 5)}")
        sys.exit(0)

    context = get_agent_context(project_root, task_dir, args.agent)

    if args.json:
        print(json.dumps({
            "task_dir": task_dir,
            "agent": args.agent,
            "context": context,
            "task_info": task_info,
        }))
    else:
        print(context)


if __name__ == "__main__":
    main()
