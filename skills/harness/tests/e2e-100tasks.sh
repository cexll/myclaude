#!/usr/bin/env bash
set -euo pipefail

# E2E test: 100 harness tasks + 5 self-reflection iterations via claude -p
# Usage: bash e2e-100tasks.sh

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(mktemp -d /tmp/harness-e2e-XXXXXX)"
LOG_FILE="${PROJECT_DIR}/test-output.log"

echo "=== Harness E2E Test: 100 tasks + 5 self-reflect ==="
echo "Project dir: ${PROJECT_DIR}"
echo ""

# --- 1. Generate harness-tasks.json with 100 trivial tasks ---
python3 - "${PROJECT_DIR}" <<'PYEOF'
import json, sys

root = sys.argv[1]
tasks = []
for i in range(1, 101):
    tid = f"task-{i:03d}"
    tasks.append({
        "id": tid,
        "title": f"Create file {tid}.txt",
        "status": "pending",
        "priority": "P1",
        "depends_on": [],
        "attempts": 0,
        "max_attempts": 3,
        "started_at_commit": None,
        "validation": {
            "command": f"test -f {tid}.txt && grep -q 'done-{tid}' {tid}.txt",
            "timeout_seconds": 10
        },
        "on_failure": {"cleanup": None},
        "error_log": [],
        "checkpoints": [],
        "completed_at": None
    })

state = {
    "version": 2,
    "created": "2026-03-01T00:00:00Z",
    "session_config": {
        "concurrency_mode": "exclusive",
        "max_tasks_per_session": 100,
        "max_sessions": 50,
        "max_reflect_iterations": 5
    },
    "tasks": tasks,
    "session_count": 0,
    "last_session": None
}

with open(f"{root}/harness-tasks.json", "w") as f:
    json.dump(state, f, indent=2, ensure_ascii=False)

print(f"Generated {len(tasks)} tasks")
PYEOF

# --- 2. Create progress log ---
touch "${PROJECT_DIR}/harness-progress.txt"

# --- 3. Create .harness-active marker ---
touch "${PROJECT_DIR}/.harness-active"

# --- 4. Init git repo (required for harness commit tracking) ---
cd "${PROJECT_DIR}"
git init -q
git add harness-tasks.json harness-progress.txt .harness-active
git commit -q -m "harness init"

echo "Setup complete. Running claude -p ..."
echo ""

# --- 5. Build the prompt ---
PROMPT="$(cat <<'PROMPT_EOF'
You are in a project with a harness setup. Run /harness run to execute all tasks.

The project is at the current working directory. There are 100 tasks in harness-tasks.json.
Each task requires creating a file: for task-001, create task-001.txt with content "done-task-001".

Execute the harness infinite loop protocol:
1. Read harness-tasks.json and harness-progress.txt
2. Pick next eligible task by priority
3. For each task: create the file with the required content, run validation, mark completed
4. Continue until all tasks are done
5. After completion, the self-reflect stop hook will trigger 5 times — complete those iterations

IMPORTANT: Do NOT use any skill tools. Just directly create files and update harness state.
For efficiency, you can batch multiple file creations in a single command.
After creating files, update harness-tasks.json to mark them completed.
Do all work directly — no planning mode, no subagents.
PROMPT_EOF
)"

# --- 6. Run claude -p ---
START_TIME=$(date +%s)

cd "${PROJECT_DIR}"
unset CLAUDECODE
REFLECT_MAX_ITERATIONS=5 \
HARNESS_STATE_ROOT="${PROJECT_DIR}" \
claude -p "${PROMPT}" \
  --model sonnet \
  --dangerously-skip-permissions \
  --disable-slash-commands \
  --no-session-persistence \
  --max-budget-usd 5 \
  --allowedTools 'Bash(*)' 'Read' 'Write' 'Glob' 'Grep' 'Edit' \
  2>&1 | tee "${LOG_FILE}"

END_TIME=$(date +%s)
ELAPSED=$((END_TIME - START_TIME))

echo ""
echo "=== Test Results ==="
echo "Duration: ${ELAPSED}s"
echo ""

# --- 7. Verify results ---
python3 - "${PROJECT_DIR}" <<'VERIFY_EOF'
import json, sys, os
from pathlib import Path

root = Path(sys.argv[1])
tasks_path = root / "harness-tasks.json"
progress_path = root / "harness-progress.txt"

# Check task files created
created = 0
for i in range(1, 101):
    tid = f"task-{i:03d}"
    fpath = root / f"{tid}.txt"
    if fpath.is_file():
        content = fpath.read_text().strip()
        if f"done-{tid}" in content:
            created += 1

# Check task statuses
with tasks_path.open() as f:
    state = json.load(f)
tasks = state.get("tasks", [])
completed = sum(1 for t in tasks if t.get("status") == "completed")
failed = sum(1 for t in tasks if t.get("status") == "failed")
pending = sum(1 for t in tasks if t.get("status") == "pending")
in_progress = sum(1 for t in tasks if t.get("status") == "in_progress")

# Check .harness-active removed
marker_removed = not (root / ".harness-active").is_file()

# Check progress log
progress_lines = 0
if progress_path.is_file():
    progress_lines = len([l for l in progress_path.read_text().splitlines() if l.strip()])

print(f"Files created:     {created}/100")
print(f"Tasks completed:   {completed}/100")
print(f"Tasks failed:      {failed}")
print(f"Tasks pending:     {pending}")
print(f"Tasks in_progress: {in_progress}")
print(f"Marker removed:    {marker_removed}")
print(f"Progress log lines: {progress_lines}")
print()

if created >= 95 and completed >= 95:
    print("PASS: >= 95% tasks completed successfully")
    sys.exit(0)
else:
    print(f"PARTIAL: {created} files, {completed} completed")
    print("Check the log for details")
    sys.exit(1)
VERIFY_EOF

echo ""
echo "Log: ${LOG_FILE}"
echo "Project: ${PROJECT_DIR}"
