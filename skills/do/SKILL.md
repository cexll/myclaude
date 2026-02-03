---
name: do
description: This skill should be used for structured feature development with codebase understanding. Triggers on /do command. Provides a 5-phase workflow (Understand, Clarify, Design, Implement, Complete) using codeagent-wrapper to orchestrate code-explorer, code-architect, code-reviewer, and develop agents in parallel.
allowed-tools: ["Bash(${SKILL_DIR}/scripts/setup-do.py:*)"]
---

# do - Feature Development Orchestrator

An orchestrator for systematic feature development. Invoke agents via `codeagent-wrapper`, never write code directly.

## Loop Initialization (REQUIRED)

When triggered via `/do <task>`, follow these steps:

### Step 1: Ask about worktree mode

Use AskUserQuestion to ask:

```
Develop in a separate worktree? (Isolates changes from main branch)
- Yes (Recommended for larger changes)
- No (Work directly in current directory)
```

### Step 2: Initialize state

```bash
# If worktree mode selected:
python3 "${SKILL_DIR}/scripts/setup-do.py" --worktree "<task description>"

# If no worktree:
python3 "${SKILL_DIR}/scripts/setup-do.py" "<task description>"
```

This creates `.claude/do.{task_id}.local.md` with:
- `active: true`
- `current_phase: 1`
- `max_phases: 5`
- `completion_promise: "<promise>DO_COMPLETE</promise>"`
- `use_worktree: true/false`

## Worktree Mode

When `use_worktree: true` in state file, ALL `codeagent-wrapper` calls that modify code MUST include `--worktree`:

```bash
# With worktree mode enabled
codeagent-wrapper --worktree --agent develop - . <<'EOF'
...
EOF

# Parallel tasks with worktree
codeagent-wrapper --worktree --parallel <<'EOF'
---TASK---
id: task1
agent: develop
workdir: .
---CONTENT---
...
EOF
```

The `--worktree` flag tells codeagent-wrapper to create/use a worktree internally. Read-only agents (code-explorer, code-architect, code-reviewer) do NOT need `--worktree`.

## Loop State Management

After each phase, update `.claude/do.{task_id}.local.md` frontmatter:
```yaml
current_phase: <next phase number>
phase_name: "<next phase name>"
```

When all 5 phases complete, output the completion signal:
```
<promise>DO_COMPLETE</promise>
```

To abort early, set `active: false` in the state file.

## Hard Constraints

1. **Never write code directly.** Delegate all code changes to `codeagent-wrapper` agents.
2. **Pass complete context forward.** Every agent invocation includes the Context Pack.
3. **Parallel-first.** Run independent tasks via `codeagent-wrapper --parallel`.
4. **Update state after each phase.** Keep `.claude/do.{task_id}.local.md` current.
5. **Expect long-running `codeagent-wrapper` calls.** High-reasoning modes can take a long time; stay in the orchestrator role and wait for agents to complete.
6. **Timeouts are not an escape hatch.** If a `codeagent-wrapper` invocation times out/errors, retry (split/narrow the task if needed); never switch to direct implementation.
7. **Respect worktree setting.** If `use_worktree: true`, always pass `--worktree` to develop agent calls.

## Agents

| Agent | Purpose | Needs --worktree |
|-------|---------|------------------|
| `code-explorer` | Trace code, map architecture, find patterns | No (read-only) |
| `code-architect` | Design approaches, file plans, build sequences | No (read-only) |
| `code-reviewer` | Review for bugs, simplicity, conventions | No (read-only) |
| `develop` | Implement code, run tests | **Yes** (if worktree enabled) |

## Issue Severity Definitions

**Blocking issues** (require user input):
- Impacts core functionality or correctness
- Security vulnerabilities
- Architectural conflicts with existing patterns
- Ambiguous requirements with multiple valid interpretations

**Minor issues** (auto-fix without asking):
- Code style inconsistencies
- Naming improvements
- Missing documentation
- Non-critical test coverage gaps

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

## 5-Phase Workflow

### Phase 1: Understand (Parallel, No Interaction)

**Goal:** Understand requirements and map codebase simultaneously.

**Actions:** Run `code-architect` and 2-3 `code-explorer` tasks in parallel.

