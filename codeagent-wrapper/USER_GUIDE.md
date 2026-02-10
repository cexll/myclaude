# Codeagent-Wrapper User Guide

Multi-backend AI code execution wrapper supporting Codex, Claude, Gemini, and OpenCode.

## Overview

`codeagent-wrapper` is a Go-based CLI tool that provides a unified interface to multiple AI coding backends. It handles:
- Multi-backend execution (Codex, Claude, Gemini, OpenCode)
- JSON stream parsing and output formatting
- Session management and resumption
- Parallel task execution with dependency resolution
- Timeout handling and signal forwarding

## Installation

```bash
# Recommended: run the installer and select "codeagent-wrapper"
npx github:cexll/myclaude

# Manual build (optional; requires repo checkout)
cd codeagent-wrapper
go build -o ~/.claude/bin/codeagent-wrapper
```

## Quick Start

### Basic Usage

```bash
# Simple task (default: codex backend)
codeagent-wrapper "explain @src/main.go"

# With backend selection
codeagent-wrapper --backend claude "refactor @utils.ts"

# With HEREDOC (recommended for complex tasks)
codeagent-wrapper --backend gemini - <<'EOF'
Implement user authentication:
- JWT tokens
- Password hashing with bcrypt
- Session management
EOF
```

### CLI Flags

| Flag | Description |
|------|-------------|
| `--backend <name>` | Select backend (codex/claude/gemini/opencode) |
| `--model <name>` | Override model for this invocation |
| `--agent <name>` | Agent preset name (from ~/.codeagent/models.json) |
| `--config <path>` | Path to models.json config file |
| `--cleanup` | Clean up log files on startup |
| `--worktree` | Execute in a new git worktree (auto-generates task ID) |
| `--skills <names>` | Comma-separated skill names for spec injection |
| `--prompt-file <path>` | Read prompt from file |
| `--reasoning-effort <level>` | Set reasoning effort (low/medium/high) |
| `--skip-permissions` | Skip permission prompts |
| `--parallel` | Enable parallel task execution |
| `--full-output` | Show full output in parallel mode |
| `--version`, `-v` | Print version and exit |

### Backend Selection

| Backend | Command | Best For |
|---------|---------|----------|
| **Codex** | `--backend codex` | General code tasks (default) |
| **Claude** | `--backend claude` | Complex reasoning, architecture |
| **Gemini** | `--backend gemini` | Fast iteration, prototyping |
| **OpenCode** | `--backend opencode` | Open-source alternative |

## Core Features

### 1. Multi-Backend Support

```bash
# Codex (default)
codeagent-wrapper "add logging to @app.js"

# Claude for architecture decisions
codeagent-wrapper --backend claude - <<'EOF'
Design a microservices architecture for e-commerce:
- Service boundaries
- Communication patterns
- Data consistency strategy
EOF

# Gemini for quick prototypes
codeagent-wrapper --backend gemini "create React component for user profile"
```

### 2. File References with @ Syntax

```bash
# Single file
codeagent-wrapper "optimize @src/utils.ts"

# Multiple files
codeagent-wrapper "refactor @src/auth.ts and @src/middleware.ts"

# Entire directory
codeagent-wrapper "analyze @src for security issues"
```

### 3. Session Management

```bash
# First task
codeagent-wrapper "add validation to user form"
# Output includes: SESSION_ID: 019a7247-ac9d-71f3-89e2-a823dbd8fd14

# Resume session
codeagent-wrapper resume 019a7247-ac9d-71f3-89e2-a823dbd8fd14 - <<'EOF'
Now add error messages for each validation rule
EOF
```

### 4. Parallel Execution

Execute multiple tasks concurrently with dependency management:

```bash
# Default: summary output (context-efficient, recommended)
codeagent-wrapper --parallel <<'EOF'
---TASK---
id: backend_1701234567
workdir: /project/backend
---CONTENT---
implement /api/users endpoints with CRUD operations

---TASK---
id: frontend_1701234568
workdir: /project/frontend
---CONTENT---
build Users page consuming /api/users

---TASK---
id: tests_1701234569
workdir: /project/tests
dependencies: backend_1701234567, frontend_1701234568
---CONTENT---
add integration tests for user management flow
EOF

# Full output mode (for debugging, includes complete task messages)
codeagent-wrapper --parallel --full-output <<'EOF'
...
EOF
```

