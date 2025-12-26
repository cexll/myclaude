[中文](README_CN.md) [English](README.md)

# Claude Code Multi-Agent Workflow System

[![Run in Smithery](https://smithery.ai/badge/skills/cexll)](https://smithery.ai/skills?ns=cexll&utm_source=github&utm_medium=badge)


[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL_v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![Claude Code](https://img.shields.io/badge/Claude-Code-blue)](https://claude.ai/code)
[![Version](https://img.shields.io/badge/Version-5.2-green)](https://github.com/cexll/myclaude)

> AI-powered development automation with multi-backend execution (Codex/Claude/Gemini)

## Core Concept: Multi-Backend Architecture

This system leverages a **dual-agent architecture** with pluggable AI backends:

| Role | Agent | Responsibility |
|------|-------|----------------|
| **Orchestrator** | Claude Code | Planning, context gathering, verification, user interaction |
| **Executor** | codeagent-wrapper | Code editing, test execution (Codex/Claude/Gemini backends) |

**Why this separation?**
- Claude Code excels at understanding context and orchestrating complex workflows
- Specialized backends (Codex for code, Claude for reasoning, Gemini for prototyping) excel at focused execution
- Backend selection via `--backend codex|claude|gemini` matches the model to the task

## Quick Start(Please execute in Powershell on Windows)

```bash
git clone https://github.com/cexll/myclaude.git
cd myclaude
python3 install.py --install-dir ~/.claude
```

## Workflows Overview

### 1. Dev Workflow (Recommended)

**The primary workflow for most development tasks.**

```bash
/dev "implement user authentication with JWT"
```

**6-Step Process:**
1. **Requirements Clarification** - Interactive Q&A to clarify scope
2. **Codex Deep Analysis** - Codebase exploration and architecture decisions
3. **Dev Plan Generation** - Structured task breakdown with test requirements
4. **Parallel Execution** - Codex executes tasks concurrently
5. **Coverage Validation** - Enforce ≥90% test coverage
6. **Completion Summary** - Report with file changes and coverage stats

**Key Features:**
- Claude Code orchestrates, Codex executes all code changes
- Automatic task parallelization for speed
- Mandatory 90% test coverage gate
- Rollback on failure

**Best For:** Feature development, refactoring, bug fixes with tests

---

### 2. BMAD Agile Workflow

**Full enterprise agile methodology with 6 specialized agents.**

```bash
/bmad-pilot "build e-commerce checkout system"
```

**Agents:**
| Agent | Role |
|-------|------|
| Product Owner | Requirements & user stories |
| Architect | System design & tech decisions |
| Tech Lead | Sprint planning & task breakdown |
| Developer | Implementation |
| Code Reviewer | Quality assurance |
| QA Engineer | Testing & validation |

**Process:**
```
Requirements → Architecture → Sprint Plan → Development → Review → QA
     ↓              ↓             ↓            ↓          ↓       ↓
   PRD.md      DESIGN.md     SPRINT.md     Code      REVIEW.md  TEST.md
```

**Best For:** Large features, team coordination, enterprise projects

---

### 3. Requirements-Driven Workflow

**Lightweight requirements-to-code pipeline.**

```bash
/requirements-pilot "implement API rate limiting"
```

**Process:**
1. Requirements generation with quality scoring
2. Implementation planning
3. Code generation
4. Review and testing

**Best For:** Quick prototypes, well-defined features

---

### 4. Development Essentials

**Direct commands for daily coding tasks.**

| Command | Purpose |
|---------|---------|
| `/code` | Implement a feature |
| `/debug` | Debug an issue |
| `/test` | Write tests |
| `/review` | Code review |
| `/optimize` | Performance optimization |
| `/refactor` | Code refactoring |
| `/docs` | Documentation |

**Best For:** Quick tasks, no workflow overhead needed

## Enterprise Workflow Features

- **Multi-backend execution:** `codeagent-wrapper --backend codex|claude|gemini` (default `codex`) so you can match the model to the task without changing workflows.
- **GitHub workflow commands:** `/gh-create-issue "short need"` creates structured issues; `/gh-issue-implement 123` pulls issue #123, drives development, and prepares the PR.
- **Skills + hooks activation:** .claude/hooks run automation (tests, reviews), while `.claude/skills/skill-rules.json` auto-suggests the right skills. Keep hooks enabled in `.claude/settings.json` to activate the enterprise workflow helpers.

---

## Version Requirements

### Codex CLI
**Minimum version:** Check compatibility with your installation

The codeagent-wrapper uses these Codex CLI features:
- `codex e` - Execute commands (shorthand for `codex exec`)
- `--skip-git-repo-check` - Skip git repository validation
- `--json` - JSON stream output format
- `-C <workdir>` - Set working directory
- `resume <session_id>` - Resume previous sessions

**Verify Codex CLI is installed:**
```bash
which codex
codex --version
```

### Claude CLI
**Minimum version:** Check compatibility with your installation

Required features:
- `--output-format stream-json` - Streaming JSON output format
- `--setting-sources` - Control setting sources (prevents infinite recursion)
- `--dangerously-skip-permissions` - Skip permission prompts (use with caution)
- `-p` - Prompt input flag
- `-r <session_id>` - Resume sessions

**Security Note:** The wrapper only adds `--dangerously-skip-permissions` for Claude when explicitly enabled (e.g. `--skip-permissions` / `CODEAGENT_SKIP_PERMISSIONS=true`). Keep it disabled unless you understand the risk.

**Verify Claude CLI is installed:**
```bash
which claude
claude --version
```

### Gemini CLI
**Minimum version:** Check compatibility with your installation

Required features:
- `-o stream-json` - JSON stream output format
- `-y` - Auto-approve prompts (non-interactive mode)
- `-r <session_id>` - Resume sessions
- `-p` - Prompt input flag

**Verify Gemini CLI is installed:**
```bash
which gemini
gemini --version
```

---

## Installation

### Modular Installation (Recommended)

```bash
# Install all enabled modules (dev + essentials by default)
python3 install.py --install-dir ~/.claude

# Install specific module
python3 install.py --module dev

# List available modules
python3 install.py --list-modules

# Force overwrite existing files
python3 install.py --force
```

### Available Modules

| Module | Default | Description |
|--------|---------|-------------|
| `dev` | ✓ Enabled | Dev workflow + Codex integration |
| `essentials` | ✓ Enabled | Core development commands |
| `bmad` | Disabled | Full BMAD agile workflow |
| `requirements` | Disabled | Requirements-driven workflow |

### What Gets Installed

```
~/.claude/
├── bin/
│   └── codeagent-wrapper    # Main executable
├── CLAUDE.md                # Core instructions and role definition
├── commands/                # Slash commands (/dev, /code, etc.)
├── agents/                  # Agent definitions
├── skills/
│   └── codex/
│       └── SKILL.md         # Codex integration skill
├── config.json              # Configuration
└── installed_modules.json   # Installation status
```

### Customizing Installation Directory

By default, myclaude installs to `~/.claude`. You can customize this using the `INSTALL_DIR` environment variable:

```bash
# Install to custom directory
INSTALL_DIR=/opt/myclaude bash install.sh

# Update your PATH accordingly
export PATH="/opt/myclaude/bin:$PATH"
```

**Directory Structure:**
- `$INSTALL_DIR/bin/` - codeagent-wrapper binary
- `$INSTALL_DIR/skills/` - Skill definitions
- `$INSTALL_DIR/config.json` - Configuration file
- `$INSTALL_DIR/commands/` - Slash command definitions
- `$INSTALL_DIR/agents/` - Agent definitions

**Note:** When using a custom installation directory, ensure that `$INSTALL_DIR/bin` is added to your `PATH` environment variable.

### Configuration

Edit `config.json` to customize:

```json
{
  "version": "1.0",
  "install_dir": "~/.claude",
  "modules": {
    "dev": {
      "enabled": true,
      "operations": [
        {"type": "merge_dir", "source": "dev-workflow"},
        {"type": "copy_file", "source": "memorys/CLAUDE.md", "target": "CLAUDE.md"},
        {"type": "copy_file", "source": "skills/codex/SKILL.md", "target": "skills/codex/SKILL.md"},
        {"type": "run_command", "command": "bash install.sh"}
      ]
    }
  }
}
```

**Operation Types:**
| Type | Description |
|------|-------------|
| `merge_dir` | Merge subdirs (commands/, agents/) into install dir |
| `copy_dir` | Copy entire directory |
| `copy_file` | Copy single file to target path |
| `run_command` | Execute shell command |

---

## Codex Integration

The `codex` skill enables Claude Code to delegate code execution to Codex CLI.

### Usage in Workflows

```bash
# Codex is invoked via the skill
codeagent-wrapper - <<'EOF'
implement @src/auth.ts with JWT validation
EOF
```

### Parallel Execution

```bash
codeagent-wrapper --parallel <<'EOF'
---TASK---
id: backend_api
workdir: /project/backend
---CONTENT---
implement REST endpoints for /api/users

---TASK---
id: frontend_ui
workdir: /project/frontend
dependencies: backend_api
---CONTENT---
create React components consuming the API
EOF
```

### Install Codex Wrapper

```bash
# Automatic (via dev module)
python3 install.py --module dev

# Manual
bash install.sh
```

#### Windows

Windows installs place `codeagent-wrapper.exe` in `%USERPROFILE%\bin`.

```powershell
# PowerShell (recommended)
powershell -ExecutionPolicy Bypass -File install.ps1

# Batch (cmd)
install.bat
```

**Add to PATH** (if installer doesn't detect it):

```powershell
# PowerShell - persistent for current user
[Environment]::SetEnvironmentVariable('PATH', "$HOME\bin;" + [Environment]::GetEnvironmentVariable('PATH','User'), 'User')

# PowerShell - current session only
$Env:PATH = "$HOME\bin;$Env:PATH"
```

```batch
REM cmd.exe - persistent for current user
setx PATH "%USERPROFILE%\bin;%PATH%"
```

---

## Workflow Selection Guide

| Scenario | Recommended Workflow |
|----------|---------------------|
| New feature with tests | `/dev` |
| Quick bug fix | `/debug` or `/code` |
| Large multi-sprint feature | `/bmad-pilot` |
| Prototype or POC | `/requirements-pilot` |
| Code review | `/review` |
| Performance issue | `/optimize` |

---

## Troubleshooting

### Common Issues

**Codex wrapper not found:**
```bash
# Installer auto-adds PATH, check if configured
if [[ ":$PATH:" != *":$HOME/.claude/bin:"* ]]; then
    echo "PATH not configured. Reinstalling..."
    bash install.sh
fi

# Or manually add (idempotent command)
[[ ":$PATH:" != *":$HOME/.claude/bin:"* ]] && echo 'export PATH="$HOME/.claude/bin:$PATH"' >> ~/.zshrc
```

**Permission denied:**
```bash
python3 install.py --install-dir ~/.claude --force
```

**Module not loading:**
```bash
# Check installation status
cat ~/.claude/installed_modules.json

# Reinstall specific module
python3 install.py --module dev --force
```

### Version Compatibility Issues

**Backend CLI not found:**
```bash
# Check if backend CLIs are installed
which codex
which claude
which gemini

# Install missing backends
# Codex: Follow installation instructions at https://codex.docs
# Claude: Follow installation instructions at https://claude.ai/docs
# Gemini: Follow installation instructions at https://ai.google.dev/docs
```

**Unsupported CLI flags:**
```bash
# If you see errors like "unknown flag" or "invalid option"

# Check backend CLI version
codex --version
claude --version
gemini --version

# For Codex: Ensure it supports `e`, `--skip-git-repo-check`, `--json`, `-C`, and `resume`
# For Claude: Ensure it supports `--output-format stream-json`, `--setting-sources`, `-r`
# For Gemini: Ensure it supports `-o stream-json`, `-y`, `-r`, `-p`

# Update your backend CLI to the latest version if needed
```

**JSON parsing errors:**
```bash
# If you see "failed to parse JSON output" errors

# Verify the backend outputs stream-json format
codex e --json "test task"  # Should output newline-delimited JSON
claude --output-format stream-json -p "test"  # Should output stream JSON

# If not, your backend CLI version may be too old or incompatible
```

**Infinite recursion with Claude backend:**
```bash
# The wrapper prevents this with `--setting-sources ""` flag
# If you still see recursion, ensure your Claude CLI supports this flag

claude --help | grep "setting-sources"

# If flag is not supported, upgrade Claude CLI
```

**Session resume failures:**
```bash
# Check if session ID is valid
codex history  # List recent sessions
claude history

# Ensure backend CLI supports session resumption
codex resume <session_id> "test"  # Should continue from previous session
claude -r <session_id> "test"

# If not supported, use new sessions instead of resume mode
```

---

## FAQ (Frequently Asked Questions)

### Q1: `codeagent-wrapper` execution fails with "Unknown event format"

**Problem:**
```
Unknown event format: {"type":"turn.started"}
Unknown event format: {"type":"assistant", ...}
```

**Solution:**
This is a logging event format display issue and does not affect actual functionality. It will be fixed in the next version. You can ignore these log outputs.

**Related Issue:** [#96](https://github.com/cexll/myclaude/issues/96)

---

### Q2: Gemini cannot read files ignored by `.gitignore`

**Problem:**
When using `codeagent-wrapper --backend gemini`, files in directories like `.claude/` that are ignored by `.gitignore` cannot be read.

**Solution:**
- **Option 1:** Remove `.claude/` from your `.gitignore` file
- **Option 2:** Ensure files that need to be read are not in `.gitignore` list

**Related Issue:** [#75](https://github.com/cexll/myclaude/issues/75)

---

### Q3: `/dev` command parallel execution is very slow

**Problem:**
Using `/dev` command for simple features takes too long (over 30 minutes) with no visibility into task progress.

**Solution:**
1. **Check logs:** Review `C:\Users\User\AppData\Local\Temp\codeagent-wrapper-*.log` to identify bottlenecks
2. **Adjust backend:**
   - Try faster models like `gpt-5.1-codex-max`
   - Running in WSL may be significantly faster
3. **Workspace:** Use a single repository instead of monorepo with multiple sub-projects

**Related Issue:** [#77](https://github.com/cexll/myclaude/issues/77)

---

### Q4: Codex permission denied with new Go version

**Problem:**
After upgrading to the new Go-based Codex implementation, execution fails with permission denied errors.

**Solution:**
Add the following configuration to `~/.codex/config.yaml` (Windows: `c:\user\.codex\config.toml`):
```yaml
model = "gpt-5.1-codex-max"
model_reasoning_effort = "high"
model_reasoning_summary = "detailed"
approval_policy = "never"
sandbox_mode = "workspace-write"
disable_response_storage = true
network_access = true
```

**Key settings:**
- `approval_policy = "never"` - Remove approval restrictions
- `sandbox_mode = "workspace-write"` - Allow workspace write access
- `network_access = true` - Enable network access

**Related Issue:** [#31](https://github.com/cexll/myclaude/issues/31)

---

**Still having issues?** Visit [GitHub Issues](https://github.com/cexll/myclaude/issues) to search or report new issues.

---

## Documentation
- **[Codeagent-Wrapper Guide](docs/CODEAGENT-WRAPPER.md)** - Multi-backend execution wrapper
- **[Hooks Documentation](docs/HOOKS.md)** - Custom hooks and automation

### Additional Resources
- **[Installation Log](install.log)** - Installation history and troubleshooting

---

## License

AGPL-3.0 License - see [LICENSE](LICENSE)

## Support

- **Issues**: [GitHub Issues](https://github.com/cexll/myclaude/issues)
- **Documentation**: [docs/](docs/)

---

**Claude Code + Codex = Better Development** - Orchestration meets execution.
