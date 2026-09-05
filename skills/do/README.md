# do - Feature Development Orchestrator

5-phase feature development workflow orchestrating multiple agents via codeagent-wrapper.

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

## 5-Phase Workflow

| Phase | Name | Goal | Key Actions |
|-------|------|------|-------------|
| 1 | Understand | Gather requirements | AskUserQuestion + code-explorer analysis |
| 2 | Clarify | Resolve ambiguities | **MANDATORY** - must answer before proceeding |
| 3 | Design | Plan implementation | code-architect approaches |
| 4 | Implement | Build the feature | **Requires approval** - develop agent |
| 5 | Complete | Finalize and document | code-reviewer summary |

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
2. **Phase 2 is mandatory** - do not proceed until questions are answered
3. **Phase 4 requires approval** - stop after Phase 3 if not approved
4. **Pass complete context forward** - every agent gets the Context Pack
5. **Parallel-first** - run independent tasks via `codeagent-wrapper --parallel`
6. **Update state after each phase** - keep `.claude/do-tasks/{task_id}/task.json` current

## Context Pack Template

```text
## Original User Request
<verbatim request>

## Context Pack
- Phase: <1-5 name>
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

When triggered via `/do <task>`, initializes `.claude/do-tasks/{task_id}/task.md` with YAML frontmatter:
```yaml
---
id: "<task_id>"
title: "<task description>"
status: "in_progress"
current_phase: 1
phase_name: "Understand"
max_phases: 5
use_worktree: false
created_at: "<ISO timestamp>"
completion_promise: "<promise>DO_COMPLETE</promise>"
---

# Requirements

<task description>

## Context

## Progress
```

The current task is tracked in `.claude/do-tasks/.current-task`.

After each phase, update `task.md` frontmatter via:
```bash
python3 ".claude/skills/do/scripts/task.py" update-phase <N>
```

When all 5 phases complete, output:
```
<promise>DO_COMPLETE</promise>
```

To abort early, manually edit `task.md` and set `status: "cancelled"` in the frontmatter.

## Stop Hook

A Stop hook is registered after installation:
1. Creates `.claude/do-tasks/{task_id}/task.md` state file
2. Updates `current_phase` in frontmatter after each phase
3. Stop hook checks state, blocks exit if incomplete
4. Outputs `<promise>DO_COMPLETE</promise>` when finished

Manual exit: Edit `task.md` and set `status: "cancelled"` in the frontmatter.

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

## Worktree Mode

Use `--worktree` to execute tasks in an isolated git worktree, preventing changes to your main branch:

```bash
codeagent-wrapper --worktree --agent develop "implement feature X" .
```

This automatically:
1. Generates a unique task ID (format: `YYYYMMDD-xxxxxx`)
2. Creates a new worktree at `.worktrees/do-{task_id}/`
3. Creates a new branch `do/{task_id}`
4. Executes the task in the isolated worktree

Output includes: `Using worktree: .worktrees/do-{task_id}/ (task_id: {id}, branch: do/{id})`

In parallel mode, add `worktree: true` to task blocks:
```
---TASK---
id: feature_impl
agent: develop
worktree: true
---CONTENT---
Implement the feature
```
