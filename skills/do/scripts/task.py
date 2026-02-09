#!/usr/bin/env python3
"""
Task Directory Management CLI for do skill workflow.

Commands:
  create <title>              - Create a new task directory with task.md
  start <task-dir>            - Set current task pointer
  finish                      - Clear current task pointer
  list                        - List active tasks
  status                      - Show current task status
  update-phase <N>            - Update current phase
"""

import argparse
import os
import random
import re
import string
import subprocess
import sys
from datetime import datetime
from pathlib import Path

# Directory constants
DIR_TASKS = ".claude/do-tasks"
FILE_CURRENT_TASK = ".current-task"
FILE_TASK_MD = "task.md"

PHASE_NAMES = {
    1: "Understand",
    2: "Clarify",
    3: "Design",
    4: "Implement",
    5: "Complete",
}


def get_project_root() -> str:
    """Get project root from env or cwd."""
    return os.environ.get("CLAUDE_PROJECT_DIR", os.getcwd())


def get_tasks_dir(project_root: str) -> str:
    """Get tasks directory path."""
    return os.path.join(project_root, DIR_TASKS)


def get_current_task_file(project_root: str) -> str:
    """Get current task pointer file path."""
    return os.path.join(project_root, DIR_TASKS, FILE_CURRENT_TASK)


def generate_task_id() -> str:
    """Generate short task ID: MMDD-XXXX format."""
    date_part = datetime.now().strftime("%m%d")
    random_part = ''.join(random.choices(string.ascii_lowercase + string.digits, k=4))
    return f"{date_part}-{random_part}"


def read_task_md(task_md_path: str) -> dict | None:
    """Read task.md and parse YAML frontmatter + body."""
    if not os.path.exists(task_md_path):
        return None

    try:
        with open(task_md_path, "r", encoding="utf-8") as f:
            content = f.read()
    except Exception:
        return None

    # Parse YAML frontmatter
    match = re.match(r'^---\n(.*?)\n---\n(.*)$', content, re.DOTALL)
    if not match:
        return None

    frontmatter_str = match.group(1)
    body = match.group(2)

    # Simple YAML parsing (no external deps)
    frontmatter = {}
    for line in frontmatter_str.split('\n'):
        if ':' in line:
            key, value = line.split(':', 1)
            key = key.strip()
            value = value.strip()
            # Handle quoted strings
            if value.startswith('"') and value.endswith('"'):
                value = value[1:-1]
            elif value == 'true':
                value = True
            elif value == 'false':
                value = False
            elif value.isdigit():
                value = int(value)
            frontmatter[key] = value

    return {"frontmatter": frontmatter, "body": body}


def write_task_md(task_md_path: str, frontmatter: dict, body: str) -> bool:
    """Write task.md with YAML frontmatter + body."""
    try:
        lines = ["---"]
        for key, value in frontmatter.items():
            if isinstance(value, bool):
                lines.append(f"{key}: {str(value).lower()}")
            elif isinstance(value, int):
                lines.append(f"{key}: {value}")
            elif isinstance(value, str) and ('<' in value or '>' in value or ':' in value):
                lines.append(f'{key}: "{value}"')
            else:
                lines.append(f'{key}: "{value}"' if isinstance(value, str) else f"{key}: {value}")
        lines.append("---")
        lines.append("")
        lines.append(body)

        with open(task_md_path, "w", encoding="utf-8") as f:
            f.write('\n'.join(lines))
        return True
    except Exception:
        return False


