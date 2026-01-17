---
name: sparv
description: Minimal SPARV workflow (Specify→Plan→Act→Review→Vault) with 10-point spec gate, unified journal, 2-action saves, 3-failure protocol, and EHRB risk detection.
---

# SPARV

Five-phase workflow: **S**pecify → **P**lan → **A**ct → **R**eview → **V**ault.

Goal: Complete "requirements → verifiable delivery" in one pass, recording key decisions in external memory instead of relying on assumptions.

## Core Rules (Mandatory)

- **10-Point Specify Gate**: Spec score `0-10`; must be `>=9` to enter Plan.
- **2-Action Save**: Append an entry to `.sparv/journal.md` every 2 tool calls.
- **3-Failure Protocol**: Stop and escalate to user after 3 consecutive failures.
- **EHRB**: Require explicit user confirmation when high-risk detected (production/sensitive data/destructive/billing API/security-critical).
- **Fixed Phase Names**: `specify|plan|act|review|vault` (stored in `.sparv/state.yaml:current_phase`).

## Enhanced Rules (v1.1)

### Uncertainty Declaration (G3)

When any Specify dimension scores < 2:
- Declare: `UNCERTAIN: <what> | ASSUMPTION: <fallback>`
- List all assumptions in journal before Plan
- Offer 2-3 options for ambiguous requirements

Example:
```
UNCERTAIN: deployment target | ASSUMPTION: Docker container
UNCERTAIN: auth method | OPTIONS: JWT / OAuth2 / Session
```

### Requirement Routing

| Mode | Condition | Flow |
|------|-----------|------|
| **Quick** | score >= 9 AND <= 3 files AND no EHRB | Specify → Act → Review |
| **Full** | otherwise | Specify → Plan → Act → Review → Vault |

Quick mode skips formal Plan phase but still requires:
- Completion promise written to journal
- 2-action save rule applies
- Review phase mandatory

### Context Acquisition (Optional)

Before Specify scoring:
1. Check `.sparv/kb.md` for existing patterns/decisions
2. If insufficient, scan codebase for relevant files
3. Document findings in journal under `## Context`

Skip if user explicitly provides full context.

### Knowledge Base Maintenance

During Vault phase, update `.sparv/kb.md`:
- **Patterns**: Reusable code patterns discovered
- **Decisions**: Architectural choices + rationale
- **Gotchas**: Common pitfalls + solutions

### CHANGELOG Update

Use during Review or Vault phase for non-trivial changes:
```bash
~/.claude/skills/sparv/scripts/changelog-update.sh --type <Added|Changed|Fixed|Removed> --desc "..."
```

## External Memory (Two Files)

Initialize (run in project root):

```bash
~/.claude/skills/sparv/scripts/init-session.sh --force
```

File conventions:

- `.sparv/state.yaml`: State machine (minimum fields: `session_id/current_phase/action_count/consecutive_failures`)
- `.sparv/journal.md`: Unified log (Plan/Progress/Findings all go here)
- `.sparv/history/<session_id>/`: Archive directory

## Phase 1: Specify (10-Point Scale)

Each item scores 0/1/2, total 0-10:

1) **Value**: Why do it, are benefits/metrics verifiable
2) **Scope**: MVP + what's out of scope
3) **Acceptance**: Testable acceptance criteria
4) **Boundaries**: Error/performance/compatibility/security critical boundaries
5) **Risk**: EHRB/dependencies/unknowns + handling approach

`score < 9`: Keep asking questions; do not enter Plan.
`score >= 9`: Write a clear `completion_promise` (verifiable completion commitment), then enter Plan.

## Phase 2: Plan

- Break into atomic tasks (2-5 minute granularity), each with a verifiable output/test point.
- Write the plan to `.sparv/journal.md` (Plan section or append directly).

## Phase 3: Act

- **TDD Rule**: No failing test → no production code.
- Auto-write journal every 2 actions (PostToolUse hook).
- Failure counting (3-Failure Protocol):

```bash
~/.claude/skills/sparv/scripts/failure-tracker.sh fail --note "short blocker"
~/.claude/skills/sparv/scripts/failure-tracker.sh reset
```

## Phase 4: Review

- Two stages: Spec conformance → Code quality (correctness/performance/security/tests).
- Maximum 3 fix rounds; escalate to user if exceeded.

Run 3-question reboot test before session ends:

```bash
~/.claude/skills/sparv/scripts/reboot-test.sh --strict
```

## Phase 5: Vault

Archive current session:

```bash
~/.claude/skills/sparv/scripts/archive-session.sh
```

## Script Tools

| Script | Purpose |
|--------|---------|
| `scripts/init-session.sh` | Initialize `.sparv/`, generate `state.yaml` + `journal.md` |
| `scripts/save-progress.sh` | Maintain `action_count`, append to `journal.md` every 2 actions |
| `scripts/check-ehrb.sh` | Scan diff/text, output (optionally write) `ehrb_flags` |
| `scripts/failure-tracker.sh` | Maintain `consecutive_failures`, exit code 3 when reaching 3 |
| `scripts/reboot-test.sh` | 3-question self-check (optional strict mode) |
| `scripts/archive-session.sh` | Archive `journal.md` + `state.yaml` to `history/` |

## Auto Hooks

`hooks/hooks.json`:

- PostToolUse: `save-progress.sh` (2-Action save)
- PreToolUse: `check-ehrb.sh --diff --dry-run` (prompt only, no state write)
- Stop: `reboot-test.sh --strict` (3-question self-check)

---

*Quality over speed—iterate until truly complete.*
