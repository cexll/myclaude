# Plugin System Guide

> Native Claude Code plugin support for modular workflow installation

## 🎯 Overview

This repository provides 4 ready-to-use Claude Code plugins that can be installed individually or as a complete suite.

## 📦 Available Plugins

### 1. bmad-agile-workflow

**Complete BMAD methodology with 6 specialized agents**

**Commands**:
- `/bmad-pilot` - Full agile workflow orchestration

**Agents**:
- `bmad-po` - Product Owner (Sarah)
- `bmad-architect` - System Architect (Winston)
- `bmad-sm` - Scrum Master (Mike)
- `bmad-dev` - Developer (Alex)
- `bmad-review` - Code Reviewer
- `bmad-qa` - QA Engineer (Emma)
- `bmad-orchestrator` - Main orchestrator

**Use for**: Enterprise projects, complex features, full agile process

### 2. requirements-driven-workflow

**Streamlined requirements-to-code workflow**

**Commands**:
- `/requirements-pilot` - Requirements-driven development flow

**Agents**:
- `requirements-generate` - Requirements generation
- `requirements-code` - Code implementation
- `requirements-review` - Code review
- `requirements-testing` - Testing strategy

**Use for**: Quick prototyping, simple features, rapid development

### 3. development-essentials

**Core development slash commands**

**Commands**:
- `/code` - Direct implementation
- `/debug` - Systematic debugging
- `/test` - Testing strategy
- `/optimize` - Performance tuning
- `/bugfix` - Bug resolution
- `/refactor` - Code improvement
- `/review` - Code validation
- `/ask` - Technical consultation
- `/docs` - Documentation
- `/think` - Advanced analysis

**Agents**:
- `code` - Code implementation
- `bugfix` - Bug fixing
- `debug` - Debugging
- `develop` - General development

**Use for**: Daily coding tasks, quick implementations

### 4. advanced-ai-agents

**GPT-5 deep reasoning integration**

**Commands**: None (agent-only)

**Agents**:
- `gpt5` - Deep reasoning and analysis

**Use for**: Complex architectural decisions, strategic planning

## 🚀 Installation Methods

### Method 1: Plugin Commands (Recommended)

```bash
# List all available plugins
/plugin list

# Get detailed information about a plugin
/plugin info bmad-agile-workflow

# Install a specific plugin
/plugin install bmad-agile-workflow

# Install all plugins
/plugin install bmad-agile-workflow
/plugin install requirements-driven-workflow
/plugin install development-essentials
/plugin install advanced-ai-agents

# Remove an installed plugin
/plugin remove development-essentials
```

### Method 2: Repository Reference

```bash
# Install from GitHub repository
/plugin marketplace add cexll/myclaude
```

This will present all available plugins from the repository.

### Method 3: Make Commands

For traditional installation or selective deployment:

```bash
# Install everything
make install

# Deploy specific workflows
make deploy-bmad          # BMAD workflow only
make deploy-requirements  # Requirements workflow only
make deploy-commands      # All slash commands
make deploy-agents       # All agents

# Deploy everything
make deploy-all

# View all options
make help
```

### Method 4: Manual Installation

Copy files to Claude Code configuration directories:

**Commands**:
```bash
cp bmad-agile-workflow/commands/*.md ~/.config/claude/commands/
cp requirements-driven-workflow/commands/*.md ~/.config/claude/commands/
cp development-essentials/commands/*.md ~/.config/claude/commands/
```

**Agents**:
```bash
cp bmad-agile-workflow/agents/*.md ~/.config/claude/agents/
cp requirements-driven-workflow/agents/*.md ~/.config/claude/agents/
cp development-essentials/agents/*.md ~/.config/claude/agents/
cp advanced-ai-agents/agents/*.md ~/.config/claude/agents/
```

**Output Styles** (optional):
```bash
cp output-styles/*.md ~/.config/claude/output-styles/
```

## 📋 Plugin Configuration

Plugins are defined in `.claude-plugin/marketplace.json` following the Claude Code plugin specification.

### Plugin Metadata Structure

