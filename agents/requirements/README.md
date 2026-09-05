# requirements - Requirements-Driven Workflow

Lightweight requirements-to-code pipeline with interactive quality gates.

## Installation

```bash
python install.py --module requirements
```

## Usage

```bash
/requirements-pilot <FEATURE_DESCRIPTION> [OPTIONS]
```

### Options

| Option | Description |
|--------|-------------|
| `--skip-tests` | Skip testing phase entirely |
| `--skip-scan` | Skip initial repository scanning |

## Workflow Phases

| Phase | Description | Output |
|-------|-------------|--------|
| 0 | Repository scanning | `00-repository-context.md` |
| 1 | Requirements confirmation | `requirements-confirm.md` (90+ score required) |
| 2 | Implementation | Code + `requirements-spec.md` |

## Quality Scoring (100-point system)

| Category | Points | Focus |
|----------|--------|-------|
| Functional Clarity | 30 | Input/output specs, success criteria |
| Technical Specificity | 25 | Integration points, constraints |
| Implementation Completeness | 25 | Edge cases, error handling |
| Business Context | 20 | User value, priority |

## Sub-Agents

| Agent | Role |
|-------|------|
| `requirements-generate` | Create technical specifications |
| `requirements-code` | Implement functionality |
| `requirements-review` | Code quality evaluation |
| `requirements-testing` | Test case creation |

## Approval Gate

One mandatory stop point after Phase 1:
- Requirements must achieve 90+ quality score
- User must explicitly approve before implementation begins

## Testing Decision

After code review passes (≥90%):
- `--skip-tests`: Complete without testing
- No option: Interactive prompt with smart recommendations based on task complexity

## Output Structure

```
.claude/specs/{feature_name}/
├── 00-repository-context.md
├── requirements-confirm.md
└── requirements-spec.md
```

## When to Use

- Quick prototypes
- Well-defined features
- Smaller scope tasks
- When full BMAD workflow is overkill

## Directory Structure

```
agents/requirements/
├── README.md
├── commands/
│   └── requirements-pilot.md
└── agents/
    ├── requirements-generate.md
    ├── requirements-code.md
    ├── requirements-review.md
    └── requirements-testing.md
```
