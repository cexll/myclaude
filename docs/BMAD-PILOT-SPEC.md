#!/usr/bin/env markdown
# `/bmad-pilot` Orchestrated Workflow and Agent Spec

> Enterprise-grade, role-based agile workflow with approval gates and quality thresholds. This spec consolidates the end-to-end logic of the BMAD pilot command and maps each phase to its agent contracts.

## 1. Overview

- Purpose: Automate a complete agile cycle with specialized agents: Product Owner (PO), Architect, Scrum Master (SM), Developer (Dev), Independent Reviewer, and QA.
- Principles: Quality gates at design points, explicit user approvals, repository-aware context, iterative refinement.
- Artifacts: Saved under `./.claude/specs/{feature_name}/`.
- Options:
  - `--skip-tests`: skip QA phase.
  - `--direct-dev`: skip SM planning; go from Architecture → Development.
  - `--skip-scan`: skip repository scan (not recommended).

## 2. Command

```
/bmad-pilot <PROJECT_DESCRIPTION> [--skip-tests] [--direct-dev] [--skip-scan]
```

Input handling
- Generate `feature_name` (kebab-case) from `<PROJECT_DESCRIPTION>`.
- Ensure directory `./.claude/specs/{feature_name}/` exists.
- If input > 500 chars: summarize and confirm with user before proceeding.
- If input unclear: ask targeted clarification questions.

## 3. Phases and Approval Gates

### Phase 0 — Repository Scan (auto unless `--skip-scan`)
- Goal: Build context on tech stack, structure, conventions, dependencies, CI, tests.
- Output: `00-repo-scan.md`.

### Phase 1 — Product Requirements (PO)
- Loop with quality scoring until PRD ≥ 90.
- Output (after approval): `01-product-requirements.md`.
- Gate #1: User must approve saving PRD and proceeding.

### Phase 2 — System Architecture (Architect)
- Loop with quality scoring until Architecture ≥ 90.
- Output (after approval): `02-system-architecture.md`.
- Gate #2: User must approve saving design and proceeding.

### Phase 3 — Sprint Planning (SM) [skipped by `--direct-dev`]
- Interactive plan: stories, tasks, estimates, risks; iterated to actionable clarity.
- Output (after approval): `03-sprint-plan.md`.
- Gate #3: User must approve plan before development.

### Phase 4 — Development (Dev)
- Implement according to PRD/Architecture/(Sprint Plan).
- Output: code changes in repo.

### Phase 4.5 — Code Review (Independent Reviewer)
- Output: `04-dev-reviewed.md` with status: Pass / Pass with Risk / Fail.
- Loop: If Fail, return to Dev for fixes, then re-review.

### Phase 5 — Quality Assurance (QA) [skipped by `--skip-tests`]
- Create and execute tests aligned to PRD and Architecture.
- Output: QA execution results (reporting within agent and/or CI output).

## 4. Agent Roles and Contracts

Each agent consumes prior artifacts and repository context from `00-repo-scan.md`.

- `bmad-po` (Product Owner)
  - Input: project description, repo context.
  - Output: PRD (≥ 90) → `01-product-requirements.md` (save only after user approval).
  - Interaction: asks targeted questions; orchestrator mediates the loop.

- `bmad-architect` (System Architect)
  - Input: PRD, repo context.
  - Output: Architecture (≥ 90) → `02-system-architecture.md` (save only after approval).
  - Interaction: clarifies technical decisions; orchestrator mediates.

- `bmad-sm` (Scrum Master)
  - Input: PRD, Architecture, repo context.
  - Output: Sprint plan → `03-sprint-plan.md` (after approval); skipped with `--direct-dev`.

- `bmad-dev` (Developer)
  - Input: PRD, Architecture, (Sprint Plan), repo context.
  - Output: Working implementation with tests as appropriate.

- `bmad-review` (Independent Reviewer)
  - Input: implementation + all specs.
  - Output: `04-dev-reviewed.md` with Pass/Risk/Fail; feeds back into Dev.