```json
{
  "name": "plugin-name",
  "displayName": "Human Readable Name",
  "description": "Plugin description",
  "version": "1.0.0",
  "author": "Author Name",
  "category": "workflow|development|analysis",
  "keywords": ["keyword1", "keyword2"],
  "commands": ["command1", "command2"],
  "agents": ["agent1", "agent2"]
}
```

## 🔧 Plugin Management

### Check Installed Plugins

```bash
/plugin list
```

Shows all installed plugins with their status.

### Plugin Information

```bash
/plugin info <plugin-name>
```

Displays detailed information:
- Description
- Version
- Commands provided
- Agents included
- Author and keywords

### Update Plugins

Plugins are updated when you pull the latest repository changes:

```bash
git pull origin main
make install
```

### Uninstall Plugins

```bash
/plugin remove <plugin-name>
```

Or manually remove files:

```bash
# Remove commands
rm ~/.config/claude/commands/<command-name>.md

# Remove agents
rm ~/.config/claude/agents/<agent-name>.md
```

## 🎯 Plugin Selection Guide

### Install Everything (Recommended for New Users)

```bash
make install
```

Provides complete functionality with all workflows and commands.

### Selective Installation

**For Agile Teams**:
```bash
/plugin install bmad-agile-workflow
```

**For Rapid Development**:
```bash
/plugin install requirements-driven-workflow
/plugin install development-essentials
```

**For Individual Developers**:
```bash
/plugin install development-essentials
/plugin install advanced-ai-agents
```

**For Code Quality Focus**:
```bash
/plugin install development-essentials  # Includes /review
/plugin install bmad-agile-workflow     # Includes bmad-review
```

## 📁 Directory Structure

```
myclaude/
├── .claude-plugin/
│   └── marketplace.json          # Plugin registry
├── bmad-agile-workflow/
│   ├── commands/
│   │   └── bmad-pilot.md
│   └── agents/
│       ├── bmad-po.md
│       ├── bmad-architect.md
│       ├── bmad-sm.md
│       ├── bmad-dev.md
│       ├── bmad-review.md
│       ├── bmad-qa.md
│       └── bmad-orchestrator.md
├── requirements-driven-workflow/
│   ├── commands/
│   │   └── requirements-pilot.md
│   └── agents/
│       ├── requirements-generate.md
│       ├── requirements-code.md
│       ├── requirements-review.md
│       └── requirements-testing.md
├── development-essentials/
│   ├── commands/
│   │   ├── code.md
│   │   ├── debug.md
│   │   ├── test.md
│   │   └── ... (more commands)
│   └── agents/
│       ├── code.md
│       ├── bugfix.md
│       ├── debug.md
│       └── develop.md
├── advanced-ai-agents/
│   └── agents/
│       └── gpt5.md
└── output-styles/
    └── bmad-phase-context.md
```

## 🔄 Plugin Dependencies

**No Dependencies**: All plugins work independently

**Complementary Combinations**:
- BMAD + Advanced Agents (enhanced reviews)
- Requirements + Development Essentials (complete toolkit)
- All four plugins (full suite)

## 🛠️ Makefile Reference

```bash
# Installation
make install              # Install all plugins
make deploy-all          # Deploy all configurations

# Selective Deployment
make deploy-bmad         # BMAD workflow only
make deploy-requirements # Requirements workflow only
make deploy-commands     # All slash commands only
make deploy-agents       # All agents only

# Testing
make test-bmad          # Test BMAD workflow
make test-requirements  # Test Requirements workflow

# Cleanup
make clean              # Remove generated artifacts
make help               # Show all available commands
```

## 📚 Related Documentation

- **[BMAD Workflow](BMAD-WORKFLOW.md)** - Complete BMAD guide
- **[Requirements Workflow](REQUIREMENTS-WORKFLOW.md)** - Lightweight workflow guide
- **[Development Commands](DEVELOPMENT-COMMANDS.md)** - Command reference
- **[Quick Start Guide](QUICK-START.md)** - Get started quickly

## 🔗 External Resources

- **[Claude Code Plugin Docs](https://docs.claude.com/en/docs/claude-code/plugins)** - Official plugin documentation
- **[Claude Code CLI](https://claude.ai/code)** - Claude Code interface

---

**Modular Installation** - Install only what you need, when you need it.
