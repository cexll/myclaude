[中文](README_CN.md) [English](README.md)

# Claude Code Multi-Agent Workflow System

[![Run in Smithery](https://smithery.ai/badge/skills/cexll)](https://smithery.ai/skills?ns=cexll&utm_source=github&utm_medium=badge)
[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL_v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![Claude Code](https://img.shields.io/badge/Claude-Code-blue)](https://claude.ai/code)
[![Version](https://img.shields.io/badge/Version-6.x-green)](https://github.com/cexll/myclaude)

> AI-powered development automation with multi-backend execution (Codex/Claude/Gemini/OpenCode)

## Quick Start

```bash
git clone https://github.com/cexll/myclaude.git
cd myclaude
python3 install.py --install-dir ~/.claude
```

## Modules Overview

| Module | Description | Documentation |
|--------|-------------|---------------|
| [do](skills/do/README.md) | **Recommended** - 7-phase feature development with codeagent orchestration | `/do` command |
| [dev](dev-workflow/README.md) | Lightweight dev workflow with Codex integration | `/dev` command |
| [omo](skills/omo/README.md) | Multi-agent orchestration with intelligent routing | `/omo` command |
| [bmad](bmad-agile-workflow/README.md) | BMAD agile workflow with 6 specialized agents | `/bmad-pilot` command |
| [requirements](requirements-driven-workflow/README.md) | Lightweight requirements-to-code pipeline | `/requirements-pilot` command |
| [essentials](development-essentials/README.md) | Core development commands and utilities | `/code`, `/debug`, etc. |
| [sparv](skills/sparv/README.md) | SPARV workflow (Specify→Plan→Act→Review→Vault) | `/sparv` command |
| course | Course development (combines dev + product-requirements + test-cases) | Composite module |

## Installation

```bash
# Install all enabled modules
python3 install.py --install-dir ~/.claude

# Install specific module
python3 install.py --module dev

# List available modules
python3 install.py --list-modules

# Force overwrite
python3 install.py --force
```

### Module Configuration

Edit `config.json` to enable/disable modules:

```json
{
  "modules": {
    "dev": { "enabled": true },
    "bmad": { "enabled": false },
    "requirements": { "enabled": false },
    "essentials": { "enabled": false },
    "omo": { "enabled": false },
    "sparv": { "enabled": false },
    "do": { "enabled": false },
    "course": { "enabled": false }
  }
}
```

## Workflow Selection Guide

| Scenario | Recommended |
|----------|-------------|
| Feature development (default) | `/do` |
| Lightweight feature | `/dev` |
| Bug investigation + fix | `/omo` |
| Large enterprise project | `/bmad-pilot` |
| Quick prototype | `/requirements-pilot` |
| Simple task | `/code`, `/debug` |

## Core Architecture

| Role | Agent | Responsibility |
|------|-------|----------------|
| **Orchestrator** | Claude Code | Planning, context gathering, verification |
| **Executor** | codeagent-wrapper | Code editing, test execution (Codex/Claude/Gemini/OpenCode) |

## Backend CLI Requirements

| Backend | Required Features |
|---------|-------------------|
| Codex | `codex e`, `--json`, `-C`, `resume` |
| Claude | `--output-format stream-json`, `-r` |
| Gemini | `-o stream-json`, `-y`, `-r` |

## Directory Structure After Installation

```
~/.claude/
├── bin/codeagent-wrapper
├── CLAUDE.md
├── commands/
├── agents/
├── skills/
└── config.json
```

## Documentation

- [Codeagent-Wrapper Guide](docs/CODEAGENT-WRAPPER.md)
- [Hooks Documentation](docs/HOOKS.md)
- [codeagent-wrapper](codeagent-wrapper/README.md)

## Troubleshooting

### Common Issues

**Codex wrapper not found:**
```bash
bash install.sh
```

**Module not loading:**
```bash
cat ~/.claude/installed_modules.json
python3 install.py --module <name> --force
```

**Backend CLI errors:**
```bash
which codex && codex --version
which claude && claude --version
which gemini && gemini --version
```

## FAQ

| Issue | Solution |
|-------|----------|
| "Unknown event format" | Logging display issue, can be ignored |
| Gemini can't read .gitignore files | Remove from .gitignore or use different backend |
| `/dev` slow | Check logs, try faster model, use single repo |
| Codex permission denied | Set `approval_policy = "never"` in ~/.codex/config.yaml |

See [GitHub Issues](https://github.com/cexll/myclaude/issues) for more.

## License

AGPL-3.0 - see [LICENSE](LICENSE)

## Support

- [GitHub Issues](https://github.com/cexll/myclaude/issues)
- [Documentation](docs/)
