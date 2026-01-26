# do - Feature Development Orchestrator

7-phase feature development workflow orchestrating multiple agents via codeagent-wrapper.

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

## 7-Phase Workflow

| Phase | Name | Goal | Key Actions |
|-------|------|------|-------------|
| 1 | Discovery | Understand requirements | AskUserQuestion + code-architect draft |
| 2 | Exploration | Map codebase patterns | 2-3 parallel code-explorer tasks |
| 3 | Clarification | Resolve ambiguities | **MANDATORY** - must answer before proceeding |
| 4 | Architecture | Design implementation | 2 parallel code-architect approaches |
| 5 | Implementation | Build the feature | **Requires approval** - develop agent |
| 6 | Review | Catch defects | 2-3 parallel code-reviewer tasks |
| 7 | Summary | Document results | code-reviewer summary |

## Agents

| Agent | Purpose | Prompt Location |
|-------|---------|----------------|
| `code-explorer` | Code tracing, architecture mapping | `agents/code-explorer.md` |
| `code-architect` | Design approaches, file planning | `agents/code-architect.md` |
| `code-reviewer` | Code review, simplification | `agents/code-reviewer.md` |
| `develop` | Implement code, run tests | global config |

To customize agents, create same-named files in `~/.codeagent/agents/` to override.

## Hard Constraints

1. **Never write code directly** - delegate all changes to codeagent-wrapper agents
2. **Phase 3 is mandatory** - do not proceed until questions are answered
3. **Phase 5 requires approval** - stop after Phase 4 if not approved
4. **Pass complete context forward** - every agent gets the Context Pack
5. **Parallel-first** - run independent tasks via `codeagent-wrapper --parallel`
6. **Update state after each phase** - keep `.claude/do.{task_id}.local.md` current

## Context Pack Template

```text
## Original User Request
<verbatim request>

## Context Pack
- Phase: <1-7 name>
- Decisions: <requirements/constraints/choices>
- Code-explorer output: <paste or "None">
- Code-architect output: <paste or "None">
- Code-reviewer output: <paste or "None">
- Develop output: <paste or "None">
- Open questions: <list or "None">

## Current Task
<specific task>

## Acceptance Criteria
<checkable outputs>
```

## Loop State Management

When triggered via `/do <task>`, initializes `.claude/do.{task_id}.local.md` with:
- `active: true`
- `current_phase: 1`
- `max_phases: 7`
- `completion_promise: "<promise>DO_COMPLETE</promise>"`

After each phase, update frontmatter:
```yaml
current_phase: <next phase number>
phase_name: "<next phase name>"
```

When all 7 phases complete, output:
```
<promise>DO_COMPLETE</promise>
```

To abort early, set `active: false` in the state file.

## Stop Hook

A Stop hook is registered after installation:
1. Creates `.claude/do.{task_id}.local.md` state file
2. Updates `current_phase` after each phase
3. Stop hook checks state, blocks exit if incomplete
4. Outputs `<promise>DO_COMPLETE</promise>` when finished

Manual exit: Set `active` to `false` in the state file.

## Parallel Execution Examples

### Phase 2: Exploration (3 parallel tasks)
```bash
codeagent-wrapper --parallel <<'EOF'
---TASK---
id: p2_similar_features
agent: code-explorer
workdir: .
---CONTENT---
Find similar features, trace end-to-end.

---TASK---
id: p2_architecture
agent: code-explorer
workdir: .
---CONTENT---
Map architecture for relevant subsystem.

---TASK---
id: p2_conventions
agent: code-explorer
workdir: .
---CONTENT---
Identify testing patterns and conventions.
EOF
```

### Phase 4: Architecture (2 approaches)
```bash
codeagent-wrapper --parallel <<'EOF'
---TASK---
id: p4_minimal
agent: code-architect
workdir: .
---CONTENT---
Propose minimal-change architecture.

---TASK---
id: p4_pragmatic
agent: code-architect
workdir: .
---CONTENT---
Propose pragmatic-clean architecture.
EOF
```

## ~/.codeagent/models.json Configuration

Required when using `agent:` in parallel tasks or `--agent`. Create `~/.codeagent/models.json` to configure agent → backend/model mappings:

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

## Uninstall

```bash
python install.py --uninstall --module do
```
