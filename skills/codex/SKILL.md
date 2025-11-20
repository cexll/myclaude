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

## Usage

**Mandatory**: Run every automated invocation through the Bash tool in the foreground with the command below, keeping the `timeout` parameter fixed at `7200000` milliseconds (do not change it or use any other entry point).
```bash
uv run ~/.claude/skills/codex/scripts/codex.py "<task>" [working_dir]
```

**Foreground only (no background/BashOutput)**: Never set `background: true`, never accept Claude's “Running in the background” mode, and avoid `BashOutput` streaming loops. Keep a single foreground Bash call per Codex task; if work might be long, split it into smaller foreground runs instead of offloading to background execution.

**Optional methods** (direct execution or via Python):
```bash
~/.claude/skills/codex/scripts/codex.py "<task>" [working_dir]
# or
python3 ~/.claude/skills/codex/scripts/codex.py "<task>" [working_dir]
```

Resume a session:
```bash
uv run ~/.claude/skills/codex/scripts/codex.py resume <session_id> "<task>" [working_dir]
```

## Environment Variables
- **CODEX_TIMEOUT**: Override timeout in milliseconds (default: 7200000 = 2 hours)
  - Example: `export CODEX_TIMEOUT=3600000` for 1 hour

## Timeout Control

- **Built-in**: Script enforces 2-hour timeout by default
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

All automated executions may only invoke `uv run ~/.claude/skills/codex/scripts/codex.py "<task>" ...` through the Bash tool in the foreground, and the `timeout` must remain fixed at `7200000` (non-negotiable):
```
Bash tool parameters:
- command: uv run ~/.claude/skills/codex/scripts/codex.py "<task>" [working_dir]
- timeout: 7200000
- description: <brief description of the task>
```
Run every call in the foreground—never append `&` to background it—so logs and errors stay visible for timely interruption or diagnosis.

Alternatives:
```
# Direct execution (simplest)
- command: ~/.claude/skills/codex/scripts/codex.py "<task>" [working_dir]

# Using python3
- command: python3 ~/.claude/skills/codex/scripts/codex.py "<task>" [working_dir]
```

### Examples

**Basic code analysis:**
```bash
# Recommended: via uv run (auto-manages Python environment)
uv run ~/.claude/skills/codex/scripts/codex.py "explain @src/main.ts"
# timeout: 7200000

# Alternative: direct execution
~/.claude/skills/codex/scripts/codex.py "explain @src/main.ts"
```

**Refactoring with custom model (via environment variable):**
```bash
# Set model via environment variable
uv run ~/.claude/skills/codex/scripts/codex.py "refactor @src/utils for performance"
# timeout: 7200000
```

**Multi-file analysis:**
```bash
uv run ~/.claude/skills/codex/scripts/codex.py "analyze @. and find security issues" "/path/to/project"
# timeout: 7200000
```

**Resume previous session:**
```bash
# First session
uv run ~/.claude/skills/codex/scripts/codex.py "add comments to @utils.js"
# Output includes: SESSION_ID: 019a7247-ac9d-71f3-89e2-a823dbd8fd14

# Continue the conversation
uv run ~/.claude/skills/codex/scripts/codex.py resume 019a7247-ac9d-71f3-89e2-a823dbd8fd14 "now add type hints"
# timeout: 7200000
```

**Using python3 directly (alternative):**
```bash
python3 ~/.claude/skills/codex/scripts/codex.py "your task here"
```

### Large Task Protocol

- For every large task, first produce a canonical task list that enumerates the Task ID, description, file/directory scope, dependencies, test commands, and the expected Codex Bash invocation.
- Tasks without dependencies should be executed concurrently via multiple foreground Bash calls (you can keep separate terminal windows) and each run must log start/end times plus any shared resource usage.
- Reuse context aggressively (such as @spec.md or prior analysis output), and after concurrent execution finishes, reconcile against the task list to report which items completed and which slipped.

| ID | Description | Scope | Dependencies | Tests | Command |
| --- | --- | --- | --- | --- | --- |
| T1 | Review @spec.md to extract requirements | docs/, @spec.md | None | None | uv run ~/.claude/skills/codex/scripts/codex.py "analyze requirements @spec.md" |
| T2 | Implement the module and add test cases | src/module | T1 | npm test -- --runInBand | uv run ~/.claude/skills/codex/scripts/codex.py "implement and test @src/module" |

## Notes

- **Recommended**: Use `uv run` for automatic Python environment management (requires uv installed)
- **Alternative**: Direct execution `./codex.py` (uses system Python via shebang)
- Python implementation using standard library (zero dependencies)
- All automated runs must use the Bash tool with the fixed timeout to provide dual timeout protection and unified logging/exit semantics; any alternative approach is limited to manual foreground execution.
- Cross-platform compatible (Windows/macOS/Linux)
- PEP 723 compliant (inline script metadata)
- Runs with `--dangerously-bypass-approvals-and-sandbox` for automation (new sessions only)
- Uses `--skip-git-repo-check` to work in any directory
- Streams progress, returns only final agent message
- Every execution returns a session ID for resuming conversations
- Requires Codex CLI installed and authenticated
