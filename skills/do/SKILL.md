---
name: do
description: This skill should be used for structured feature development with codebase understanding. Triggers on /do command. Provides a 7-phase workflow (Discovery, Exploration, Clarification, Architecture, Implementation, Review, Summary) using codeagent-wrapper to orchestrate code-explorer, code-architect, code-reviewer, and develop agents in parallel.
allowed-tools: ["Bash(${SKILL_DIR}/scripts/setup-do.sh:*)"]
---

# do - Feature Development Orchestrator

An orchestrator for systematic feature development. Invoke agents via `codeagent-wrapper`, never write code directly.

## Loop Initialization (REQUIRED)

When triggered via `/do <task>`, **first** initialize the loop state:

```bash
"${SKILL_DIR}/scripts/setup-do.sh" "<task description>"
```

This creates `.claude/do.{task_id}.local.md` with:
- `active: true`
- `current_phase: 1`
- `max_phases: 7`
- `completion_promise: "<promise>DO_COMPLETE</promise>"`

## Loop State Management

After each phase, update `.claude/do.{task_id}.local.md` frontmatter:
```yaml
current_phase: <next phase number>
phase_name: "<next phase name>"
```

When all 7 phases complete, output the completion signal:
```
<promise>DO_COMPLETE</promise>
```

To abort early, set `active: false` in the state file.

## Hard Constraints

1. **Never write code directly.** Delegate all code changes to `codeagent-wrapper` agents.
2. **Phase 3 (Clarification) is mandatory.** Do not proceed until questions are answered.
3. **Phase 5 (Implementation) requires explicit approval.** Stop after Phase 4 if not approved.
4. **Pass complete context forward.** Every agent invocation includes the Context Pack.
5. **Parallel-first.** Run independent tasks via `codeagent-wrapper --parallel`.
6. **Update state after each phase.** Keep `.claude/do.{task_id}.local.md` current.
7. **Expect long-running `codeagent-wrapper` calls.** High-reasoning modes (e.g. `xhigh`) can take a long time; stay in the orchestrator role and wait for agents to complete.
8. **Timeouts are not an escape hatch.** If a `codeagent-wrapper` invocation times out/errors, retry `codeagent-wrapper` (split/narrow the task if needed); never switch to direct implementation.

## Agents

| Agent | Purpose | Prompt |
|-------|---------|--------|
| `code-explorer` | Trace code, map architecture, find patterns | `agents/code-explorer.md` |
| `code-architect` | Design approaches, file plans, build sequences | `agents/code-architect.md` |
| `code-reviewer` | Review for bugs, simplicity, conventions | `agents/code-reviewer.md` |
| `develop` | Implement code, run tests | (uses global config) |

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

## 7-Phase Workflow

### Phase 1: Discovery

**Goal:** Understand what to build.

**Actions:**
1. Use AskUserQuestion for: user-visible behavior, scope, constraints, acceptance criteria
2. Invoke `code-architect` to draft requirements checklist and clarifying questions

```bash
codeagent-wrapper --agent code-architect - . <<'EOF'
## Original User Request
/do <request>

## Context Pack
- Code-explorer output: None
- Code-architect output: None

## Current Task
Produce requirements checklist and identify missing information.
Output: Requirements, Non-goals, Risks, Acceptance criteria, Questions (<= 10)

## Acceptance Criteria
Concrete, testable checklist; specific questions; no implementation.
EOF
```

### Phase 2: Exploration

**Goal:** Map codebase patterns and extension points.

**Actions:** Run 2-3 `code-explorer` tasks in parallel (similar features, architecture, tests/conventions).

```bash
codeagent-wrapper --parallel <<'EOF'
---TASK---
id: p2_similar_features
agent: code-explorer
workdir: .
---CONTENT---
## Original User Request
/do <request>

## Context Pack
- Code-architect output: <Phase 1 output>

## Current Task
Find 1-3 similar features, trace end-to-end. Return: key files with line numbers, call flow, extension points.

## Acceptance Criteria
Concrete file:line map + reuse points.

---TASK---
id: p2_architecture
agent: code-explorer
workdir: .
---CONTENT---
## Original User Request
/do <request>

## Context Pack
- Code-architect output: <Phase 1 output>

## Current Task
Map architecture for relevant subsystem. Return: module map + 5-10 key files.

## Acceptance Criteria
Clear boundaries; file:line references.

---TASK---
id: p2_conventions
agent: code-explorer
workdir: .
---CONTENT---
## Original User Request
/do <request>

## Context Pack
- Code-architect output: <Phase 1 output>

## Current Task
Identify testing patterns, conventions, config. Return: test commands + file locations.

## Acceptance Criteria
Test commands + relevant test file paths.
EOF
```

