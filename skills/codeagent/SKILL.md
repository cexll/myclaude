---
name: codeagent
description: Execute codeagent-wrapper for multi-backend AI code tasks. Supports Codex, Claude, and Gemini backends with file references (@syntax) and structured output.
---

# Codeagent Wrapper Integration

## Overview

Execute codeagent-wrapper commands with pluggable AI backends (Codex, Claude, Gemini). Supports file references via `@` syntax and parallel task execution.

## When to Use

- Complex code analysis requiring deep understanding
- Large-scale refactoring across multiple files
- Automated code generation with backend selection

## Usage

**HEREDOC syntax** (recommended):
```bash
codeagent-wrapper - [working_dir] <<'EOF'
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
codeagent-wrapper "simple task" [working_dir]
codeagent-wrapper --backend gemini "simple task"
```

## Backends

| Backend | Command | Description |
|---------|---------|-------------|
| codex | `--backend codex` | OpenAI Codex (default) |
| claude | `--backend claude` | Anthropic Claude |
| gemini | `--backend gemini` | Google Gemini |

## Parameters

- `task` (required): Task description, supports `@file` references
- `working_dir` (optional): Working directory (default: current)
- `--backend` (optional): Select AI backend (codex/claude/gemini)

## Return Format

```
Agent response text here...

---
SESSION_ID: 019a7247-ac9d-71f3-89e2-a823dbd8fd14
```

## Resume Session 

```bash
codeagent-wrapper resume <session_id> - <<'EOF'
<follow-up task>
EOF
```

## Parallel Execution

```bash
codeagent-wrapper --parallel <<'EOF'
---TASK---
id: task1
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

## Environment Variables

- `CODEX_TIMEOUT`: Override timeout in milliseconds (default: 7200000)

## Invocation Pattern

```
Bash tool parameters:
- command: codeagent-wrapper --backend <backend> - [working_dir] <<'EOF'
  <task content>
  EOF
- timeout: 7200000
- description: <brief description>
```
