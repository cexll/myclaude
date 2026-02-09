#!/usr/bin/env python3
"""
Verify Loop Hook for do skill workflow.

SubagentStop hook that intercepts when code-reviewer agent tries to stop.
Runs verification commands to ensure code quality before allowing exit.

Mechanism:
- Intercepts SubagentStop event for code-reviewer agent
- Runs verify commands from task.json if configured
- Blocks stopping until verification passes
- Has max iterations as safety limit (MAX_ITERATIONS=5)

State file: .claude/do-tasks/.verify-state.json
"""

import json
import os
import subprocess
import sys
from datetime import datetime
from pathlib import Path

# Configuration
MAX_ITERATIONS = 5
STATE_TIMEOUT_MINUTES = 30
DIR_TASKS = ".claude/do-tasks"
FILE_CURRENT_TASK = ".current-task"
FILE_TASK_JSON = "task.json"
STATE_FILE = ".claude/do-tasks/.verify-state.json"

# Only control loop for code-reviewer agent
TARGET_AGENTS = {"code-reviewer"}


def get_project_root(cwd: str) -> str | None:
    """Find project root (directory with .claude folder)."""
    current = Path(cwd).resolve()
    while current != current.parent:
        if (current / ".claude").exists():
            return str(current)
        current = current.parent
    return None


def get_current_task(project_root: str) -> str | None:
    """Read current task directory path."""
    current_task_file = os.path.join(project_root, DIR_TASKS, FILE_CURRENT_TASK)
    if not os.path.exists(current_task_file):
        return None
    try:
        with open(current_task_file, "r", encoding="utf-8") as f:
            content = f.read().strip()
            return content if content else None
    except Exception:
        return None


def get_task_info(project_root: str, task_dir: str) -> dict | None:
    """Read task.json data."""
    task_json_path = os.path.join(project_root, task_dir, FILE_TASK_JSON)
    if not os.path.exists(task_json_path):
        return None
    try:
        with open(task_json_path, "r", encoding="utf-8") as f:
            return json.load(f)
    except Exception:
        return None


def get_verify_commands(task_info: dict) -> list[str]:
    """Get verify commands from task.json."""
    return task_info.get("verify_commands", [])


def run_verify_commands(project_root: str, commands: list[str]) -> tuple[bool, str]:
    """Run verify commands and return (success, message)."""
    for cmd in commands:
        try:
            result = subprocess.run(
                cmd,
                shell=True,
                cwd=project_root,
                capture_output=True,
                timeout=120,
            )
            if result.returncode != 0:
                stderr = result.stderr.decode("utf-8", errors="replace")
                stdout = result.stdout.decode("utf-8", errors="replace")
                error_output = stderr or stdout
                if len(error_output) > 500:
                    error_output = error_output[:500] + "..."
                return False, f"Command failed: {cmd}\n{error_output}"
        except subprocess.TimeoutExpired:
            return False, f"Command timed out: {cmd}"
        except Exception as e:
            return False, f"Command error: {cmd} - {str(e)}"
    return True, "All verify commands passed"


def load_state(project_root: str) -> dict:
    """Load verify loop state."""
    state_path = os.path.join(project_root, STATE_FILE)
    if not os.path.exists(state_path):
        return {"task": None, "iteration": 0, "started_at": None}
    try:
        with open(state_path, "r", encoding="utf-8") as f:
            return json.load(f)
    except Exception:
        return {"task": None, "iteration": 0, "started_at": None}


def save_state(project_root: str, state: dict) -> None:
    """Save verify loop state."""
    state_path = os.path.join(project_root, STATE_FILE)
    try:
        os.makedirs(os.path.dirname(state_path), exist_ok=True)
        with open(state_path, "w", encoding="utf-8") as f:
            json.dump(state, f, indent=2, ensure_ascii=False)
    except Exception:
        pass


def main():
    try:
        input_data = json.load(sys.stdin)
    except json.JSONDecodeError:
        sys.exit(0)

    hook_event = input_data.get("hook_event_name", "")
    if hook_event != "SubagentStop":
        sys.exit(0)

    subagent_type = input_data.get("subagent_type", "")
    agent_output = input_data.get("agent_output", "")
    cwd = input_data.get("cwd", os.getcwd())

    if subagent_type not in TARGET_AGENTS:
        sys.exit(0)

    project_root = get_project_root(cwd)
    if not project_root:
        sys.exit(0)

    task_dir = get_current_task(project_root)
    if not task_dir:
        sys.exit(0)

    task_info = get_task_info(project_root, task_dir)
    if not task_info:
        sys.exit(0)

    verify_commands = get_verify_commands(task_info)
    if not verify_commands:
        # No verify commands configured, allow exit
        sys.exit(0)

    # Load state
    state = load_state(project_root)

    # Reset state if task changed or too old
    should_reset = False
    if state.get("task") != task_dir:
        should_reset = True
    elif state.get("started_at"):
        try:
            started = datetime.fromisoformat(state["started_at"])
            if (datetime.now() - started).total_seconds() > STATE_TIMEOUT_MINUTES * 60:
                should_reset = True
        except (ValueError, TypeError):
            should_reset = True

    if should_reset:
        state = {
            "task": task_dir,
            "iteration": 0,
            "started_at": datetime.now().isoformat(),
        }

    # Increment iteration
    state["iteration"] = state.get("iteration", 0) + 1
    current_iteration = state["iteration"]
    save_state(project_root, state)

    # Safety check: max iterations
    if current_iteration >= MAX_ITERATIONS:
        state["iteration"] = 0
        save_state(project_root, state)
        output = {
            "decision": "allow",
            "reason": f"Max iterations ({MAX_ITERATIONS}) reached. Stopping to prevent infinite loop.",
        }
        print(json.dumps(output, ensure_ascii=False))
        sys.exit(0)

    # Run verify commands
    passed, message = run_verify_commands(project_root, verify_commands)

    if passed:
        state["iteration"] = 0
        save_state(project_root, state)
        output = {
            "decision": "allow",
            "reason": "All verify commands passed. Review phase complete.",
        }
        print(json.dumps(output, ensure_ascii=False))
        sys.exit(0)
    else:
        output = {
            "decision": "block",
            "reason": f"Iteration {current_iteration}/{MAX_ITERATIONS}. Verification failed:\n{message}\n\nPlease fix the issues and try again.",
        }
        print(json.dumps(output, ensure_ascii=False))
        sys.exit(0)


if __name__ == "__main__":
    main()
