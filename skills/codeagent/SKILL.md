---
name: codeagent
description: Execute codeagent-wrapper for multi-backend AI code tasks. Supports Codex, Claude, Gemini, and OpenCode backends with agent presets, skill injection, file references (@syntax), worktree isolation, parallel execution, and structured output.
---

# Codeagent Wrapper Integration

## Overview

Execute `codeagent-wrapper` commands with pluggable AI backends (Codex, Claude, Gemini, OpenCode), agent presets, auto-detected skill injection, and parallel task orchestration. Default to background execution, and prefer `--parallel` whenever work can be split into independent tasks.

## When to Use

- Complex code analysis requiring deep understanding
- Large-scale refactoring across multiple files
- Multi-agent orchestration (explore → design → implement → review)
- Automated code generation with backend/agent selection
- Parallel task execution with dependency management

## Quick Reference

```
codeagent-wrapper [flags] <task|-> [workdir]
codeagent-wrapper [flags] resume <session_id> <task|-> [workdir]
codeagent-wrapper --parallel [flags] < tasks_config
```

## CLI Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--backend <name>` | Backend: codex, claude, gemini, opencode | codex |
| `--agent <name>` | Agent preset (from models.json or agents/ dir) | none |
| `--model <name>` | Model override for any backend | backend default |
| `--skills <names>` | Comma-separated skill names to inject | auto-detected |
| `--reasoning-effort <level>` | Reasoning level: low, medium, high | backend default |
| `--prompt-file <path>` | Custom prompt file (restricted to ~/.claude or ~/.codeagent/agents/) | none |
| `--output <path>` | Write structured JSON output to file | none |
| `--worktree` | Execute in isolated git worktree (branch: do/{task_id}) | false |
| `--skip-permissions` | Skip Claude backend permission prompts | false |
| `--parallel` | Enable parallel task execution from stdin | false |
| `--full-output` | Include full messages in parallel output (default: summary) | false |
| `--config <path>` | Config file path | ~/.codeagent/config.* |
| `--cleanup` | Clean up old logs and exit | — |
| `-v`, `--version` | Print version and exit | — |

## Backends

| Backend | Flag | Best For |
|---------|------|----------|
| **Codex** | `--backend codex` (default) | Deep code analysis, complex logic, algorithm optimization, large-scale refactoring |
| **Claude** | `--backend claude` | Documentation, prompt engineering, clear-requirement features |
| **Gemini** | `--backend gemini` | UI/UX prototyping, design system implementation |
| **OpenCode** | `--backend opencode` | Lightweight tasks, minimal feature set |

## Agent Presets

Agent presets bundle backend, model, prompt, and tool control into a reusable name. Use `--agent <name>` to select.

**Sources (checked in order):**
1. `~/.codeagent/models.json` → `agents.<name>` object
2. `~/.codeagent/agents/<name>.md` → markdown file becomes the prompt

**Agent config fields** (in models.json):
```json
{
  "agents": {
    "develop": {
      "backend": "codex",
      "model": "gpt-4.1",
      "prompt_file": "~/.codeagent/prompts/develop.md",
      "reasoning": "high",
      "yolo": true,
      "allowed_tools": ["Read", "Write", "Bash"],
      "disallowed_tools": ["WebFetch"]
    }
  }
}
```

**Common agent presets:**

| Agent | Purpose | Read-Only |
|-------|---------|-----------|
| `code-explorer` | Trace code, map architecture, find patterns | Yes |
| `code-architect` | Design approaches, file plans, build sequences | Yes |
| `code-reviewer` | Review for bugs, simplicity, conventions | Yes |
| `develop` | Implement code, run tests, make changes | No |

## Skill Injection

### Auto-Detection

When `--skills` is not specified, skills are auto-detected from the working directory:

| Detected Files | Injected Skills |
|---|---|
| `go.mod` / `go.sum` | `golang-base-practices` |
| `Cargo.toml` | `rust-best-practices` |
| `pyproject.toml` / `setup.py` / `requirements.txt` | `python-best-practices` |
| `package.json` | `vercel-react-best-practices`, `frontend-design` |
| `vue.config.js` / `vite.config.ts` / `nuxt.config.ts` | `vue-web-app` |

### Manual Override

