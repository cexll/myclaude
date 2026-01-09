---
name: codeagent
description: Execute codeagent-wrapper for multi-backend AI code tasks. Supports Codex, Claude, and Gemini backends with file references (@syntax) and structured output.
---

# Codeagent Wrapper Integration

## Overview

Execute codeagent-wrapper commands with pluggable AI backends (Codex, Claude, Gemini). Supports file references via `@` syntax, parallel task execution with backend selection, and configurable security controls.

## When to Use

- Complex code analysis requiring deep understanding
- Large-scale refactoring across multiple files
- Automated code generation with backend selection

## Usage

**HEREDOC syntax** (recommended):
```bash
codeagent-wrapper --backend codex - [working_dir] <<'EOF'
<task content here>
EOF
```

**With backend selection**:
```bash
codeagent-wrapper --backend claude - <<'EOF'
<task content here>
EOF
```

**Simple tasks**:
```bash
codeagent-wrapper --backend codex "simple task" [working_dir]
codeagent-wrapper --backend gemini "simple task"
```

## Backends

| Backend | Command | Description | Best For |
|---------|---------|-------------|----------|
| codex | `--backend codex` | OpenAI Codex (default) | Code analysis, complex development |
| claude | `--backend claude` | Anthropic Claude | Simple tasks, documentation, prompts |
| gemini | `--backend gemini` | Google Gemini | UI/UX prototyping |

### Backend Selection Guide

**Codex** (default):
- Deep code understanding and complex logic implementation
- Large-scale refactoring with precise dependency tracking
- Algorithm optimization and performance tuning
- Example: "Analyze the call graph of @src/core and refactor the module dependency structure"

**Claude**:
- Quick feature implementation with clear requirements
- Technical documentation, API specs, README generation
- Professional prompt engineering (e.g., product requirements, design specs)
- Example: "Generate a comprehensive README for @package.json with installation, usage, and API docs"

**Gemini**:
- UI component scaffolding and layout prototyping
- Design system implementation with style consistency
- Interactive element generation with accessibility support
- Example: "Create a responsive dashboard layout with sidebar navigation and data visualization cards"

**Backend Switching**:
- Start with Codex for analysis, switch to Claude for documentation, then Gemini for UI implementation
- Use per-task backend selection in parallel mode to optimize for each task's strengths

## Parameters

- `task` (required): Task description, supports `@file` references
- `working_dir` (optional): Working directory (default: current)
- `--backend` (required): Select AI backend (codex/claude/gemini)
  - **Note**: Claude backend only adds `--dangerously-skip-permissions` when explicitly enabled

## Return Format

```
Agent response text here...

---
SESSION_ID: 019a7247-ac9d-71f3-89e2-a823dbd8fd14
```

## Resume Session

```bash
# Resume with codex backend
codeagent-wrapper --backend codex resume <session_id> - <<'EOF'
<follow-up task>
EOF

# Resume with specific backend
codeagent-wrapper --backend claude resume <session_id> - <<'EOF'
<follow-up task>
EOF
```

## Parallel Execution

**Default (summary mode - context-efficient):**
```bash
codeagent-wrapper --parallel <<'EOF'
---TASK---
id: task1
backend: codex
workdir: /path/to/dir
---CONTENT---
task content
---TASK---
id: task2
dependencies: task1
---CONTENT---
dependent task
EOF
```

**Full output mode (for debugging):**
```bash
codeagent-wrapper --parallel --full-output <<'EOF'
...
EOF
```

**Output Modes:**
- **Summary (default)**: Structured report with changes, output, verification, and review summary.
- **Full (`--full-output`)**: Complete task messages. Use only when debugging specific failures.

**With per-task backend**:
```bash
codeagent-wrapper --parallel <<'EOF'
---TASK---
id: task1
backend: codex
workdir: /path/to/dir
---CONTENT---
analyze code structure
---TASK---
id: task2
backend: claude
dependencies: task1
---CONTENT---
design architecture based on analysis
---TASK---
id: task3
backend: gemini
dependencies: task2
---CONTENT---
generate implementation code
EOF
```

**Concurrency Control**:
Set `CODEAGENT_MAX_PARALLEL_WORKERS` to limit concurrent tasks (default: unlimited).

## Environment Variables

- `CODEX_TIMEOUT`: Override timeout in milliseconds (default: 7200000 = 2 hours)
- `CODEAGENT_SKIP_PERMISSIONS`: Control Claude CLI permission checks
  - For **Claude** backend: Set to `true`/`1` to add `--dangerously-skip-permissions` (default: disabled)
  - For **Codex/Gemini** backends: Currently has no effect
- `CODEAGENT_MAX_PARALLEL_WORKERS`: Limit concurrent tasks in parallel mode (default: unlimited, recommended: 8)

## Invocation Pattern

**Single Task**:
```
Bash tool parameters:
- command: codeagent-wrapper --backend <backend> - [working_dir] <<'EOF'
  <task content>
  EOF
- timeout: 7200000
- description: <brief description>

Note: --backend is required (codex/claude/gemini)
```

**Parallel Tasks**:
```
Bash tool parameters:
- command: codeagent-wrapper --parallel --backend <backend> <<'EOF'
  ---TASK---
  id: task_id
  backend: <backend>  # Optional, overrides global
  workdir: /path
  dependencies: dep1, dep2
  ---CONTENT---
  task content
  EOF
- timeout: 7200000
- description: <brief description>

Note: Global --backend is required; per-task backend is optional
```

## Critical Rules

**NEVER kill codeagent processes.** Long-running tasks are normal. Instead:

1. **Check task status via log file**:
   ```bash
   # View real-time output
   tail -f /tmp/claude/<workdir>/tasks/<task_id>.output

   # Check if task is still running
   cat /tmp/claude/<workdir>/tasks/<task_id>.output | tail -50
   ```

2. **Wait with timeout**:
   ```bash
   # Use TaskOutput tool with block=true and timeout
   TaskOutput(task_id="<id>", block=true, timeout=300000)
   ```

3. **Check process without killing**:
   ```bash
   ps aux | grep codeagent-wrapper | grep -v grep
   ```

**Why:** codeagent tasks often take 2-10 minutes. Killing them wastes API costs and loses progress.

## Security Best Practices

- **Claude Backend**: Permission checks enabled by default
  - To skip checks: set `CODEAGENT_SKIP_PERMISSIONS=true` or pass `--skip-permissions`
- **Concurrency Limits**: Set `CODEAGENT_MAX_PARALLEL_WORKERS` in production to prevent resource exhaustion
- **Automation Context**: This wrapper is designed for AI-driven automation where permission prompts would block execution

## Recent Updates

- Multi-backend support for all modes (workdir, resume, parallel)
- Security controls with configurable permission checks
- Concurrency limits with worker pool and fail-fast cancellation