- `bmad-qa` (QA Engineer)
  - Input: implementation + all specs.
  - Output: executed test suite and results; ensures acceptance criteria.

- `bmad-orchestrator`
  - Input: user intent; coordinates all phases; manages approval gates; ensures saves happen only after approval.

## 5. Quality and Gates

- PRD Quality (≥ 90) → proceed to Architecture.
- Architecture Quality (≥ 90) → proceed to SM or Dev (`--direct-dev`).
- Sprint Plan → approval gate before Dev.
- Review Status:
  - Pass → proceed to QA (unless `--skip-tests`).
  - Pass with Risk → optional follow-up.
  - Fail → return to Dev for fixes and re-review.

## 6. Artifacts

Saved under `./.claude/specs/{feature_name}/`:
```
00-repo-scan.md
01-product-requirements.md
02-system-architecture.md
03-sprint-plan.md            # if not skipped
04-dev-reviewed.md
```

## 7. Execution Logic (Simplified)

```pseudo
parse_options()
feature = to_kebab_case(PROJECT_DESCRIPTION)
ensure_specs_dir(feature)

if not --skip-scan:
  scan_repo() -> write 00-repo-scan.md

prd_score = 0
while prd_score < 90:
  prd, prd_score = po_iterate()
user_approve_or_loop()
save('01-product-requirements.md', prd)

arch_score = 0
while arch_score < 90:
  arch, arch_score = architect_iterate()
user_approve_or_loop()
save('02-system-architecture.md', arch)

if not --direct-dev:
  sprint_plan = sm_iterate_until_actionable()
  user_approve_or_loop()
  save('03-sprint-plan.md', sprint_plan)

develop()
status = review()
while status == 'Fail':
  develop_fix()
  status = review()

if not --skip-tests:
  qa_execute()
finish()
```

## 8. When to Use `/bmad-pilot` vs `/alin-dev`

Key differences
- Scope & Rigor: BMAD is a full agile process with 6 role-based agents and 3 approval gates (PRD, Architecture, Sprint Plan). alin-dev is an implementation-first, lighter workflow.
- Artifacts Path: BMAD writes to `./.claude/specs/{feature_name}/`; alin-dev writes to `./.alin/specs/{feature_name}/`.
- Phases: BMAD includes PO + Architect + SM before Dev; alin-dev uses `alin-generate` spec → `alin-code` → `alin-review` and optionally `alin-manual-validate` and `alin-testing`.
- Extra Deliverable: alin-dev generates a manual validation guide by default (`requirements-manual-valid.md`), BMAD does not.
- Options: BMAD has `--direct-dev` to skip SM; alin-dev has `--skip-manual` to skip the validation guide.

Recommended usage
- Choose `/bmad-pilot` when:
  - Requirements are complex, multi-stakeholder, or ambiguous.
  - Architecture decisions carry material risk.
  - You need sprint planning and formal approval gates.
  - Enterprise/cross-team alignment and documentation are priorities.
- Choose `/alin-dev` when:
  - The task is well understood and speed to implementation matters.
  - You want a single, concrete technical spec driving code.
  - You prefer a validation guide to hand-test changes end-to-end.
  - The feature is moderate in scope and does not require a separate Architect/SM phase.

Quick decision guide
```
If “complex + uncertain + high risk” → /bmad-pilot
If “clear + fast delivery + practical validation” → /alin-dev
```

## 9. Notes and Best Practices

- Do not skip repository scan unless you already know the project context.
- Use approval gates to enforce quality and stakeholder alignment.
- Keep artifacts up-to-date; each phase should reflect the latest decisions.
- Prefer small, iterative loops within each phase to raise quality before saving.

---

References
- Command source: `bmad-agile-workflow/commands/bmad-pilot.md`
- Workflow guide: `docs/BMAD-WORKFLOW.md`
- alin-dev spec: `alin-dev-workflow/commands/alin-dev.md`
