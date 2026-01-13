# OmO Multi-Agent Orchestration

OmO (Oh-My-OpenCode) is a multi-agent orchestration skill that uses Sisyphus as the primary coordinator to delegate tasks to specialized agents.

## Quick Start

```
/omo <your task>
```

## Agent Hierarchy

| Agent | Role | Backend | Model |
|-------|------|---------|-------|
| sisyphus | Primary orchestrator | claude | claude-sonnet-4-20250514 |
| oracle | Technical advisor (EXPENSIVE) | claude | claude-sonnet-4-20250514 |
| librarian | External research | claude | claude-sonnet-4-5-20250514 |
| explore | Codebase search (FREE) | opencode | opencode/grok-code |
| develop | Code implementation | codex | (default) |
| frontend-ui-ux-engineer | UI/UX specialist | gemini | gemini-3-pro-preview |
| document-writer | Documentation | gemini | gemini-3-flash-preview |

## How It Works

1. `/omo` loads Sisyphus as the entry point
2. Sisyphus analyzes your request via routing signals
3. Based on task type, Sisyphus either:
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

Sisyphus delegates via codeagent-wrapper with full Context Pack:

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
  "default_backend": "opencode",
  "default_model": "opencode/grok-code",
  "agents": {
    "sisyphus": {"backend": "claude", "model": "claude-sonnet-4-20250514"},
    "oracle": {"backend": "claude", "model": "claude-sonnet-4-20250514"}
  }
}
```

## Requirements

- codeagent-wrapper with `--agent` support
- Backend CLIs: claude, opencode, gemini
