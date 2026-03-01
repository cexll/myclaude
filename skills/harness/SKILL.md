---
name: harness
description: "This skill should be used for multi-session autonomous agent work requiring progress checkpointing, failure recovery, and task dependency management. Triggers on '/harness' command, or when a task involves many subtasks needing progress persistence, sleep/resume cycles across context windows, recovery from mid-task failures with partial state, or distributed work across multiple agent sessions. Synthesized from Anthropic and OpenAI engineering practices for long-running agents."
---

# Harness — Long-Running Agent Framework

Executable protocol enabling any agent task to run continuously across multiple sessions with automatic progress recovery, task dependency resolution, failure rollback, and standardized error handling.

## Design Principles

1. **Design for the agent, not the human** — Test output, docs, and task structure are the agent's primary interface
2. **Progress files ARE the context** — When context window resets, progress files + git history = full recovery
3. **Premature completion is the #1 failure mode** — Structured task lists with explicit completion criteria prevent declaring victory early
4. **Standardize everything grep-able** — ERROR on same line, structured timestamps, consistent prefixes
5. **Fast feedback loops** — Pre-compute stats, run smoke tests before full validation
6. **Idempotent everything** — Init scripts, task execution, environment setup must all be safe to re-run
7. **Fail safe, not fail silent** — Every failure must have an explicit recovery strategy

## Commands

```
/harness init <project-path>     # Initialize harness files in project
/harness run                     # Start/resume the infinite loop
/harness status                  # Show current progress and stats
/harness add "task description"  # Add a task to the list
```

## Activation Marker

Hooks only take effect when `.harness-active` marker file exists in the harness root (same directory as `harness-tasks.json`).
Hook 注册配置在 `hooks/hooks.json`。

- `/harness init` and `/harness run` MUST create this marker: `touch <project-path>/.harness-active`
- When all tasks complete (no pending/in_progress/retryable left), remove it: `rm <project-path>/.harness-active`
- Without this marker, all hooks are no-ops — they exit 0 immediately

## Progress Persistence (Dual-File System)

Maintain two files in the project working directory:

### harness-progress.txt (Append-Only Log)

Free-text log of all agent actions across sessions. Never truncate.

```
[2025-07-01T10:00:00Z] [SESSION-1] INIT Harness initialized for project /path/to/project
[2025-07-01T10:00:05Z] [SESSION-1] INIT Environment health check: PASS
[2025-07-01T10:00:10Z] [SESSION-1] LOCK acquired (pid=12345)
[2025-07-01T10:00:11Z] [SESSION-1] Starting [task-001] Implement user authentication (base=def5678)
[2025-07-01T10:05:00Z] [SESSION-1] CHECKPOINT [task-001] step=2/4 "auth routes created, tests pending"
[2025-07-01T10:15:30Z] [SESSION-1] Completed [task-001] (commit abc1234)
[2025-07-01T10:15:31Z] [SESSION-1] Starting [task-002] Add rate limiting (base=abc1234)
[2025-07-01T10:20:00Z] [SESSION-1] ERROR [task-002] [TASK_EXEC] Redis connection refused
[2025-07-01T10:20:01Z] [SESSION-1] ROLLBACK [task-002] git reset --hard abc1234
[2025-07-01T10:20:02Z] [SESSION-1] STATS tasks_total=5 completed=1 failed=1 pending=3 blocked=0 attempts_total=2 checkpoints=1
```

### harness-tasks.json (Structured State)

