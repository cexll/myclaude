# SPARV - Unified Development Workflow (Simplified)

[![Skill Version](https://img.shields.io/badge/version-1.0.0-blue.svg)]()
[![Claude Code](https://img.shields.io/badge/Claude%20Code-Compatible-green.svg)]()

**SPARV** is an end-to-end development workflow: maximize delivery quality with minimal rules while avoiding "infinite iteration + self-rationalization."

```
S-Specify → P-Plan → A-Act → R-Review → V-Vault
   Clarify     Plan      Execute   Review     Archive
```

## Key Changes (Over-engineering Removed)

- External memory merged from 3 files into 1 `.sparv/journal.md`
- Specify scoring simplified from 100-point to 10-point scale (threshold `>=9`)
- Reboot Test reduced from 5 questions to 3 questions
- Removed concurrency locks (Claude is single-threaded; locks only cause failures)

## Installation

SPARV is installed at `~/.claude/skills/sparv/`.

Install from ZIP:

```bash
unzip sparv.zip -d ~/.claude/skills/
```

## Quick Start

Run in project root:

```bash
~/.claude/skills/sparv/scripts/init-session.sh --force
```

Creates:

```
.sparv/
├── state.yaml
├── journal.md
└── history/
```

## External Memory System (Two Files)

- `state.yaml`: State (minimum fields: `session_id/current_phase/action_count/consecutive_failures`)
- `journal.md`: Unified log (Plan/Progress/Findings all go here)

After archiving:

```
.sparv/history/<session_id>/
├── state.yaml
└── journal.md
```

## Key Numbers

| Number | Meaning |
|--------|---------|
| **9/10** | Specify score passing threshold |
| **2** | Write to journal every 2 tool calls |
| **3** | Failure retry limit / Review fix limit |
| **3** | Reboot Test question count |
| **12** | Default max iterations (optional safety valve) |

## Script Tools

```bash
~/.claude/skills/sparv/scripts/init-session.sh --force
~/.claude/skills/sparv/scripts/save-progress.sh "Edit" "done"
~/.claude/skills/sparv/scripts/check-ehrb.sh --diff --fail-on-flags
~/.claude/skills/sparv/scripts/failure-tracker.sh fail --note "tests are flaky"
~/.claude/skills/sparv/scripts/reboot-test.sh --strict
~/.claude/skills/sparv/scripts/archive-session.sh
```

## Hooks

Hooks defined in `hooks/hooks.json`:

- PostToolUse: 2-Action auto-write to `journal.md`
- PreToolUse: EHRB risk prompt (default dry-run)
- Stop: 3-question reboot test (strict)

## References

- `SKILL.md`: Skill definition (for agent use)
- `references/methodology.md`: Methodology quick reference

---

*Quality over speed—iterate until truly complete.*