```bash
codeagent-wrapper --agent develop --skills golang-base-practices,frontend-design - . <<'EOF'
Implement full-stack feature...
EOF
```

Skills are loaded from `~/.claude/skills/{name}/SKILL.md`, stripped of YAML frontmatter, and injected into the task prompt.

## Usage Patterns

### Single Task (HEREDOC recommended)

```bash
codeagent-wrapper --backend codex - [workdir] <<'EOF'
<task content here>
EOF
```

Default execution mode is background. Run in foreground only when the next step requires the full response immediately.

When running in foreground, `codeagent-wrapper` emits liveness frames on stdout before the final answer:

```text
[codeagent-progress] status=started ...
[codeagent-progress] status=streaming ...
[codeagent-progress] status=running ...
[codeagent-progress] status=completed ...
```

Treat these lines as progress only. Do not conclude "no data returned" and do not start doing the task yourself while progress frames are still arriving.

### With Agent Preset

```bash
codeagent-wrapper --agent develop --skills golang-base-practices - . <<'EOF'
Implement the authentication middleware following existing patterns.
EOF
```

### Simple Task (short prompts only)

```bash
codeagent-wrapper --backend codex "simple task description" [workdir]
```

**Auto-stdin detection**: When task length exceeds 800 characters or contains special characters (`\n`, `\`, `"`, `'`, `` ` ``, `$`), stdin mode is used automatically. Use `-` to force stdin mode explicitly.

### Resume Session

```bash
codeagent-wrapper --backend codex resume <session_id> - <<'EOF'
<follow-up task>
EOF

# Or with agent preset
codeagent-wrapper --agent develop resume <session_id> - <<'EOF'
<follow-up task>
EOF
```

### Worktree Isolation

Execute in an isolated git worktree to keep changes separate from the main branch:

```bash
codeagent-wrapper --agent develop --worktree - . <<'EOF'
Implement feature in isolation...
EOF
```

**Rules:**
- Read-only agents (code-explorer, code-architect, code-reviewer) do NOT need worktree
- Only `develop` agent needs worktree when making changes

## Parallel Execution

Use `--parallel` by default for multi-step or multi-agent work. Fall back to single-task mode only when the work is truly linear or the next step depends on the full output of the current task.

### Task Config Format

```bash
codeagent-wrapper --parallel <<'EOF'
---TASK---
id: <unique_id>
agent: <agent_name>
workdir: <path>
backend: <name>
model: <model_name>
reasoning_effort: <low|medium|high>
skills: <skill1>, <skill2>
dependencies: <id1>, <id2>
session_id: <id>
skip_permissions: true|false
worktree: true|false
---CONTENT---
<task content>
EOF
```

**Task header fields** (all optional except `id`):

| Field | Description |
|-------|-------------|
| `id` | Unique task identifier (required) |
| `agent` | Agent preset name |
| `workdir` | Working directory |
| `backend` | Override global backend |
| `model` | Override model |
| `reasoning_effort` | Reasoning level |
| `skills` | Comma-separated skill names |
| `dependencies` | Comma-separated task IDs that must complete first |
| `session_id` | Resume a previous session |
| `skip_permissions` | Skip permission prompts (Claude backend) |
| `worktree` | Execute in git worktree |

### Multi-Agent Orchestration Example

```bash
codeagent-wrapper --parallel <<'EOF'
---TASK---
id: p1_architecture
agent: code-explorer
workdir: .
---CONTENT---
Map architecture for the authentication subsystem. Return: module map + key files with line numbers.

---TASK---
id: p1_conventions
agent: code-explorer
workdir: .
---CONTENT---
Identify testing patterns, conventions, config. Return: test commands + file locations.

---TASK---
id: p2_design
agent: code-architect
workdir: .
dependencies: p1_architecture, p1_conventions
---CONTENT---
Design minimal-change implementation plan based on architecture analysis.

---TASK---
id: p3_backend
agent: develop
workdir: .
skills: golang-base-practices
dependencies: p2_design
---CONTENT---
Implement backend changes following the design plan.

---TASK---
id: p3_frontend
agent: develop
workdir: .
skills: vercel-react-best-practices,frontend-design
dependencies: p2_design
---CONTENT---
Implement frontend changes following the design plan.

---TASK---
id: p4_review
agent: code-reviewer
workdir: .
dependencies: p3_backend, p3_frontend
---CONTENT---
Review all changes for correctness, edge cases, and KISS compliance.
Classify each issue as BLOCKING or MINOR.
EOF
```

### Dependency Resolution

- Tasks are topologically sorted (Kahn's algorithm)
- Circular dependencies are detected and reported
- Failed parent tasks cause dependent tasks to be skipped
- Independent tasks at the same level run concurrently

### Output Modes

**Summary (default)**: Structured report per task with extracted fields:
```
=== Execution Report ===
3 tasks | 2 passed | 1 failed

### task_id PASS 92%
Did: Brief description of work done
Files: file1.ts, file2.ts
Tests: 12 passed
Log: /tmp/codeagent-xxx.log

### task_id FAIL
Exit code: 1
Error: Assertion failed
Detail: Expected status 200 but got 401
Log: /tmp/codeagent-zzz.log
```

**Full output** (`--full-output`): Complete task messages included. Use only for debugging specific failures.

### Structured JSON Output

```bash
codeagent-wrapper --parallel --output results.json <<'EOF'
...
EOF
```

Produces:
```json
{
  "results": [
    {
      "task_id": "task_1",
      "exit_code": 0,
      "message": "...",
      "session_id": "...",
      "coverage": "92%",
      "files_changed": ["file.ts"],
      "tests_passed": 12,
      "log_path": "/tmp/..."
    }
  ],
  "summary": { "total": 3, "success": 2, "failed": 1 }
}
```

## Return Format

Single task output:
```
Agent response text here...

---
SESSION_ID: 019a7247-ac9d-71f3-89e2-a823dbd8fd14
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `CODEAGENT_SKIP_PERMISSIONS` | Skip Claude backend permission prompts (`true`/`false`) | true |
| `CODEX_BYPASS_SANDBOX` | Control Codex sandbox bypass (`true`/`false`) | true |
| `CODEAGENT_MAX_PARALLEL_WORKERS` | Max concurrent parallel workers (0=unlimited, max 100) | 0 |
| `CODEAGENT_TMPDIR` | Custom temp directory for executable scripts | system temp |
| `CODEAGENT_ASCII_MODE` | Use ASCII symbols (PASS/WARN/FAIL) instead of Unicode | false |

**Config file**: Supports `~/.codeagent/config.(yaml|yml|json|toml)` with the same keys as CLI flags (kebab-case). Env vars use `CODEAGENT_` prefix with underscores.

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | General error (missing args, failed task) |
| `127` | Backend command not found |
| `130` | Interrupted (Ctrl+C) |

## Invocation Pattern

**Single Task**:
```
Bash tool parameters:
- command: codeagent-wrapper --agent <agent> --skills <skills> - [workdir] <<'EOF'
  <task content>
  EOF
- background: true
- description: <brief description>
```

**Parallel Tasks**:
```
Bash tool parameters:
- command: codeagent-wrapper --parallel <<'EOF'
  ---TASK---
  id: task_id
  agent: <agent>
  workdir: /path
  skills: <skill1>, <skill2>
  dependencies: dep1, dep2
  ---CONTENT---
  task content
  EOF
- background: true
- description: <brief description>
```

## Critical Rules

**NEVER kill codeagent processes.** Long-running tasks are normal. Default to background execution and inspect progress instead of blocking on the command.

1. **Check task status via log file**:
   ```bash
   tail -f /tmp/claude/<workdir>/tasks/<task_id>.output
   ```

2. **Prefer wrapper progress frames over ad-hoc log scraping**:
   If stdout is still emitting `[codeagent-progress] ...`, the task is alive and has not stalled.

3. **Check process without killing**:
   ```bash
   ps aux | grep codeagent-wrapper | grep -v grep
   ```

**Why:** Killing wastes API costs and loses progress.

## Tool Control (Claude Backend)

When using Claude backend with agent presets, control available tools:

```json
{
  "agents": {
    "safe-develop": {
      "backend": "claude",
      "allowed_tools": ["Read", "Write", "Bash", "Grep", "Glob"],
      "disallowed_tools": ["WebFetch", "WebSearch"]
    }
  }
}
```

Passed as `--allowedTools` and `--disallowedTools` to Claude CLI. Explicit enumeration only (no wildcards).