```json
{
  "version": 2,
  "created": "2025-07-01T10:00:00Z",
  "session_config": {
    "concurrency_mode": "exclusive",
    "max_tasks_per_session": 20,
    "max_sessions": 50
  },
  "tasks": [
    {
      "id": "task-001",
      "title": "Implement user authentication",
      "status": "completed",
      "priority": "P0",
      "depends_on": [],
      "attempts": 1,
      "max_attempts": 3,
      "started_at_commit": "def5678",
      "validation": {
        "command": "npm test -- --testPathPattern=auth",
        "timeout_seconds": 300
      },
      "on_failure": {
        "cleanup": null
      },
      "error_log": [],
      "checkpoints": [],
      "completed_at": "2025-07-01T10:15:30Z"
    },
    {
      "id": "task-002",
      "title": "Add rate limiting",
      "status": "failed",
      "priority": "P1",
      "depends_on": [],
      "attempts": 1,
      "max_attempts": 3,
      "started_at_commit": "abc1234",
      "validation": {
        "command": "npm test -- --testPathPattern=rate-limit",
        "timeout_seconds": 120
      },
      "on_failure": {
        "cleanup": "docker compose down redis"
      },
      "error_log": ["[TASK_EXEC] Redis connection refused"],
      "checkpoints": [],
      "completed_at": null
    },
    {
      "id": "task-003",
      "title": "Add OAuth providers",
      "status": "pending",
      "priority": "P1",
      "depends_on": ["task-001"],
      "attempts": 0,
      "max_attempts": 3,
      "started_at_commit": null,
      "validation": {
        "command": "npm test -- --testPathPattern=oauth",
        "timeout_seconds": 180
      },
      "on_failure": {
        "cleanup": null
      },
      "error_log": [],
      "checkpoints": [],
      "completed_at": null
    }
  ],
  "session_count": 1,
  "last_session": "2025-07-01T10:20:02Z"
}
```

Task statuses: `pending` → `in_progress` (transient, set only during active execution) → `completed` or `failed`. A task found as `in_progress` at session start means the previous session was interrupted — handle via Context Window Recovery Protocol.

In concurrent mode (see Concurrency Control), tasks may also carry claim metadata: `claimed_by` and `lease_expires_at` (ISO timestamp).

**Session boundary**: A session starts when the agent begins executing the Session Start protocol and ends when a Stopping Condition is met or the context window resets. Each session gets a unique `SESSION-N` identifier (N = `session_count` after increment).

## Concurrency Control

Before modifying `harness-tasks.json`, acquire an exclusive lock using portable `mkdir` (atomic on all POSIX systems, works on both macOS and Linux):

```bash
# Acquire lock (fail fast if another agent is running)
# Lock key must be stable even if invoked from a subdirectory.
ROOT="$PWD"
SEARCH="$PWD"
while [ "$SEARCH" != "/" ] && [ ! -f "$SEARCH/harness-tasks.json" ]; do
  SEARCH="$(dirname "$SEARCH")"
done
if [ -f "$SEARCH/harness-tasks.json" ]; then
  ROOT="$SEARCH"
fi

PWD_HASH="$(
  printf '%s' "$ROOT" |
    (shasum -a 256 2>/dev/null || sha256sum 2>/dev/null) |
    awk '{print $1}' |
    cut -c1-16
)"
LOCKDIR="/tmp/harness-${PWD_HASH:-unknown}.lock"
if ! mkdir "$LOCKDIR" 2>/dev/null; then
  # Check if lock holder is still alive
  LOCK_PID=$(cat "$LOCKDIR/pid" 2>/dev/null)
  if [ -n "$LOCK_PID" ] && kill -0 "$LOCK_PID" 2>/dev/null; then
    echo "ERROR: Another harness session is active (pid=$LOCK_PID)"; exit 1
  fi
  # Stale lock — atomically reclaim via mv to avoid TOCTOU race
  STALE="$LOCKDIR.stale.$$"
  if mv "$LOCKDIR" "$STALE" 2>/dev/null; then
    rm -rf "$STALE"
    mkdir "$LOCKDIR" || { echo "ERROR: Lock contention"; exit 1; }
    echo "WARN: Removed stale lock${LOCK_PID:+ from pid=$LOCK_PID}"
  else
    echo "ERROR: Another agent reclaimed the lock"; exit 1
  fi
fi
echo "$$" > "$LOCKDIR/pid"
trap 'rm -rf "$LOCKDIR"' EXIT
```

Log lock acquisition: `[timestamp] [SESSION-N] LOCK acquired (pid=<PID>)`
Log lock release: `[timestamp] [SESSION-N] LOCK released`

Modes:

- **Exclusive (default)**: hold the lock for the entire session (the `trap EXIT` handler releases it automatically). Any second session in the same state root fails fast.
- **Concurrent (opt-in via `session_config.concurrency_mode: "concurrent"`)**: treat this as a **state transaction lock**. Hold it only while reading/modifying/writing `harness-tasks.json` (including `.bak`/`.tmp`) and appending to `harness-progress.txt`. Release it immediately before doing real work.

