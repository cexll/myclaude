# OmO Multi-Agent Orchestration

OmO (Oh-My-OpenCode) is a multi-agent orchestration skill that delegates tasks to specialized agents based on routing signals.

## Installation

```bash
python3 install.py --module omo
```

## Quick Start

```
/omo <your task>
```

## Agent Hierarchy

| Agent | Role | Backend | Model |
|-------|------|---------|-------|
| oracle | Technical advisor | claude | claude-opus-4-5-20251101 |
| librarian | External research | claude | claude-sonnet-4-5-20250929 |
| explore | Codebase search | opencode | opencode/grok-code |
| develop | Code implementation | codex | gpt-5.2 |
| frontend-ui-ux-engineer | UI/UX specialist | gemini | gemini-3-pro-high |
| document-writer | Documentation | gemini | gemini-3-flash |

## How It Works

1. `/omo` analyzes your request via routing signals
2. Based on task type, it either:
   - Answers directly (analysis/explanation tasks - no code changes)
   - Delegates to specialized agents (implementation tasks)
   - Fires parallel agents (exploration + research)

## Examples

```bash
# Refactoring
/omo Help me refactor this authentication module

# Feature development
/omo I need to add a new payment feature with frontend UI and backend API

# Research
/omo What authentication scheme does this project use?
```

## Agent Delegation

Delegates via codeagent-wrapper with full Context Pack:

```bash
codeagent-wrapper --agent oracle - . <<'EOF'
## Original User Request
Analyze the authentication architecture and recommend improvements.

## Context Pack (include anything relevant; write "None" if absent)
- Explore output: [paste explore output if available]
- Librarian output: None
- Oracle output: None

## Current Task
Review auth architecture, identify risks, propose minimal improvements.

## Acceptance Criteria
Output: recommendation, action plan, risk assessment, effort estimate.
EOF
```

## Configuration

Agent-model mappings are configured in `~/.codeagent/models.json`:

```json
{
  "default_backend": "codex",
  "default_model": "gpt-5.2",
  "agents": {
    "oracle": {
      "backend": "claude",
      "model": "claude-opus-4-5-20251101",
      "description": "Technical advisor",
      "yolo": true
    },
    "librarian": {
      "backend": "claude",
      "model": "claude-sonnet-4-5-20250929",
      "description": "Researcher",
      "yolo": true
    },
    "explore": {
      "backend": "opencode",
      "model": "opencode/grok-code",
      "description": "Code search"
    },
    "frontend-ui-ux-engineer": {
      "backend": "gemini",
      "model": "gemini-3-pro-preview",
      "description": "Frontend engineer"
    },
    "document-writer": {
      "backend": "gemini",
      "model": "gemini-3-flash-preview",
      "description": "Documentation"
    },
    "develop": {
      "backend": "codex",
      "model": "gpt-5.2",
      "description": "codex develop",
      "yolo": true,
      "reasoning": "xhigh"
    }
  }
}
```

## Requirements

- codeagent-wrapper with `--agent` support
- Backend CLIs: claude, opencode, codex, gemini