### Phase 3: Clarification (MANDATORY)

**Goal:** Resolve all ambiguities before design.

**Actions:**
1. Invoke `code-architect` to generate prioritized questions from Phase 1+2 outputs
2. Use AskUserQuestion to present questions and wait for answers
3. **Do not proceed until answered or defaults accepted**

### Phase 4: Architecture

**Goal:** Produce implementation plan fitting existing patterns.

**Actions:** Run 2 `code-architect` tasks in parallel (minimal-change vs pragmatic-clean).

```bash
codeagent-wrapper --parallel <<'EOF'
---TASK---
id: p4_minimal
agent: code-architect
workdir: .
---CONTENT---
## Original User Request
/do <request>

## Context Pack
- Code-explorer output: <ALL Phase 2 outputs>
- Code-architect output: <Phase 1 + Phase 3 answers>

## Current Task
Propose minimal-change architecture: reuse existing abstractions, minimize new files.
Output: file touch list, risks, edge cases.

## Acceptance Criteria
Concrete blueprint; minimal moving parts.

---TASK---
id: p4_pragmatic
agent: code-architect
workdir: .
---CONTENT---
## Original User Request
/do <request>

## Context Pack
- Code-explorer output: <ALL Phase 2 outputs>
- Code-architect output: <Phase 1 + Phase 3 answers>

## Current Task
Propose pragmatic-clean architecture: introduce seams for testability.
Output: file touch list, testing plan, risks.

## Acceptance Criteria
Implementable blueprint with build sequence and tests.
EOF
```

Use AskUserQuestion to let user choose approach.

### Phase 5: Implementation (Approval Required)

**Goal:** Build the feature.

**Actions:**
1. Use AskUserQuestion: "Approve starting implementation?" (Approve / Not yet)
2. If approved, invoke `develop`:

```bash
codeagent-wrapper --agent develop - . <<'EOF'
## Original User Request
/do <request>

## Context Pack
- Code-explorer output: <ALL Phase 2 outputs>
- Code-architect output: <selected Phase 4 blueprint + Phase 3 answers>

## Current Task
Implement with minimal change set following chosen architecture.
- Follow Phase 2 patterns
- Add/adjust tests per Phase 4 plan
- Run narrowest relevant tests

## Acceptance Criteria
Feature works end-to-end; tests pass; diff is minimal.
EOF
```

### Phase 6: Review

**Goal:** Catch defects and unnecessary complexity.

**Actions:** Run 2-3 `code-reviewer` tasks in parallel (correctness, simplicity).

```bash
codeagent-wrapper --parallel <<'EOF'
---TASK---
id: p6_correctness
agent: code-reviewer
workdir: .
---CONTENT---
## Original User Request
/do <request>

## Context Pack
- Code-architect output: <Phase 4 blueprint>
- Develop output: <Phase 5 output>

## Current Task
Review for correctness, edge cases, failure modes. Assume adversarial inputs.

## Acceptance Criteria
Issues with file:line references and concrete fixes.

---TASK---
id: p6_simplicity
agent: code-reviewer
workdir: .
---CONTENT---
## Original User Request
/do <request>

## Context Pack
- Code-architect output: <Phase 4 blueprint>
- Develop output: <Phase 5 output>

## Current Task
Review for KISS: remove bloat, collapse needless abstractions.

## Acceptance Criteria
Actionable simplifications with justification.
EOF
```

Use AskUserQuestion: Fix now / Fix later / Proceed as-is.

### Phase 7: Summary

**Goal:** Document what was built.

**Actions:** Invoke `code-reviewer` to produce summary:

```bash
codeagent-wrapper --agent code-reviewer - . <<'EOF'
## Original User Request
/do <request>

## Context Pack
- Code-architect output: <Phase 4 blueprint>
- Code-reviewer output: <Phase 6 outcomes>
- Develop output: <Phase 5 output + fixes>

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
