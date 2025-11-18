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
- Tasks requiring specialized reasoning models (gpt-5.1, gpt-5.1-codex)

## Usage

**推荐方式**（使用 uv run，自动管理 Python 环境）：
```bash
uv run ~/.claude/skills/codex/scripts/codex.py "<task>" [model] [working_dir]
```

**备选方式**（直接执行或使用 Python）：
```bash
~/.claude/skills/codex/scripts/codex.py "<task>" [model] [working_dir]
# 或
python3 ~/.claude/skills/codex/scripts/codex.py "<task>" [model] [working_dir]
```

恢复会话:
```bash
uv run ~/.claude/skills/codex/scripts/codex.py resume <session_id> "<task>" [model] [working_dir]
```

## Timeout Control

- **Built-in**: Script enforces 2-hour timeout by default
- **Override**: Set `CODEX_TIMEOUT` environment variable (in milliseconds, e.g., `CODEX_TIMEOUT=3600000` for 1 hour)
- **Behavior**: On timeout, sends SIGTERM, then SIGKILL after 5s if process doesn't exit
- **Exit code**: Returns 124 on timeout (consistent with GNU timeout)
- **Bash tool**: Always set `timeout: 7200000` parameter for double protection

### Parameters

- `task` (required): Task description, supports `@file` references
- `model` (optional): Model to use (default: gpt-5.1-codex)
  - `gpt-5.1-codex`: Default, optimized for code
  - `gpt-5.1`: Fast general purpose
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

### Invocation Pattern

When calling via Bash tool, always include the timeout parameter:
```
Bash tool parameters:
- command: uv run ~/.claude/skills/codex/scripts/codex.py "<task>" [model] [working_dir]
- timeout: 7200000
- description: <brief description of the task>
```

Alternatives:
```
# Direct execution (simplest)
- command: ~/.claude/skills/codex/scripts/codex.py "<task>" [model] [working_dir]

# Using python3
- command: python3 ~/.claude/skills/codex/scripts/codex.py "<task>" [model] [working_dir]
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

**Refactoring with specific model:**
```bash
uv run ~/.claude/skills/codex/scripts/codex.py "refactor @src/utils for performance" "gpt-5.1-codex"
# timeout: 7200000
```

**Multi-file analysis:**
```bash
uv run ~/.claude/skills/codex/scripts/codex.py "analyze @. and find security issues" "gpt-5.1-codex" "/path/to/project"
# timeout: 7200000
```

**Resume previous session:**
```bash
# First session
uv run ~/.claude/skills/codex/scripts/codex.py "add comments to @utils.js" "gpt-5.1-codex"
# Output includes: SESSION_ID: 019a7247-ac9d-71f3-89e2-a823dbd8fd14

# Continue the conversation
uv run ~/.claude/skills/codex/scripts/codex.py resume 019a7247-ac9d-71f3-89e2-a823dbd8fd14 "now add type hints"
# timeout: 7200000
```

**Using python3 directly (alternative):**
```bash
python3 ~/.claude/skills/codex/scripts/codex.py "your task here"
```

## Notes

- **Recommended**: Use `uv run` for automatic Python environment management (requires uv installed)
- **Alternative**: Direct execution `./codex.py` (uses system Python via shebang)
- Python implementation using standard library (zero dependencies)
- Cross-platform compatible (Windows/macOS/Linux)
- PEP 723 compliant (inline script metadata)
- Runs with `--dangerously-bypass-approvals-and-sandbox` for automation (new sessions only)
- Uses `--skip-git-repo-check` to work in any directory
- Streams progress, returns only final agent message
- Every execution returns a session ID for resuming conversations
- Requires Codex CLI installed and authenticated
