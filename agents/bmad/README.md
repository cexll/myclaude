# bmad - BMAD Agile Workflow

Full enterprise agile methodology with 6 specialized agents, UltraThink analysis, and repository-aware development.

## Installation

```bash
python install.py --module bmad
```

## Usage

```bash
/bmad-pilot <PROJECT_DESCRIPTION> [OPTIONS]
```

### Options

| Option | Description |
|--------|-------------|
| `--skip-tests` | Skip QA testing phase |
| `--direct-dev` | Skip SM planning, go directly to development |
| `--skip-scan` | Skip initial repository scanning |

## Workflow Phases

| Phase | Agent | Deliverable | Description |
|-------|-------|-------------|-------------|
| 0 | Orchestrator | `00-repo-scan.md` | Repository scanning with UltraThink analysis |
| 1 | Product Owner (PO) | `01-product-requirements.md` | PRD with 90+ quality score required |
| 2 | Architect | `02-system-architecture.md` | Technical design with 90+ score required |
| 3 | Scrum Master (SM) | `03-sprint-plan.md` | Sprint backlog with stories and estimates |
| 4 | Developer | Implementation code | Multi-sprint implementation |
| 4.5 | Reviewer | `04-dev-reviewed.md` | Code review (Pass/Pass with Risk/Fail) |
| 5 | QA Engineer | Test suite | Comprehensive testing and validation |

## Agents

| Agent | Role |
|-------|------|
| `bmad-orchestrator` | Repository scanning, workflow coordination |
| `bmad-po` | Requirements gathering, PRD creation |
| `bmad-architect` | System design, technology decisions |
| `bmad-sm` | Sprint planning, task breakdown |
| `bmad-dev` | Code implementation |
| `bmad-review` | Code review, quality assessment |
| `bmad-qa` | Testing, validation |

## Approval Gates

Two mandatory stop points require explicit user approval:

1. **After PRD** (Phase 1 → 2): User must approve requirements before architecture
2. **After Architecture** (Phase 2 → 3): User must approve design before implementation

## Output Structure

```
.claude/specs/{feature_name}/
├── 00-repo-scan.md
├── 01-product-requirements.md
├── 02-system-architecture.md
├── 03-sprint-plan.md
└── 04-dev-reviewed.md
```

## UltraThink Methodology

Applied throughout the workflow for deep analysis:

1. **Hypothesis Generation** - Form hypotheses about the problem
2. **Evidence Collection** - Gather evidence from codebase
3. **Pattern Recognition** - Identify recurring patterns
4. **Synthesis** - Create comprehensive understanding
5. **Validation** - Cross-check findings

## Interactive Confirmation Flow

PO and Architect phases use iterative refinement:

1. Agent produces initial draft + quality score
2. Orchestrator presents to user with clarification questions
3. User provides responses
4. Agent refines until quality >= 90
5. User confirms to save deliverable

## When to Use

- Large multi-sprint features
- Enterprise projects requiring documentation
- Team coordination scenarios
- Projects needing formal approval gates

## Directory Structure

```
agents/bmad/
├── README.md
├── commands/
│   └── bmad-pilot.md
└── agents/
    ├── bmad-orchestrator.md
    ├── bmad-po.md
    ├── bmad-architect.md
    ├── bmad-sm.md
    ├── bmad-dev.md
    ├── bmad-review.md
    └── bmad-qa.md
```