Concurrent mode invariants:

- All workers MUST point at the same state root (the directory that contains `harness-tasks.json`). If you are using separate worktrees/clones, pin it explicitly (e.g., `HARNESS_STATE_ROOT=/abs/path/to/state-root`).
- Task selection is advisory; the real gate is **atomic claim** under the lock: set `status="in_progress"`, set `claimed_by` (stable worker id, e.g., `HARNESS_WORKER_ID`), set `lease_expires_at`. If claim fails (already `in_progress` with a valid lease), pick another eligible task and retry.
- Never run two workers in the same git working directory. Use separate worktrees/clones. Otherwise rollback (`git reset --hard` / `git clean -fd`) will destroy other workers.

## Infinite Loop Protocol

### Session Start (Execute Every Time)

1. **Read state**: Read last 200 lines of `harness-progress.txt` + full `harness-tasks.json`. If JSON is unparseable, see JSON corruption recovery in Error Handling.
2. **Read git**: Run `git log --oneline -20` and `git diff --stat` to detect uncommitted work
3. **Acquire lock** (mode-dependent): Exclusive mode fails if another session is active. Concurrent mode uses the lock only for state transactions.
4. **Recover interrupted tasks** (see Context Window Recovery below)
5. **Health check**: Run `harness-init.sh` if it exists
6. **Track session**: Increment `session_count` in JSON. Check `session_count` against `max_sessions` — if reached, log STATS and STOP. Initialize per-session task counter to 0.
7. **Pick next task** using Task Selection Algorithm below

### Task Selection Algorithm

Before selecting, run dependency validation:

1. **Cycle detection**: For each non-completed task, walk `depends_on` transitively. If any task appears in its own chain, mark it `failed` with `[DEPENDENCY] Circular dependency detected: task-A -> task-B -> task-A`. Self-references (`depends_on` includes own id) are also cycles.
2. **Blocked propagation**: If a task's `depends_on` includes a task that is `failed` and will never be retried (either `attempts >= max_attempts` OR its `error_log` contains a `[DEPENDENCY]` entry), mark the blocked task as `failed` with `[DEPENDENCY] Blocked by failed task-XXX`. Repeat until no more tasks can be propagated.

Then pick the next task in this priority order:

1. Tasks with `status: "pending"` where ALL `depends_on` tasks are `completed` — sorted by `priority` (P0 > P1 > P2), then by `id` (lowest first)
2. Tasks with `status: "failed"` where `attempts < max_attempts` and ALL `depends_on` are `completed` — sorted by priority, then oldest failure first
3. If no eligible tasks remain → log final STATS → STOP

### Task Execution Cycle

For each task, execute this exact sequence:

1. **Claim** (atomic, under lock): Record `started_at_commit` = current HEAD hash. Set status to `in_progress`, set `claimed_by`, set `lease_expires_at`, log `Starting [<task-id>] <title> (base=<hash>)`. If the task is already claimed (`in_progress` with a valid lease), pick another eligible task and retry.
2. **Execute with checkpoints**: Perform the work. After each significant step, log:
   ```
   [timestamp] [SESSION-N] CHECKPOINT [task-id] step=M/N "description of what was done"
   ```
   Also append to the task's `checkpoints` array: `{ "step": M, "total": N, "description": "...", "timestamp": "ISO" }`. In concurrent mode, renew the lease at each checkpoint (push `lease_expires_at` forward).
3. **Validate**: Run the task's `validation.command` with a timeout wrapper (prefer `timeout`; on macOS use `gtimeout` from coreutils). If `validation.command` is empty/null, log `ERROR [<task-id>] [CONFIG] Missing validation.command` and STOP — do not declare completion without an objective check. Before running, verify the command exists (e.g., `command -v <binary>`) — if missing, treat as `ENV_SETUP` error.
   - Command exits 0 → PASS
   - Command exits non-zero → FAIL
   - Command exceeds timeout → TIMEOUT
