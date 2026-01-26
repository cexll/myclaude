[中文](README_CN.md) [English](README.md)

# Claude Code Multi-Agent Workflow System

[![Run in Smithery](https://smithery.ai/badge/skills/cexll)](https://smithery.ai/skills?ns=cexll&utm_source=github&utm_medium=badge)
[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL_v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![Claude Code](https://img.shields.io/badge/Claude-Code-blue)](https://claude.ai/code)
[![Version](https://img.shields.io/badge/Version-6.x-green)](https://github.com/cexll/myclaude)

> AI-powered development automation with multi-backend execution (Codex/Claude/Gemini/OpenCode)

## Quick Start

```bash
npx github:cexll/myclaude
```

## Modules Overview

| Module | Description | Documentation |
|--------|-------------|---------------|
| [do](skills/do/README.md) | **Recommended** - 7-phase feature development with codeagent orchestration | `/do` command |
| [omo](skills/omo/README.md) | Multi-agent orchestration with intelligent routing | `/omo` command |
| [bmad](agents/bmad/README.md) | BMAD agile workflow with 6 specialized agents | `/bmad-pilot` command |
| [requirements](agents/requirements/README.md) | Lightweight requirements-to-code pipeline | `/requirements-pilot` command |
| [essentials](agents/development-essentials/README.md) | Core development commands and utilities | `/code`, `/debug`, etc. |
| [sparv](skills/sparv/README.md) | SPARV workflow (Specify→Plan→Act→Review→Vault) | `/sparv` command |
| course | Course development (combines dev + product-requirements + test-cases) | Composite module |

## Installation

```bash
# Interactive installer (recommended)
npx github:cexll/myclaude

# List installable items (modules / skills / wrapper)
npx github:cexll/myclaude --list

# Custom install directory / overwrite
npx github:cexll/myclaude --install-dir ~/.claude --force
```

### Module Configuration

Edit `config.json` to enable/disable modules:

```json
{
  "modules": {
    "bmad": { "enabled": false },
    "requirements": { "enabled": false },
    "essentials": { "enabled": false },
    "omo": { "enabled": false },
    "sparv": { "enabled": false },
    "do": { "enabled": true },
    "course": { "enabled": false }
  }
}
```

## Workflow Selection Guide

| Scenario | Recommended |
|----------|-------------|
| Feature development (default) | `/do` |
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

- [codeagent-wrapper](codeagent-wrapper/README.md)
- [Plugin System](PLUGIN_README.md)

## Troubleshooting

### Common Issues

**Codex wrapper not found:**
```bash
# Select: codeagent-wrapper
npx github:cexll/myclaude
```

**Module not loading:**
```bash
cat ~/.claude/installed_modules.json
npx github:cexll/myclaude --force
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
| Codex permission denied | Set `approval_policy = "never"` in ~/.codex/config.yaml |

See [GitHub Issues](https://github.com/cexll/myclaude/issues) for more.

## License

AGPL-3.0 - see [LICENSE](LICENSE)

### Commercial Licensing

For commercial use without AGPL obligations, contact: evanxian9@gmail.com

## Support

- [GitHub Issues](https://github.com/cexll/myclaude/issues)
