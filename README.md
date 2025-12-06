# Claude Code Multi-Agent Workflow System

[![Run in Smithery](https://smithery.ai/badge/skills/cexll)](https://smithery.ai/skills?ns=cexll&utm_source=github&utm_medium=badge)


[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Claude Code](https://img.shields.io/badge/Claude-Code-blue)](https://claude.ai/code)
[![Version](https://img.shields.io/badge/Version-5.0-green)](https://github.com/cexll/myclaude)

> AI-powered development automation with Claude Code + Codex collaboration

## Core Concept: Claude Code + Codex

This system leverages a **dual-agent architecture**:

| Role | Agent | Responsibility |
|------|-------|----------------|
| **Orchestrator** | Claude Code | Planning, context gathering, verification, user interaction |
| **Executor** | Codex | Code editing, test execution, file operations |

**Why this separation?**
- Claude Code excels at understanding context and orchestrating complex workflows
- Codex excels at focused code generation and execution
- Together they provide better results than either alone

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
├── CLAUDE.md              # Core instructions and role definition
├── commands/              # Slash commands (/dev, /code, etc.)
├── agents/                # Agent definitions
├── skills/
│   └── codex/
│       └── SKILL.md       # Codex integration skill
└── installed_modules.json # Installation status
```

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
codex-wrapper - <<'EOF'
implement @src/auth.ts with JWT validation
EOF
```

### Parallel Execution

```bash
codex-wrapper --parallel <<'EOF'
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

Windows installs place `codex-wrapper.exe` in `%USERPROFILE%\bin`.

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
# Check PATH
echo $PATH | grep -q "$HOME/bin" || echo 'export PATH="$HOME/bin:$PATH"' >> ~/.zshrc

# Reinstall
bash install.sh
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

---

## License

MIT License - see [LICENSE](LICENSE)

## Support

- **Issues**: [GitHub Issues](https://github.com/cexll/myclaude/issues)
- **Documentation**: [docs/](docs/)

---

**Claude Code + Codex = Better Development** - Orchestration meets execution.