4. **Record outcome**:
   - **Success**: status=`completed`, set `completed_at`, log `Completed [<task-id>] (commit <hash>)`, git commit
   - **Failure**: increment `attempts`, append error to `error_log`. Verify `started_at_commit` exists via `git cat-file -t <hash>` — if missing, mark failed at max_attempts. Otherwise execute `git reset --hard <started_at_commit>` and `git clean -fd` to rollback ALL commits and remove untracked files. Execute `on_failure.cleanup` if defined. Log `ERROR [<task-id>] [<category>] <message>`. Set status=`failed` (Task Selection Algorithm pass 2 handles retries when attempts < max_attempts)
5. **Track**: Increment per-session task counter. If `max_tasks_per_session` reached, log STATS and STOP.
6. **Continue**: Immediately pick next task (zero idle time)

### Stopping Conditions

- All tasks `completed`
- All remaining tasks `failed` at max_attempts or blocked by failed dependencies
- `session_config.max_tasks_per_session` reached for this session
- `session_config.max_sessions` reached across all sessions
- User interrupts

## Context Window Recovery Protocol

When a new session starts and finds a task with `status: "in_progress"`:

- Exclusive mode: treat this as an interrupted previous session and run the Recovery Protocol below.
- Concurrent mode: only recover a task if either (a) `claimed_by` matches this worker, or (b) `lease_expires_at` is in the past (stale lease). Otherwise, treat it as owned by another worker and do not modify it.

1. **Check git state**:
   ```bash
   git diff --stat          # Uncommitted changes?
   git log --oneline -5     # Recent commits since task started?
   git stash list           # Any stashed work?
   ```
2. **Check checkpoints**: Read the task's `checkpoints` array to determine last completed step
3. **Decision matrix** (verify recent commits belong to this task by checking commit messages for the task-id):

| Uncommitted? | Recent task commits? | Checkpoints? | Action |
|---|---|---|---|
| No | No | None | Mark `failed` with `[SESSION_TIMEOUT] No progress detected`, increment attempts |
| No | No | Some | Verify file state matches checkpoint claims. If files reflect checkpoint progress, resume from last step. If not, mark `failed` — work was lost |
| No | Yes | Any | Run `validation.command`. If passes → mark `completed`. If fails → `git reset --hard <started_at_commit>`, mark `failed` |
| Yes | No | Any | Run validation WITH uncommitted changes present. If passes → commit, mark `completed`. If fails → `git reset --hard <started_at_commit>` + `git clean -fd`, mark `failed` |
| Yes | Yes | Any | Commit uncommitted changes, run `validation.command`. If passes → mark `completed`. If fails → `git reset --hard <started_at_commit>` + `git clean -fd`, mark `failed` |

4. **Log recovery**: `[timestamp] [SESSION-N] RECOVERY [task-id] action="<action taken>" reason="<reason>"`

## Error Handling & Recovery Strategies

Each error category has a default recovery strategy:

| Category | Default Recovery | Agent Action |
|----------|-----------------|--------------|
| `ENV_SETUP` | Re-run init, then STOP if still failing | Run `harness-init.sh` again immediately. If fails twice, log and stop — environment is broken |
| `CONFIG` | STOP (requires human fix) | Log the config error precisely (file + field), then STOP. Do not guess or auto-mutate task metadata |
| `TASK_EXEC` | Rollback via `git reset --hard <started_at_commit>`, retry | Verify `started_at_commit` exists (`git cat-file -t <hash>`). If missing, mark failed at max_attempts. Otherwise reset, run `on_failure.cleanup` if defined, retry if attempts < max_attempts |
| `TEST_FAIL` | Rollback via `git reset --hard <started_at_commit>`, retry | Reset to `started_at_commit`, analyze test output to identify fix, retry with targeted changes |
| `TIMEOUT` | Kill process, execute cleanup, retry | Wrap validation with `timeout <seconds> <command>`. On timeout, run `on_failure.cleanup`, retry (consider splitting task if repeated) |
| `DEPENDENCY` | Skip task, mark blocked | Log which dependency failed, mark task as `failed` with dependency reason |
| `SESSION_TIMEOUT` | Use Context Window Recovery Protocol | New session assesses partial progress via Recovery Protocol — may result in completion or failure depending on validation |

