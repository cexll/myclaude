# do

7-phase feature development workflow, orchestrating multiple agents via codeagent-wrapper.

## Installation

```bash
python install.py --module do
```

Installs:
- `~/.claude/skills/do/` - skill files
- hooks auto-merged into `~/.claude/settings.json`

## Usage

```
/do <feature description>
```

Examples:
```
/do add user login feature
/do implement order export to CSV
```

## Workflow Phases

| Phase | Name | Goal |
|-------|------|------|
| 1 | Discovery | Understand requirements |
| 2 | Exploration | Explore codebase |
| 3 | Clarification | Resolve ambiguities (mandatory) |
| 4 | Architecture | Design approach |
| 5 | Implementation | Build (requires approval) |
| 6 | Review | Code review |
| 7 | Summary | Document results |

## Agents

- `code-explorer` - Code tracing, architecture mapping
- `code-architect` - Design approaches, file planning
- `code-reviewer` - Code review, simplification suggestions
- `develop` - Implement code, run tests

Agent prompts are in the `agents/` directory. To customize, create same-named files in `~/.codeagent/agents/` to override.

## ~/.codeagent/models.json Configuration

Optional. Uses codeagent-wrapper built-in config by default. To customize agent models:

```json
{
  "agents": {
    "code-explorer": {
      "backend": "claude",
      "model": "claude-sonnet-4-5-20250929"
    },
    "code-architect": {
      "backend": "claude",
      "model": "claude-sonnet-4-5-20250929"
    },
    "code-reviewer": {
      "backend": "claude",
      "model": "claude-sonnet-4-5-20250929"
    }
  }
}
```

## Loop Mechanism

A Stop hook is registered after installation. When `/do` runs:

1. Creates `.claude/do.local.md` state file
2. Updates `current_phase` after each phase
3. Stop hook checks state, blocks exit if incomplete
4. Outputs `<promise>DO_COMPLETE</promise>` when finished

Manual exit: Set `active` to `false` in the state file.

## Uninstall

```bash
python install.py --uninstall --module do
```
