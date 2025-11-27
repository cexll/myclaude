# Claude Code Multi-Agent Workflow System

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Claude Code](https://img.shields.io/badge/Claude-Code-blue)](https://claude.ai/code)
[![Version](https://img.shields.io/badge/Version-4.4-green)](https://github.com/cexll/myclaude)
[![Plugin Ready](https://img.shields.io/badge/Plugin-Ready-purple)](https://docs.claude.com/en/docs/claude-code/plugins)

> Enterprise-grade agile development automation with AI-powered multi-agent orchestration

[中文文档](README_CN.md) | [Documentation](docs/)

## 🚀 Quick Start

### Installation

**Plugin System (Recommended)**
```bash
/plugin marketplace add cexll/myclaude
```

**Traditional Installation**
```bash
git clone https://github.com/cexll/myclaude.git
cd myclaude
make install
```

### Basic Usage

```bash
# Full agile workflow
/bmad-pilot "Build user authentication with OAuth2 and MFA"

# Lightweight development
/requirements-pilot "Implement JWT token refresh"

# Direct development commands
/code "Add API rate limiting"
```

## 📦 Plugin Modules

| Plugin | Description | Key Commands |
|--------|-------------|--------------|
| **[bmad-agile-workflow](docs/BMAD-WORKFLOW.md)** | Complete BMAD methodology with 6 specialized agents | `/bmad-pilot` |
| **[requirements-driven-workflow](docs/REQUIREMENTS-WORKFLOW.md)** | Streamlined requirements-to-code workflow | `/requirements-pilot` |
| **[dev-workflow](dev-workflow/README.md)** | Extreme lightweight end-to-end development workflow | `/dev` |
| **[development-essentials](docs/DEVELOPMENT-COMMANDS.md)** | Core development slash commands | `/code` `/debug` `/test` `/optimize` |
| **[advanced-ai-agents](docs/ADVANCED-AGENTS.md)** | GPT-5 deep reasoning integration | Agent: `gpt5` |
| **[requirements-clarity](docs/REQUIREMENTS-CLARITY.md)** | Automated requirements clarification with 100-point scoring | Auto-activated skill |

## 💡 Use Cases

**BMAD Workflow** - Full agile process automation
- Product requirements → Architecture design → Sprint planning → Development → Code review → QA testing
- Quality gates with 90% thresholds
- Automated document generation

**Requirements Workflow** - Fast prototyping
- Requirements generation → Implementation → Review → Testing
- Lightweight and practical

**Development Commands** - Daily coding
- Direct implementation, debugging, testing, optimization
- No workflow overhead

**Requirements Clarity** - Automated requirements engineering
- Auto-detects vague requirements and initiates clarification
- 100-point quality scoring system
- Generates complete PRD documents

## 🎯 Key Features

- **🤖 Role-Based Agents**: Specialized AI agents for each development phase
- **📊 Quality Gates**: Automatic quality scoring with iterative refinement
- **✅ Approval Points**: User confirmation at critical workflow stages
- **📁 Persistent Artifacts**: All specs saved to `.claude/specs/`
- **🔌 Plugin System**: Native Claude Code plugin support
- **🔄 Flexible Workflows**: Choose full agile or lightweight development
- **🎯 Requirements Clarity**: Automated requirements clarification with quality scoring

## 📚 Documentation

- **[BMAD Workflow Guide](docs/BMAD-WORKFLOW.md)** - Complete methodology and agent roles
- **[Requirements Workflow](docs/REQUIREMENTS-WORKFLOW.md)** - Lightweight development process
- **[Development Commands](docs/DEVELOPMENT-COMMANDS.md)** - Slash command reference
- **[Plugin System](docs/PLUGIN-SYSTEM.md)** - Installation and configuration
- **[Quick Start Guide](docs/QUICK-START.md)** - Get started in 5 minutes

## 🛠️ Installation Methods

**Method 1: Plugin Install** (One command)
```bash
/plugin install bmad-agile-workflow
```

**Method 2: Make Commands** (Selective installation)
```bash
make deploy-bmad          # BMAD workflow only
make deploy-requirements  # Requirements workflow only
make deploy-all          # Everything
```

**Method 3: Manual Setup**
- Copy `./commands/*.md` to `~/.config/claude/commands/`
- Copy `./agents/*.md` to `~/.config/claude/agents/`

Run `make help` for all options.

## 📄 License

MIT License - see [LICENSE](LICENSE)

## 🙋 Support

- **Issues**: [GitHub Issues](https://github.com/cexll/myclaude/issues)
- **Documentation**: [docs/](docs/)
- **Plugin Guide**: [PLUGIN_README.md](PLUGIN_README.md)

---

**Transform your development with AI-powered automation** - One command, complete workflow, quality assured.