**Output Modes:**
- **Summary (default)**: Structured report with extracted `Did/Files/Tests/Coverage`, plus a short action summary.
- **Full (`--full-output`)**: Complete task messages included. Use only for debugging.

**Summary Output Example:**
```
=== Execution Report ===
3 tasks | 2 passed | 1 failed | 1 below 90%

## Task Results

### backend_api ✓ 92%
Did: Implemented /api/users CRUD endpoints
Files: backend/users.go, backend/router.go
Tests: 12 passed
Log: /tmp/codeagent-xxx.log

### frontend_form ⚠️ 88% (below 90%)
Did: Created login form with validation
Files: frontend/LoginForm.tsx
Tests: 8 passed
Gap: lines not covered: frontend/LoginForm.tsx:42-47
Log: /tmp/codeagent-yyy.log

### integration_tests ✗ FAILED
Exit code: 1
Error: Assertion failed at line 45
Detail: Expected status 200 but got 401
Log: /tmp/codeagent-zzz.log

## Summary
- 2/3 completed successfully
- Fix: integration_tests (Assertion failed at line 45)
- Coverage: frontend_form
```

**Parallel Task Format:**
- `---TASK---` - Starts task block
- `id: <unique_id>` - Required, use `<feature>_<timestamp>` format
- `workdir: <path>` - Optional, defaults to current directory
- `dependencies: <id1>, <id2>` - Optional, comma-separated task IDs
- `---CONTENT---` - Separates metadata from task content