**JSON corruption**: If `harness-tasks.json` cannot be parsed, check for `harness-tasks.json.bak` (written before each modification). If backup exists and is valid, restore from it. If no valid backup, log `ERROR [ENV_SETUP] harness-tasks.json corrupted and unrecoverable` and STOP — task metadata (validation commands, dependencies, cleanup) cannot be reconstructed from logs alone.

**Backup protocol**: Before every write to `harness-tasks.json`, copy the current file to `harness-tasks.json.bak`. Write updates atomically: write JSON to `harness-tasks.json.tmp` then `mv` it into place (readers should never see a partial file).

## Environment Initialization

If `harness-init.sh` exists in the project root, run it at every session start. The script must be idempotent.

Example `harness-init.sh`:
```bash
#!/bin/bash
set -e
npm install 2>/dev/null || pip install -r requirements.txt 2>/dev/null || true
curl -sf http://localhost:5432 >/dev/null 2>&1 || echo "WARN: DB not reachable"
npm test -- --bail --silent 2>/dev/null || echo "WARN: Smoke test failed"
echo "Environment health check complete"
```

## Standardized Log Format

All log entries use grep-friendly format on a single line:

```
[ISO-timestamp] [SESSION-N] <TYPE> [task-id]? [category]? message
```

`[task-id]` and `[category]` are included when applicable (task-scoped entries). Session-level entries (`INIT`, `LOCK`, `STATS`) omit them.

Types: `INIT`, `Starting`, `Completed`, `ERROR`, `CHECKPOINT`, `ROLLBACK`, `RECOVERY`, `STATS`, `LOCK`, `WARN`

Error categories: `ENV_SETUP`, `CONFIG`, `TASK_EXEC`, `TEST_FAIL`, `TIMEOUT`, `DEPENDENCY`, `SESSION_TIMEOUT`

Filtering:
```bash
grep "ERROR" harness-progress.txt                    # All errors
grep "ERROR" harness-progress.txt | grep "TASK_EXEC" # Execution errors only
grep "SESSION-3" harness-progress.txt                # All session 3 activity
grep "STATS" harness-progress.txt                    # All session summaries
grep "CHECKPOINT" harness-progress.txt               # All checkpoints
grep "RECOVERY" harness-progress.txt                 # All recovery actions
```

## Session Statistics

At session end, update `harness-tasks.json`: set `last_session` to current timestamp. (Do NOT increment `session_count` here — it is incremented at Session Start.) Then append:

```
[timestamp] [SESSION-N] STATS tasks_total=10 completed=7 failed=1 pending=2 blocked=0 attempts_total=12 checkpoints=23
```

`blocked` is computed at stats time: count of pending tasks whose `depends_on` includes a permanently failed task. It is not a stored status value.

## Init Command (`/harness init`)

1. Create `harness-progress.txt` with initialization entry
2. Create `harness-tasks.json` with empty task list and default `session_config`
3. Optionally create `harness-init.sh` template (chmod +x)
4. Ask user: add harness files to `.gitignore`?

## Status Command (`/harness status`)

Read `harness-tasks.json` and `harness-progress.txt`, then display:

1. Task summary: count by status (completed, failed, pending, blocked). `blocked` = pending tasks whose `depends_on` includes a permanently failed task (computed, not a stored status).
2. Per-task one-liner: `[status] task-id: title (attempts/max_attempts)`
3. Last 5 lines from `harness-progress.txt`
4. Session count and last session timestamp

Does NOT acquire the lock (read-only operation).

## Add Command (`/harness add`)

Append a new task to `harness-tasks.json` with auto-incremented id (`task-NNN`), status `pending`, default `max_attempts: 3`, empty `depends_on`, and no validation command (required before the task can be completed). Prompt user for optional fields: `priority`, `depends_on`, `validation.command`, `timeout_seconds`. Requires lock acquisition (modifies JSON).

## Tool Dependencies

Requires: Bash, file read/write, git. All harness operations must be executed from the project root directory.
Does NOT require: specific MCP servers, programming languages, or test frameworks.

Concurrent mode requires isolated working directories (`git worktree` or separate clones). Do not run concurrent workers in the same working tree.
