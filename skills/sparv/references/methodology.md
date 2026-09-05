# SPARV Methodology (Short)

This document is a quick reference; the canonical spec is in `SKILL.md`.

## Five Phases

- **Specify**: Write requirements as verifiable specs (10-point gate)
- **Plan**: Break into atomic tasks (2-5 minute granularity)
- **Act**: TDD-driven implementation; write to journal every 2 actions
- **Review**: Spec conformance → Code quality; maximum 3 fix rounds
- **Vault**: Archive session (state + journal)

## Enhanced Rules (v1.1)

### Uncertainty Declaration (G3)

When any Specify dimension scores < 2:
- Declare: `UNCERTAIN: <what> | ASSUMPTION: <fallback>`
- List all assumptions in journal before Plan
- Offer 2-3 options for ambiguous requirements

### Requirement Routing

| Mode | Condition | Flow |
|------|-----------|------|
| **Quick** | score >= 9 AND <= 3 files AND no EHRB | Specify → Act → Review |
| **Full** | otherwise | Specify → Plan → Act → Review → Vault |

### Context Acquisition (Optional)

Before Specify scoring:
1. Check `.sparv/kb.md` for existing patterns/decisions
2. If insufficient, scan codebase for relevant files
3. Document findings in journal under `## Context`

### Knowledge Base Maintenance

During Vault phase, update `.sparv/kb.md`:
- **Patterns**: Reusable code patterns discovered
- **Decisions**: Architectural choices + rationale
- **Gotchas**: Common pitfalls + solutions

### CHANGELOG Update

```bash
~/.claude/skills/sparv/scripts/changelog-update.sh --type <Added|Changed|Fixed|Removed> --desc "..."
```

## Specify (10-Point Scale)

Each item scores 0/1/2, total 0-10; `>=9` required to enter Plan:

1) Value: Why do it, are benefits/metrics verifiable
2) Scope: MVP + what's out of scope
3) Acceptance: Testable acceptance criteria
4) Boundaries: Error/performance/compatibility/security critical boundaries
5) Risk: EHRB/dependencies/unknowns + handling approach

If below threshold, keep asking—don't "just start coding."

## Journal Convention (Unified Log)

All Plan/Progress/Findings go into `.sparv/journal.md`.

Recommended format (just append, no need to "insert into specific sections"):

```markdown
## 14:32 - Action #12
- Tool: Edit
- Result: Updated auth flow
- Next: Add test for invalid token
```

## 2-Action Save

Hook triggers `save-progress.sh` after each tool call; script only writes to journal when `action_count` is even.

## 3-Failure Protocol

When you fail consecutively, escalate by level:

1. Diagnose and fix (read errors, verify assumptions, minimal fix)
2. Alternative approach (change strategy/entry point)
3. Escalate (stop: document blocker + attempted solutions + request user decision)

Tools:

```bash
~/.claude/skills/sparv/scripts/failure-tracker.sh fail --note "short reason"
~/.claude/skills/sparv/scripts/failure-tracker.sh reset
```

## 3-Question Reboot Test

Self-check before session ends (or when lost):

1) Where am I? (current_phase)
2) Where am I going? (next_phase)
3) How do I prove completion? (completion_promise + evidence at journal end)

```bash
~/.claude/skills/sparv/scripts/reboot-test.sh --strict
```

## EHRB (High-Risk Changes)

Detection items (any match requires explicit user confirmation):

- Production access
- Sensitive data
- Destructive operations
- Billing external API
- Security-critical changes

```bash
~/.claude/skills/sparv/scripts/check-ehrb.sh --diff --fail-on-flags
```

## state.yaml (Minimal Schema)

Scripts only enforce 4 core fields; other fields are optional:

```yaml
session_id: "20260114-143022"
current_phase: "act"
action_count: 14
consecutive_failures: 0
max_iterations: 12
iteration_count: 0
completion_promise: "All acceptance criteria have tests and are green."
ehrb_flags: []
```
