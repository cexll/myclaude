#!/usr/bin/env python3
import argparse
import os
import secrets
import subprocess
import sys
import time

PHASE_NAMES = {
    1: "Understand",
    2: "Clarify",
    3: "Design",
    4: "Implement",
    5: "Complete",
}

def phase_name_for(n: int) -> str:
    return PHASE_NAMES.get(n, f"Phase {n}")

def die(msg: str):
    print(f"❌ {msg}", file=sys.stderr)
    sys.exit(1)


def create_worktree(project_dir: str, task_id: str) -> str:
    """Create a git worktree for the task. Returns the worktree directory path."""
    # Get git root
    result = subprocess.run(
        ["git", "-C", project_dir, "rev-parse", "--show-toplevel"],
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        die(f"Not a git repository: {project_dir}")
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
        die(f"Failed to create worktree: {result.stderr}")

    return worktree_dir

def main():
    parser = argparse.ArgumentParser(
        description="Creates (or overwrites) project state file: .claude/do.local.md"
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

    max_phases = args.max_phases
    completion_promise = args.completion_promise
    use_worktree = args.worktree
    prompt = " ".join(args.prompt)

    if max_phases < 1:
        die("--max-phases must be a positive integer")

    project_dir = os.environ.get("CLAUDE_PROJECT_DIR", os.getcwd())
    state_dir = os.path.join(project_dir, ".claude")

    task_id = f"{int(time.time())}-{os.getpid()}-{secrets.token_hex(4)}"
    state_file = os.path.join(state_dir, f"do.{task_id}.local.md")

    os.makedirs(state_dir, exist_ok=True)

    # Create worktree if requested (before writing state file)
    worktree_dir = ""
    if use_worktree:
        worktree_dir = create_worktree(project_dir, task_id)

    phase_name = phase_name_for(1)

    content = f"""---
active: true
current_phase: 1
phase_name: "{phase_name}"
max_phases: {max_phases}
completion_promise: "{completion_promise}"
use_worktree: {str(use_worktree).lower()}
worktree_dir: "{worktree_dir}"
---

# do loop state

## Prompt
{prompt}

## Notes
- Update frontmatter current_phase/phase_name as you progress
- When complete, include the frontmatter completion_promise in your final output
"""

    with open(state_file, "w", encoding="utf-8") as f:
        f.write(content)

    print(f"Initialized: {state_file}")
    print(f"task_id: {task_id}")
    print(f"phase: 1/{max_phases} ({phase_name})")
    print(f"completion_promise: {completion_promise}")
    print(f"use_worktree: {use_worktree}")
    print(f"export DO_TASK_ID={task_id}")
    if worktree_dir:
        print(f"worktree_dir: {worktree_dir}")
        print(f"export DO_WORKTREE_DIR={worktree_dir}")

if __name__ == "__main__":
    main()