def create_worktree(project_root: str, task_id: str) -> str:
    """Create a git worktree for the task. Returns the worktree directory path."""
    # Get git root
    result = subprocess.run(
        ["git", "-C", project_root, "rev-parse", "--show-toplevel"],
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        raise RuntimeError(f"Not a git repository: {project_root}")
    git_root = result.stdout.strip()

    # Calculate paths
    worktree_dir = os.path.join(git_root, ".worktrees", f"do-{task_id}")
    branch_name = f"do/{task_id}"

    # Create worktree with new branch
    result = subprocess.run(
        ["git", "-C", git_root, "worktree", "add", "-b", branch_name, worktree_dir],
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        raise RuntimeError(f"Failed to create worktree: {result.stderr}")

    return worktree_dir


def create_task(title: str, use_worktree: bool = False) -> dict:
    """Create a new task directory with task.md."""
    project_root = get_project_root()
    tasks_dir = get_tasks_dir(project_root)
    os.makedirs(tasks_dir, exist_ok=True)

    task_id = generate_task_id()
    task_dir = os.path.join(tasks_dir, task_id)

    os.makedirs(task_dir, exist_ok=True)

    # Create worktree if requested
    worktree_dir = ""
    if use_worktree:
        try:
            worktree_dir = create_worktree(project_root, task_id)
        except RuntimeError as e:
            print(f"Warning: {e}", file=sys.stderr)
            use_worktree = False

    frontmatter = {
        "id": task_id,
        "title": title,
        "status": "in_progress",
        "current_phase": 1,
        "phase_name": PHASE_NAMES[1],
        "max_phases": 5,
        "use_worktree": use_worktree,
        "worktree_dir": worktree_dir,
        "created_at": datetime.now().isoformat(),
        "completion_promise": "<promise>DO_COMPLETE</promise>",
    }

    body = f"""# Requirements

{title}

## Context

## Progress
"""

    task_md_path = os.path.join(task_dir, FILE_TASK_MD)
    write_task_md(task_md_path, frontmatter, body)

    current_task_file = get_current_task_file(project_root)
    relative_task_dir = os.path.relpath(task_dir, project_root)
    with open(current_task_file, "w", encoding="utf-8") as f:
        f.write(relative_task_dir)

    return {
        "task_dir": task_dir,
        "relative_path": relative_task_dir,
        "task_data": frontmatter,
        "worktree_dir": worktree_dir,
    }


def get_current_task(project_root: str) -> str | None:
    """Read current task directory path."""
    current_task_file = get_current_task_file(project_root)
    if not os.path.exists(current_task_file):
        return None

    try:
        with open(current_task_file, "r", encoding="utf-8") as f:
            content = f.read().strip()
            return content if content else None
    except Exception:
        return None


def start_task(task_dir: str) -> bool:
    """Set current task pointer."""
    project_root = get_project_root()
    tasks_dir = get_tasks_dir(project_root)

    if os.path.isabs(task_dir):
        full_path = task_dir
        relative_path = os.path.relpath(task_dir, project_root)
    else:
        if not task_dir.startswith(DIR_TASKS):
            full_path = os.path.join(tasks_dir, task_dir)
            relative_path = os.path.join(DIR_TASKS, task_dir)
        else:
            full_path = os.path.join(project_root, task_dir)
            relative_path = task_dir

    if not os.path.exists(full_path):
        print(f"Error: Task directory not found: {full_path}", file=sys.stderr)
        return False

    current_task_file = get_current_task_file(project_root)
    os.makedirs(os.path.dirname(current_task_file), exist_ok=True)

    with open(current_task_file, "w", encoding="utf-8") as f:
        f.write(relative_path)

    return True


def finish_task() -> bool:
    """Clear current task pointer."""
    project_root = get_project_root()
    current_task_file = get_current_task_file(project_root)

    if os.path.exists(current_task_file):
        os.remove(current_task_file)

    return True


def list_tasks() -> list[dict]:
    """List all task directories."""
    project_root = get_project_root()
    tasks_dir = get_tasks_dir(project_root)

    if not os.path.exists(tasks_dir):
        return []

    tasks = []
    current_task = get_current_task(project_root)

    for entry in sorted(os.listdir(tasks_dir), reverse=True):
        entry_path = os.path.join(tasks_dir, entry)
        if not os.path.isdir(entry_path):
            continue

        task_md_path = os.path.join(entry_path, FILE_TASK_MD)
        if not os.path.exists(task_md_path):
            continue

        parsed = read_task_md(task_md_path)
        if parsed:
            task_data = parsed["frontmatter"]
        else:
            task_data = {"id": entry, "title": entry, "status": "unknown"}

        relative_path = os.path.join(DIR_TASKS, entry)
        task_data["path"] = relative_path
        task_data["is_current"] = current_task == relative_path
        tasks.append(task_data)

    return tasks


def get_status() -> dict | None:
    """Get current task status."""
    project_root = get_project_root()
    current_task = get_current_task(project_root)

    if not current_task:
        return None

    task_dir = os.path.join(project_root, current_task)
    task_md_path = os.path.join(task_dir, FILE_TASK_MD)

    parsed = read_task_md(task_md_path)
    if not parsed:
        return None

    task_data = parsed["frontmatter"]
    task_data["path"] = current_task
    return task_data


def update_phase(phase: int) -> bool:
    """Update current task phase."""
    project_root = get_project_root()
    current_task = get_current_task(project_root)

    if not current_task:
        print("Error: No active task.", file=sys.stderr)
        return False

    task_dir = os.path.join(project_root, current_task)
    task_md_path = os.path.join(task_dir, FILE_TASK_MD)

    parsed = read_task_md(task_md_path)
    if not parsed:
        print("Error: task.md not found or invalid.", file=sys.stderr)
        return False

    frontmatter = parsed["frontmatter"]
    frontmatter["current_phase"] = phase
    frontmatter["phase_name"] = PHASE_NAMES.get(phase, f"Phase {phase}")

    if not write_task_md(task_md_path, frontmatter, parsed["body"]):
        print("Error: Failed to write task.md.", file=sys.stderr)
        return False

    return True


def main():
    parser = argparse.ArgumentParser(
        description="Task directory management for do skill workflow"
    )
    subparsers = parser.add_subparsers(dest="command", help="Available commands")

    # create command
    create_parser = subparsers.add_parser("create", help="Create a new task")
    create_parser.add_argument("title", nargs="+", help="Task title")
    create_parser.add_argument("--worktree", action="store_true", help="Enable worktree mode")

    # start command
    start_parser = subparsers.add_parser("start", help="Set current task")
    start_parser.add_argument("task_dir", help="Task directory path")

    # finish command
    subparsers.add_parser("finish", help="Clear current task")

    # list command
    subparsers.add_parser("list", help="List all tasks")

    # status command
    subparsers.add_parser("status", help="Show current task status")

    # update-phase command
    phase_parser = subparsers.add_parser("update-phase", help="Update current phase")
    phase_parser.add_argument("phase", type=int, help="Phase number (1-5)")

    args = parser.parse_args()

    if args.command == "create":
        title = " ".join(args.title)
        result = create_task(title, args.worktree)
        print(f"Created task: {result['relative_path']}")
        print(f"Task ID: {result['task_data']['id']}")
        print(f"Phase: 1/{result['task_data']['max_phases']} (Understand)")
        print(f"Worktree: {result['task_data']['use_worktree']}")

    elif args.command == "start":
        if start_task(args.task_dir):
            print(f"Started task: {args.task_dir}")
        else:
            sys.exit(1)

    elif args.command == "finish":
        if finish_task():
            print("Task finished, current task cleared.")
        else:
            sys.exit(1)

    elif args.command == "list":
        tasks = list_tasks()
        if not tasks:
            print("No tasks found.")
        else:
            for task in tasks:
                marker = "* " if task.get("is_current") else "  "
                phase = task.get("current_phase", "?")
                max_phase = task.get("max_phases", 5)
                status = task.get("status", "unknown")
                print(f"{marker}{task['id']} [{status}] phase {phase}/{max_phase}")
                print(f"    {task.get('title', 'No title')}")

    elif args.command == "status":
        status = get_status()
        if not status:
            print("No active task.")
        else:
            print(f"Task: {status['id']}")
            print(f"Title: {status.get('title', 'No title')}")
            print(f"Status: {status.get('status', 'unknown')}")
            print(f"Phase: {status.get('current_phase', '?')}/{status.get('max_phases', 5)}")
            print(f"Worktree: {status.get('use_worktree', False)}")
            print(f"Path: {status['path']}")

    elif args.command == "update-phase":
        if update_phase(args.phase):
            phase_name = PHASE_NAMES.get(args.phase, f"Phase {args.phase}")
            print(f"Updated to phase {args.phase} ({phase_name})")
        else:
            sys.exit(1)

    else:
        parser.print_help()
        sys.exit(1)


if __name__ == "__main__":
    main()
