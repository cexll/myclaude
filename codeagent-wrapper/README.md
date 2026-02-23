# codeagent-wrapper

[English](README.md) | [中文](README_CN.md)

A multi-backend AI code agent CLI wrapper written in Go. Provides a unified CLI entry point wrapping different AI tool backends (Codex / Claude / Gemini / OpenCode) with consistent flags, configuration, skill injection, and session resumption.

Entry point: `cmd/codeagent-wrapper/main.go` (binary: `codeagent-wrapper`).

## Features

- **Multi-backend support**: `codex` / `claude` / `gemini` / `opencode`
- **Unified CLI**: `codeagent-wrapper [flags] <task>` / `codeagent-wrapper resume <session_id> <task> [workdir]`
- **Auto stdin**: Automatically pipes via stdin when task contains newlines, special characters, or exceeds length; also supports explicit `-`
- **Config merging**: Config files + `CODEAGENT_*` environment variables (viper)
- **Agent presets**: Read backend/model/prompt/reasoning/yolo/allowed_tools from `~/.codeagent/models.json`
- **Dynamic agents**: Place a `{name}.md` prompt file in `~/.codeagent/agents/` to use as an agent
- **Skill auto-injection**: `--skills` for manual specification, or auto-detect from project tech stack (Go/Rust/Python/Node.js/Vue)
- **Git worktree isolation**: `--worktree` executes tasks in an isolated git worktree with auto-generated task_id and branch
- **Parallel execution**: `--parallel` reads multi-task config from stdin with dependency-aware topological concurrent execution and structured summary reports
- **Backend config**: `backends` section in `models.json` supports per-backend `base_url` / `api_key` injection
- **Claude tool control**: `allowed_tools` / `disallowed_tools` to restrict available tools for Claude backend
- **Stderr noise filtering**: Automatically filters noisy stderr output from Gemini and Codex backends
- **Log cleanup**: `codeagent-wrapper cleanup` cleans old logs (logs written to system temp directory)
- **Cross-platform**: macOS / Linux / Windows

## Installation

### Recommended (interactive installer)

```bash
npx github:stellarlinkco/myclaude
```

Select the `codeagent-wrapper` module to install.

### Manual build

Requires: Go 1.21+.

```bash
# Build from source
make build

# Or install to $GOPATH/bin
make install
```

Verify installation:

```bash
codeagent-wrapper --version
```

## Usage

Basic usage (default backend: `codex`):

```bash
codeagent-wrapper "analyze the entry logic of internal/app/cli.go"
```

Specify backend:

```bash
codeagent-wrapper --backend claude "explain the parallel config format in internal/executor/parallel_config.go"
```

Specify working directory (2nd positional argument):

```bash
codeagent-wrapper "search for potential data races in this repo" .
```

Explicit stdin (using `-`):

```bash
cat task.txt | codeagent-wrapper -
```

HEREDOC (recommended for multi-line tasks):

```bash
codeagent-wrapper --backend claude - <<'EOF'
Implement user authentication:
- JWT tokens
- bcrypt password hashing
- Session management
EOF
```

Resume session:

```bash
codeagent-wrapper resume <session_id> "continue the previous task"
```

Execute in isolated git worktree:

```bash
codeagent-wrapper --worktree "refactor the auth module"
```

Manual skill injection:

```bash
codeagent-wrapper --skills golang-base-practices "optimize database queries"
```

Parallel mode (task config from stdin):

```bash
codeagent-wrapper --parallel <<'EOF'
---TASK---
id: t1
workdir: .
backend: codex
---CONTENT---
List the main modules and their responsibilities.
---TASK---
id: t2
dependencies: t1
backend: claude
---CONTENT---
Based on t1's findings, identify refactoring risks and suggestions.
EOF
```

## CLI Flags

| Flag | Description |
|------|-------------|
| `--backend <name>` | Backend selection (codex/claude/gemini/opencode) |
| `--model <name>` | Model override |
| `--agent <name>` | Agent preset name (from models.json or ~/.codeagent/agents/) |
| `--prompt-file <path>` | Read prompt from file |
| `--skills <names>` | Comma-separated skill names for spec injection |
| `--reasoning-effort <level>` | Reasoning effort (backend-specific) |
| `--skip-permissions` | Skip permission prompts |
| `--dangerously-skip-permissions` | Alias for `--skip-permissions` |
| `--worktree` | Execute in a new git worktree (auto-generates task_id) |
| `--parallel` | Parallel task mode (config from stdin) |
| `--full-output` | Full output in parallel mode (default: summary only) |
| `--config <path>` | Config file path (default: `$HOME/.codeagent/config.*`) |
| `--version`, `-v` | Print version |
| `--cleanup` | Clean up old logs |

## Configuration

### Config File

Default search path (when `--config` is empty):

- `$HOME/.codeagent/config.(yaml|yml|json|toml|...)`

Example (YAML):

```yaml
backend: codex
model: gpt-4.1
skip-permissions: false
```

