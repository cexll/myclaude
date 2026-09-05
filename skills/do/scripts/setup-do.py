#!/usr/bin/env python3
"""
Initialize do skill workflow - wrapper around task.py.

Creates a task directory under .claude/do-tasks/ with:
- task.md: Task metadata (YAML frontmatter) + requirements (Markdown body)

If --worktree is specified, also creates a git worktree for isolated development.
"""

import argparse
import sys

from task import create_task, PHASE_NAMES


def die(msg: str):
    print(f"Error: {msg}", file=sys.stderr)
    sys.exit(1)


def main():
    parser = argparse.ArgumentParser(
        description="Initialize do skill workflow with task directory"
    )
    parser.add_argument("--max-phases", type=int, default=5, help="Default: 5")
    parser.add_argument(
        "--completion-promise",
        default="<promise>DO_COMPLETE</promise>",
        help="Default: <promise>DO_COMPLETE</promise>",
    )
    parser.add_argument("--worktree", action="store_true", help="Enable worktree mode")
    parser.add_argument("prompt", nargs="+", help="Task description")
    args = parser.parse_args()

    if args.max_phases < 1:
        die("--max-phases must be a positive integer")

    prompt = " ".join(args.prompt)
    result = create_task(title=prompt, use_worktree=args.worktree)

    task_data = result["task_data"]
    worktree_dir = result.get("worktree_dir", "")

    print(f"Initialized: {result['relative_path']}")
    print(f"task_id: {task_data['id']}")
    print(f"phase: 1/{task_data['max_phases']} ({PHASE_NAMES[1]})")
    print(f"completion_promise: {task_data['completion_promise']}")
    print(f"use_worktree: {task_data['use_worktree']}")
    print(f"export DO_TASK_DIR={result['relative_path']}")

    if worktree_dir:
        print(f"worktree_dir: {worktree_dir}")
        print(f"export DO_WORKTREE_DIR={worktree_dir}")


if __name__ == "__main__":
    main()
