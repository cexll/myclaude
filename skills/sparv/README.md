# sparv - SPARV Workflow

Minimal 5-phase workflow: **S**pecify → **P**lan → **A**ct → **R**eview → **V**ault.

Completes "requirements → verifiable delivery" in one pass with external memory.

## Installation

```bash
python install.py --module sparv
```

Installs to `~/.claude/skills/sparv/`.

## Usage

```
/sparv <task description>
```

## Core Rules (Mandatory)

| Rule | Description |
|------|-------------|
| **10-Point Specify Gate** | Spec score 0-10; must be >=9 to enter Plan |
| **2-Action Save** | Append to `.sparv/journal.md` every 2 tool calls |
| **3-Failure Protocol** | Stop and escalate after 3 consecutive failures |
| **EHRB** | Explicit confirmation for high-risk (production/sensitive/destructive/billing/security) |
| **Fixed Phase Names** | `specify|plan|act|review|vault` in `state.yaml` |

## 5-Phase Workflow

### Phase 1: Specify (10-Point Scale)

Each dimension scores 0/1/2, total 0-10:

| Dimension | Focus |
|-----------|-------|
| Value | Why do it, verifiable benefits/metrics |
| Scope | MVP + what's out of scope |
| Acceptance | Testable acceptance criteria |
| Boundaries | Error/performance/compatibility/security limits |
| Risk | EHRB/dependencies/unknowns + handling |

- `score < 9`: Keep asking questions; do not enter Plan
- `score >= 9`: Write `completion_promise`, then enter Plan

### Phase 2: Plan

- Break into atomic tasks (2-5 minute granularity)
- Each task has verifiable output/test point
- Write plan to `.sparv/journal.md`

### Phase 3: Act

- **TDD Rule**: No failing test → no production code
- Auto-write journal every 2 actions (PostToolUse hook)
- 3-Failure Protocol enforced

### Phase 4: Review

- Two stages: Spec conformance → Code quality
- Maximum 3 fix rounds; escalate if exceeded
- Run 3-question reboot test before session ends

### Phase 5: Vault

- Archive current session to `.sparv/history/`
- Update knowledge base `.sparv/kb.md`

## Enhanced Rules (v1.1)

### Uncertainty Declaration (G3)

When any Specify dimension scores < 2:
```
UNCERTAIN: <what> | ASSUMPTION: <fallback>
UNCERTAIN: deployment target | ASSUMPTION: Docker container
UNCERTAIN: auth method | OPTIONS: JWT / OAuth2 / Session
```

### Requirement Routing

| Mode | Condition | Flow |
|------|-----------|------|
| **Quick** | score >= 9 AND <= 3 files AND no EHRB | Specify → Act → Review |
| **Full** | otherwise | Specify → Plan → Act → Review → Vault |

### Knowledge Base Maintenance

During Vault phase, update `.sparv/kb.md`:
- **Patterns**: Reusable code patterns discovered
- **Decisions**: Architectural choices + rationale
- **Gotchas**: Common pitfalls + solutions

### CHANGELOG Update

For non-trivial changes:
```bash
~/.claude/skills/sparv/scripts/changelog-update.sh --type <Added|Changed|Fixed|Removed> --desc "..."
```

## External Memory

Initialize (run in project root):
```bash
~/.claude/skills/sparv/scripts/init-session.sh --force
```

Creates:
```
.sparv/
├── state.yaml      # State machine
├── journal.md      # Unified log
├── kb.md           # Knowledge base
└── history/        # Archive directory
```

| File | Purpose |
|------|--------|
| `state.yaml` | session_id, current_phase, action_count, consecutive_failures |
| `journal.md` | Plan/Progress/Findings unified log |
| `kb.md` | patterns/decisions/gotchas |
| `history/` | Archived sessions |

## Key Numbers

| Number | Meaning |
|--------|--------|
| **9/10** | Specify score passing threshold |
| **2** | Write to journal every 2 tool calls |
| **3** | Failure retry limit / Review fix limit |
| **3** | Reboot Test question count |

## Script Tools

| Script | Purpose |
|--------|--------|
| `init-session.sh` | Initialize `.sparv/`, generate state + journal |
| `save-progress.sh` | Maintain action_count, append journal |
| `check-ehrb.sh` | Scan diff/text, output ehrb_flags |
| `failure-tracker.sh` | Maintain consecutive_failures |
| `reboot-test.sh` | 3-question self-check |
| `archive-session.sh` | Archive to history/ |
| `changelog-update.sh` | Update CHANGELOG.md |

## Auto Hooks

Configured in `hooks/hooks.json`:

- **PostToolUse**: `save-progress.sh` (2-Action save)
- **PreToolUse**: `check-ehrb.sh --diff --dry-run` (prompt only)
- **Stop**: `reboot-test.sh --strict` (3-question self-check)

## Failure Tracking

```bash
# Record failure
~/.claude/skills/sparv/scripts/failure-tracker.sh fail --note "short blocker"

# Reset counter
~/.claude/skills/sparv/scripts/failure-tracker.sh reset
```

## Uninstall

```bash
python install.py --uninstall --module sparv
```