Can also be specified explicitly via `--config /path/to/config.yaml`.

### Environment Variables (`CODEAGENT_*`)

Read via viper with automatic `-` to `_` mapping:

| Variable | Description |
|----------|-------------|
| `CODEAGENT_BACKEND` | Backend name (codex/claude/gemini/opencode) |
| `CODEAGENT_MODEL` | Model name |
| `CODEAGENT_AGENT` | Agent preset name |
| `CODEAGENT_PROMPT_FILE` | Prompt file path |
| `CODEAGENT_REASONING_EFFORT` | Reasoning effort |
| `CODEAGENT_SKIP_PERMISSIONS` | Skip permission prompts (default true; set `false` to disable) |
| `CODEAGENT_FULL_OUTPUT` | Full output in parallel mode |
| `CODEAGENT_MAX_PARALLEL_WORKERS` | Parallel worker count (0=unlimited, max 100) |
| `CODEAGENT_TMPDIR` | Custom temp directory (for macOS permission issues) |
| `CODEX_TIMEOUT` | Timeout in ms (default 7200000 = 2 hours) |
| `CODEX_BYPASS_SANDBOX` | Codex sandbox bypass (default true; set `false` to disable) |
| `DO_WORKTREE_DIR` | Reuse existing worktree directory (set by /do workflow) |

### Agent Presets (`~/.codeagent/models.json`)

```json
{
  "default_backend": "codex",
  "default_model": "gpt-4.1",
  "backends": {
    "codex": { "api_key": "..." },
    "claude": { "base_url": "http://localhost:23001", "api_key": "..." }
  },
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

Use `--agent <name>` to select a preset. Agents inherit `base_url` / `api_key` from the corresponding `backends` entry.

### Dynamic Agents

Place a `{name}.md` file in `~/.codeagent/agents/` to use it via `--agent {name}`. The Markdown file is read as the prompt, using `default_backend` and `default_model`.

### Skill Auto-Detection

When no skills are specified via `--skills`, codeagent-wrapper auto-detects the tech stack from files in the working directory:

| Detected Files | Injected Skills |
|----------------|-----------------|
| `go.mod` / `go.sum` | `golang-base-practices` |
| `Cargo.toml` | `rust-best-practices` |
| `pyproject.toml` / `setup.py` / `requirements.txt` | `python-best-practices` |
| `package.json` | `vercel-react-best-practices`, `frontend-design` |
| `vue.config.js` / `vite.config.ts` / `nuxt.config.ts` | `vue-web-app` |

Skill specs are read from `~/.claude/skills/{name}/SKILL.md`, subject to a 16000-character budget.

## Supported Backends

This project does not embed model capabilities. It requires the corresponding CLI tools installed and available in `PATH`:

| Backend | Command | Notes |
|---------|---------|-------|
| `codex` | `codex e ...` | Adds `--dangerously-bypass-approvals-and-sandbox` by default; set `CODEX_BYPASS_SANDBOX=false` to disable |
| `claude` | `claude -p ... --output-format stream-json` | Skips permissions and disables setting-sources to prevent recursion; set `CODEAGENT_SKIP_PERMISSIONS=false` to enable prompts; auto-reads env and model from `~/.claude/settings.json` |
| `gemini` | `gemini -o stream-json -y ...` | Auto-loads env vars from `~/.gemini/.env` (GEMINI_API_KEY, GEMINI_MODEL, etc.) |
| `opencode` | `opencode run --format json` | — |

## Project Structure

```
cmd/codeagent-wrapper/main.go   # CLI entry point
internal/
  app/          # CLI command definitions, argument parsing, main orchestration
  backend/      # Backend abstraction and implementations (codex/claude/gemini/opencode)
  config/       # Config loading, agent resolution, viper bindings
  executor/     # Task execution engine: single/parallel/worktree/skill injection
  logger/       # Structured logging system
  parser/       # JSON stream parser
  utils/        # Common utility functions
  worktree/     # Git worktree management
```

## Development

```bash
make build    # Build binary
make test     # Run tests
make lint     # golangci-lint + staticcheck
make clean    # Clean build artifacts
make install  # Install to $GOPATH/bin
```

CI uses GitHub Actions with Go 1.21 / 1.22 matrix testing.

## Troubleshooting

- On macOS, if you see `permission denied` related to temp directories, set: `CODEAGENT_TMPDIR=$HOME/.codeagent/tmp`
- `claude` backend's `base_url` / `api_key` (from `~/.codeagent/models.json` `backends.claude`) are injected as `ANTHROPIC_BASE_URL` / `ANTHROPIC_API_KEY` env vars
- `gemini` backend's API key is loaded from `~/.gemini/.env`, injected as `GEMINI_API_KEY` with `GEMINI_API_KEY_AUTH_MECHANISM=bearer` auto-set
- Exit codes: 127 = backend not found, 124 = timeout, 130 = interrupted
- Parallel mode outputs structured summary by default; use `--full-output` for complete output when debugging
