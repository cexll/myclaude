---
name: codex
description: Execute Codex CLI for code analysis, refactoring, and automated code changes. Use when you need to delegate complex code tasks to Codex AI with file references (@syntax) and structured output.
---

# Codex CLI Integration

## Overview

Execute Codex CLI commands and parse structured JSON responses. Supports file references via `@` syntax, multiple models, and sandbox controls.

## When to Use

- Complex code analysis requiring deep understanding
- Large-scale refactoring across multiple files
- Automated code generation with safety controls

## Fallback Policy

Codex is the **primary execution method** for all code edits and tests. Direct execution is only permitted when:

1. Codex is unavailable (service down, network issues)
2. Codex fails **twice consecutively** on the same task

When falling back to direct execution:
- Log `CODEX_FALLBACK` with the reason
- Retry Codex on the next task (don't permanently switch)
- Document the fallback in the final summary

## Usage

**Mandatory**: Run every automated invocation through the Bash tool in the foreground with **HEREDOC syntax** to avoid shell quoting issues, keeping the `timeout` parameter fixed at `7200000` milliseconds (do not change it or use any other entry point).

```bash
codex-wrapper - [working_dir] <<'EOF'
<task content here>
EOF
```

**Why HEREDOC?** Tasks often contain code blocks, nested quotes, shell metacharacters (`$`, `` ` ``, `\`), and multiline text. HEREDOC (Here Document) syntax passes these safely without shell interpretation, eliminating quote-escaping nightmares.

**Foreground only (no background/BashOutput)**: Never set `background: true`, never accept Claude's "Running in the background" mode, and avoid `BashOutput` streaming loops. Keep a single foreground Bash call per Codex task; if work might be long, split it into smaller foreground runs instead of offloading to background execution.

**Simple tasks** (backward compatibility):
For simple single-line tasks without special characters, you can still use direct quoting:
```bash
codex-wrapper "simple task here" [working_dir]
```

**Resume a session with HEREDOC:**
```bash
codex-wrapper resume <session_id> - [working_dir] <<'EOF'
<task content>
EOF
```

**Cross-platform notes:**
- **Bash/Zsh**: Use `<<'EOF'` (single quotes prevent variable expansion)
- **PowerShell 5.1+**: Use `@'` and `'@` (here-string syntax)
  ```powershell
  codex-wrapper - @'
  task content
  '@
  ```

## Environment Variables

- **CODEX_TIMEOUT**: Override timeout in milliseconds (default: 7200000 = 2 hours)
  - Example: `export CODEX_TIMEOUT=3600000` for 1 hour

## Timeout Control

- **Built-in**: Binary enforces 2-hour timeout by default
- **Override**: Set `CODEX_TIMEOUT` environment variable (in milliseconds, e.g., `CODEX_TIMEOUT=3600000` for 1 hour)
- **Behavior**: On timeout, sends SIGTERM, then SIGKILL after 5s if process doesn't exit
- **Exit code**: Returns 124 on timeout (consistent with GNU timeout)
- **Bash tool**: Always set `timeout: 7200000` parameter for double protection

### Parameters

- `task` (required): Task description, supports `@file` references
- `working_dir` (optional): Working directory (default: current)

### Return Format

Extracts `agent_message` from Codex JSON stream and appends session ID:
```
Agent response text here...

---
SESSION_ID: 019a7247-ac9d-71f3-89e2-a823dbd8fd14
```

Error format (stderr):
```
ERROR: Error message
```

Return only the final agent message and session ID—do not paste raw `BashOutput` logs or background-task chatter into the conversation.

### Invocation Pattern

All automated executions must use HEREDOC syntax through the Bash tool in the foreground, with `timeout` fixed at `7200000` (non-negotiable):

```
Bash tool parameters:
- command: codex-wrapper - [working_dir] <<'EOF'
  <task content>
  EOF
- timeout: 7200000
- description: <brief description of the task>
```

Run every call in the foreground—never append `&` to background it—so logs and errors stay visible for timely interruption or diagnosis.

**Important:** Use HEREDOC (`<<'EOF'`) for all but the simplest tasks. This prevents shell interpretation of quotes, variables, and special characters.

### Examples

**Basic code analysis:**
```bash
# Recommended: with HEREDOC (handles any special characters)
codex-wrapper - <<'EOF'
explain @src/main.ts
EOF
# timeout: 7200000

# Alternative: simple direct quoting (if task is simple)
codex-wrapper "explain @src/main.ts"
```

**Refactoring with multiline instructions:**
```bash
codex-wrapper - <<'EOF'
refactor @src/utils for performance:
- Extract duplicate code into helpers
- Use memoization for expensive calculations
- Add inline comments for non-obvious logic
EOF
# timeout: 7200000
```

**Multi-file analysis:**
```bash
codex-wrapper - "/path/to/project" <<'EOF'
analyze @. and find security issues:
1. Check for SQL injection vulnerabilities
2. Identify XSS risks in templates
3. Review authentication/authorization logic
4. Flag hardcoded credentials or secrets
EOF
# timeout: 7200000
```

**Resume previous session:**
```bash
# First session
codex-wrapper - <<'EOF'
add comments to @utils.js explaining the caching logic
EOF
# Output includes: SESSION_ID: 019a7247-ac9d-71f3-89e2-a823dbd8fd14

# Continue the conversation with more context
codex-wrapper resume 019a7247-ac9d-71f3-89e2-a823dbd8fd14 - <<'EOF'
now add TypeScript type hints and handle edge cases where cache is null
EOF
# timeout: 7200000
```

**Task with code snippets and special characters:**
```bash
codex-wrapper - <<'EOF'
Fix the bug in @app.js where the regex /\d+/ doesn't match "123"
The current code is:
  const re = /\d+/;
  if (re.test(input)) { ... }
Add proper escaping and handle $variables correctly.
EOF
```

### Parallel Execution

> Important:
> - `--parallel` only reads task definitions from stdin.
> - It does not accept extra command-line arguments (no inline `workdir`, `task`, or other params).
> - Put all task metadata and content in stdin; nothing belongs after `--parallel` on the command line.

**Correct vs Incorrect Usage**

**Correct:**
```bash
# Option 1: file redirection
codex-wrapper --parallel < tasks.txt

# Option 2: heredoc (recommended for multiple tasks)
codex-wrapper --parallel <<'EOF'
---TASK---
id: task1
workdir: /path/to/dir
---CONTENT---
task content
EOF

# Option 3: pipe
echo "---TASK---..." | codex-wrapper --parallel
```

**Incorrect (will trigger shell parsing errors):**
```bash
# Bad: no extra args allowed after --parallel
codex-wrapper --parallel - /path/to/dir <<'EOF'
...
EOF

# Bad: --parallel does not take a task argument
codex-wrapper --parallel "task description"

# Bad: workdir must live inside the task config
codex-wrapper --parallel /path/to/dir < tasks.txt
```

For multiple independent or dependent tasks, use `--parallel` mode with delimiter format:

**Typical Workflow (analyze → implement → test, chained in a single parallel call)**:
```bash
codex-wrapper --parallel <<'EOF'
---TASK---
id: analyze_1732876800
workdir: /home/user/project
---CONTENT---
analyze @spec.md and summarize API and UI requirements
---TASK---
id: implement_1732876801
workdir: /home/user/project
dependencies: analyze_1732876800
---CONTENT---
implement features from analyze_1732876800 summary in backend @services and frontend @ui
---TASK---
id: test_1732876802
workdir: /home/user/project
dependencies: implement_1732876801
---CONTENT---
add and run regression tests covering the new endpoints and UI flows
EOF
```
A single `codex-wrapper --parallel` call schedules all three stages concurrently, using `dependencies` to enforce sequential ordering without multiple invocations.

```bash
codex-wrapper --parallel <<'EOF'
---TASK---
id: backend_1732876800
workdir: /home/user/project/backend
---CONTENT---
implement /api/orders endpoints with validation and pagination
---TASK---
id: frontend_1732876801
workdir: /home/user/project/frontend
---CONTENT---
build Orders page consuming /api/orders with loading/error states
---TASK---
id: tests_1732876802
workdir: /home/user/project/tests
dependencies: backend_1732876800, frontend_1732876801
---CONTENT---
run API contract tests and UI smoke tests (waits for backend+frontend)
EOF
```

**Delimiter Format**:
- `---TASK---`: Starts a new task block
- `id: <task-id>`: Required, unique task identifier
  - Best practice: use `<feature>_<timestamp>` format (e.g., `auth_1732876800`, `api_test_1732876801`)
  - Ensures uniqueness across runs and makes tasks traceable
- `workdir: <path>`: Optional, working directory (default: `.`)
  - Best practice: use absolute paths (e.g., `/home/user/project/backend`)
  - Avoids ambiguity and ensures consistent behavior across environments
  - Must be specified inside each task block; do not pass `workdir` as a CLI argument to `--parallel`
  - Each task can set its own `workdir` when different directories are needed
- `dependencies: <id1>, <id2>`: Optional, comma-separated task IDs
- `session_id: <uuid>`: Optional, resume a previous session
- `---CONTENT---`: Separates metadata from task content
- Task content: Any text, code, special characters (no escaping needed)

**Dependencies Best Practices**

- Avoid multiple invocations: Place "analyze then implement" in a single `codex-wrapper --parallel` call, chaining them via `dependencies`, rather than running analysis first and then launching implementation separately.
- Naming convention: Use `<action>_<timestamp>` format (e.g., `analyze_1732876800`, `implement_1732876801`), where action names map to features/stages and timestamps ensure uniqueness and sortability.
- Dependency chain design: Keep chains short; only add dependencies for tasks that truly require ordering, let others run in parallel, avoiding over-serialization that reduces throughput.

**Resume Failed Tasks**:
```bash
# Use session_id from previous output to resume
codex-wrapper --parallel <<'EOF'
---TASK---
id: T2
session_id: 019xxx-previous-session-id
---CONTENT---
fix the previous error and retry
EOF
```

**Output**: Human-readable text format
```
=== Parallel Execution Summary ===
Total: 3 | Success: 2 | Failed: 1

--- Task: T1 ---
Status: SUCCESS
Session: 019xxx

Task output message...

--- Task: T2 ---
Status: FAILED (exit code 1)
Error: some error message
```

**Features**:
- Automatic topological sorting based on dependencies
- Unlimited concurrency for independent tasks
- Error isolation (failed tasks don't stop others)
- Dependency blocking (dependent tasks skip if parent fails)

## Notes

- **Binary distribution**: Single Go binary, zero dependencies
- **Installation**: Download from GitHub Releases or use install.sh
- **Cross-platform compatible**: Linux (amd64/arm64), macOS (amd64/arm64)
- All automated runs must use the Bash tool with the fixed timeout to provide dual timeout protection and unified logging/exit semantics
for automation (new sessions only)
- Uses `--skip-git-repo-check` to work in any directory
- Streams progress, returns only final agent message
- Every execution returns a session ID for resuming conversations
- Requires Codex CLI installed and authenticated