```bash
codeagent-wrapper --parallel <<'EOF'
---TASK---
id: p1_requirements
agent: code-architect
workdir: .
---CONTENT---
## Original User Request
/do <request>

## Context Pack
- Code-explorer output: None
- Code-architect output: None

## Current Task
1. Analyze requirements completeness (score 1-10)
2. Extract explicit requirements, constraints, acceptance criteria
3. Identify blocking questions (issues that prevent implementation)
4. Identify minor clarifications (nice-to-have but can proceed without)

Output format:
- Completeness score: X/10
- Requirements: [list]
- Non-goals: [list]
- Blocking questions: [list, if any]
- Minor clarifications: [list, if any]

## Acceptance Criteria
Concrete checklist; blocking vs minor questions clearly separated.

---TASK---
id: p1_similar_features
agent: code-explorer
workdir: .
---CONTENT---
## Original User Request
/do <request>

## Current Task
Find 1-3 similar features, trace end-to-end. Return: key files with line numbers, call flow, extension points.

## Acceptance Criteria
Concrete file:line map + reuse points.

---TASK---
id: p1_architecture
agent: code-explorer
workdir: .
---CONTENT---
## Original User Request
/do <request>

## Current Task
Map architecture for relevant subsystem. Return: module map + 5-10 key files.

## Acceptance Criteria
Clear boundaries; file:line references.

---TASK---
id: p1_conventions
agent: code-explorer
workdir: .
---CONTENT---
## Original User Request
/do <request>

## Current Task
Identify testing patterns, conventions, config. Return: test commands + file locations.

## Acceptance Criteria
Test commands + relevant test file paths.
EOF
```

### Phase 2: Clarify (Conditional)

**Goal:** Resolve blocking ambiguities only.

**Actions:**
1. Review `p1_requirements` output for blocking questions
2. **IF blocking questions exist** → Use AskUserQuestion
3. **IF no blocking questions (completeness >= 8)** → Skip to Phase 3, log "Requirements clear, proceeding"

```bash
# Only if blocking questions exist:
# Use AskUserQuestion with the blocking questions from Phase 1
```

### Phase 3: Design (No Interaction)

**Goal:** Produce minimal-change implementation plan.

**Actions:** Invoke `code-architect` with all Phase 1 context to generate a single implementation plan.

```bash
codeagent-wrapper --agent code-architect - . <<'EOF'
## Original User Request
/do <request>

## Context Pack
- Code-explorer output: <ALL Phase 1 explorer outputs>
- Code-architect output: <Phase 1 requirements + Phase 2 answers if any>

## Current Task
Design minimal-change implementation:
- Reuse existing abstractions
- Minimize new files
- Follow established patterns from code-explorer output

Output:
- File touch list with specific changes
- Build sequence
- Test plan
- Risks and mitigations

## Acceptance Criteria
Concrete, implementable blueprint with minimal moving parts.
EOF
```

### Phase 4: Implement + Review (Single Interaction Point)

**Goal:** Build feature and review in one phase.

**Actions:**

1. Invoke `develop` to implement (add `--worktree` if `use_worktree: true`):

```bash
# Check use_worktree from state file, add --worktree if true
codeagent-wrapper --worktree --agent develop - . <<'EOF'
## Original User Request
/do <request>

## Context Pack
- Code-explorer output: <ALL Phase 1 outputs>
- Code-architect output: <Phase 3 blueprint>

## Current Task
Implement with minimal change set following the blueprint.
- Follow Phase 1 patterns
- Add/adjust tests per Phase 3 plan
- Run narrowest relevant tests

## Acceptance Criteria
Feature works end-to-end; tests pass; diff is minimal.
EOF
```

2. Run parallel reviews (no --worktree needed, read-only):

```bash
codeagent-wrapper --parallel <<'EOF'
---TASK---
id: p4_correctness
agent: code-reviewer
workdir: .
---CONTENT---
## Original User Request
/do <request>

## Context Pack
- Code-architect output: <Phase 3 blueprint>
- Develop output: <implementation output>

## Current Task
Review for correctness, edge cases, failure modes.
Classify each issue as BLOCKING or MINOR.

## Acceptance Criteria
Issues with file:line references, severity, and concrete fixes.

---TASK---
id: p4_simplicity
agent: code-reviewer
workdir: .
---CONTENT---
## Original User Request
/do <request>

## Context Pack
- Code-architect output: <Phase 3 blueprint>
- Develop output: <implementation output>

## Current Task
Review for KISS: remove bloat, collapse needless abstractions.
Classify each issue as BLOCKING or MINOR.

## Acceptance Criteria
Actionable simplifications with severity and justification.
EOF
```

3. Handle review results:
   - **MINOR issues only** → Auto-fix via `develop` (with `--worktree` if enabled), no user interaction
   - **BLOCKING issues** → Use AskUserQuestion: "Fix now / Proceed as-is"

### Phase 5: Complete (No Interaction)

**Goal:** Document what was built.

**Actions:** Invoke `code-reviewer` to produce summary:

```bash
codeagent-wrapper --agent code-reviewer - . <<'EOF'
## Original User Request
/do <request>

## Context Pack
- Code-architect output: <Phase 3 blueprint>
- Code-reviewer output: <Phase 4 review outcomes>
- Develop output: <Phase 4 implementation + fixes>

## Current Task
Write completion summary:
- What was built
- Key decisions/tradeoffs
- Files modified (paths)
- How to verify (commands)
- Follow-ups (optional)

## Acceptance Criteria
Short, technical, actionable summary.
EOF
```

Output the completion signal:
```
<promise>DO_COMPLETE</promise>
```