**Features:**
- Automatic topological sorting
- Unlimited concurrency for independent tasks
- Error isolation (failures don't stop other tasks)
- Dependency blocking (skip if parent fails)

### 5. Working Directory

```bash
# Execute in specific directory
codeagent-wrapper "run tests" /path/to/project

# With backend selection
codeagent-wrapper --backend claude "analyze code" /project/backend

# With HEREDOC
codeagent-wrapper - /path/to/project <<'EOF'
refactor database layer
EOF
```

## Advanced Usage

### Timeout Control

```bash
# Set custom timeout (1 hour = 3600000ms)
CODEX_TIMEOUT=3600000 codeagent-wrapper "long running task"

# Default timeout: 7200000ms (2 hours)
```

**Timeout behavior:**
- Sends SIGTERM to backend process
- Waits 5 seconds
- Sends SIGKILL if process doesn't exit
- Returns exit code 124 (consistent with GNU timeout)

### Complex Multi-line Tasks

Use HEREDOC to avoid shell escaping issues:

```bash
codeagent-wrapper - <<'EOF'
Refactor authentication system:

Current issues:
- Password stored as plain text
- No rate limiting on login
- Sessions don't expire

Requirements:
1. Hash passwords with bcrypt
2. Add rate limiting (5 attempts/15min)
3. Session expiry after 24h
4. Add refresh token mechanism

Files to modify:
- @src/auth/login.ts
- @src/middleware/rateLimit.ts
- @config/session.ts
EOF
```

### Backend-Specific Features

**Codex:**
```bash
# Best for code editing and refactoring
codeagent-wrapper --backend codex - <<'EOF'
extract duplicate code in @src into reusable helpers
EOF
```

**Claude:**
```bash
# Best for complex reasoning
codeagent-wrapper --backend claude - <<'EOF'
review @src/payment/processor.ts for:
- Race conditions
- Edge cases
- Security vulnerabilities
EOF
```

**Gemini:**
```bash
# Best for fast iteration
codeagent-wrapper --backend gemini "add TypeScript types to @api.js"
```

## Output Format

Standard output includes parsed agent messages and session ID:

```
Agent response text here...
Implementation details...

---
SESSION_ID: 019a7247-ac9d-71f3-89e2-a823dbd8fd14
```

Error output (stderr):
```
ERROR: Error message details
```

Parallel execution output:
```
=== Parallel Execution Summary ===
Total: 3 | Success: 2 | Failed: 1

--- Task: backend_1701234567 ---
Status: SUCCESS
Session: 019a7247-ac9d-71f3-89e2-a823dbd8fd14

Implementation complete...

--- Task: frontend_1701234568 ---
Status: SUCCESS
Session: 019a7248-ac9d-71f3-89e2-a823dbd8fd14

UI components created...

--- Task: tests_1701234569 ---
Status: FAILED (exit code 1)
Error: dependency backend_1701234567 failed
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error (missing args, no output) |
| 124 | Timeout |
| 127 | Backend command not found |
| 130 | Interrupted (Ctrl+C) |
| * | Passthrough from backend process |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CODEX_TIMEOUT` | 7200000 | Timeout in milliseconds |
| `CODEX_BYPASS_SANDBOX` | true | Bypass Codex sandbox/approval. Set `false` to disable |
| `CODEAGENT_SKIP_PERMISSIONS` | true | Skip Claude permission prompts. Set `false` to disable |

## Troubleshooting

**Backend not found:**
```bash
# Ensure backend CLI is installed
which codex
which claude
which gemini

# Check PATH
echo $PATH
```

**Timeout too short:**
```bash
# Increase timeout to 4 hours
CODEX_TIMEOUT=14400000 codeagent-wrapper "complex task"
```

**Session ID not found:**
```bash
# List recent sessions (backend-specific)
codex history

# Ensure session ID is copied correctly
codeagent-wrapper resume <session_id> "continue task"
```

**Parallel tasks not running:**
```bash
# Check task format
# Ensure ---TASK--- and ---CONTENT--- delimiters are correct
# Verify task IDs are unique
# Check dependencies reference existing task IDs
```

## Integration with Claude Code

Use via the `codeagent` skill:

```bash
# In Claude Code conversation
User: Use codeagent to implement authentication

# Claude will execute:
codeagent-wrapper --backend codex - <<'EOF'
implement JWT authentication in @src/auth
EOF
```

## Performance Tips

1. **Use parallel execution** for independent tasks
2. **Choose the right backend** for the task type
3. **Keep working directory specific** to reduce context
4. **Resume sessions** for multi-step workflows
5. **Use @ syntax** to minimize file content in prompts

## Best Practices

1. **HEREDOC for complex tasks** - Avoid shell escaping nightmares
2. **Descriptive task IDs** - Use `<feature>_<timestamp>` format
3. **Absolute paths** - Avoid relative path confusion
4. **Session resumption** - Continue conversations with context
5. **Timeout tuning** - Set appropriate timeouts for task complexity

## Examples

### Example 1: Code Review

```bash
codeagent-wrapper --backend claude - <<'EOF'
Review @src/payment/stripe.ts for:
1. Security issues (API key handling, input validation)
2. Error handling (network failures, API errors)
3. Edge cases (duplicate charges, partial refunds)
4. Code quality (naming, structure, comments)
EOF
```

### Example 2: Refactoring

```bash
codeagent-wrapper --backend codex - <<'EOF'
Refactor @src/utils:
- Extract duplicate code into helpers
- Add TypeScript types
- Improve function naming
- Add JSDoc comments
EOF
```

### Example 3: Full-Stack Feature

```bash
codeagent-wrapper --parallel <<'EOF'
---TASK---
id: api_1701234567
workdir: /project/backend
---CONTENT---
implement /api/notifications endpoints with WebSocket support

---TASK---
id: ui_1701234568
workdir: /project/frontend
dependencies: api_1701234567
---CONTENT---
build Notifications component with real-time updates

---TASK---
id: tests_1701234569
workdir: /project
dependencies: api_1701234567, ui_1701234568
---CONTENT---
add E2E tests for notification flow
EOF
```

## Further Reading

- [Codex CLI Documentation](https://codex.docs)
- [Claude CLI Documentation](https://claude.ai/docs)
- [Gemini CLI Documentation](https://ai.google.dev/docs)
- [Architecture Overview](./architecture.md)
