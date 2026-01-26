# BMAD Workflow Complete Guide

> **BMAD (Business-Minded Agile Development)** - AI-driven agile development automation with role-based agents

## 🎯 What is BMAD?

BMAD is an enterprise-grade agile development methodology that transforms your development process into a fully automated workflow with 6 specialized AI agents and quality gates.

### Core Principles

- **Agent Planning**: Specialized agents collaborate to create detailed, consistent PRDs and architecture documents
- **Context-Driven Development**: Transform detailed plans into ultra-detailed development stories
- **Role Specialization**: Each agent focuses on specific domains, avoiding quality degradation from role switching

## 🤖 BMAD Agent System

### Agent Roles

| Agent | Role | Quality Gate | Artifacts |
|-------|------|--------------|-----------|
| **bmad-po** (Sarah) | Product Owner - requirements gathering, user stories | PRD ≥ 90/100 | `01-product-requirements.md` |
| **bmad-architect** (Winston) | System Architect - technical design, system architecture | Design ≥ 90/100 | `02-system-architecture.md` |
| **bmad-sm** (Mike) | Scrum Master - task breakdown, sprint planning | User approval | `03-sprint-plan.md` |
| **bmad-dev** (Alex) | Developer - code implementation, technical docs | Code completion | Implementation files |
| **bmad-review** | Code Reviewer - independent review between Dev and QA | Pass/Risk/Fail | `04-dev-reviewed.md` |
| **bmad-qa** (Emma) | QA Engineer - testing strategy, quality assurance | Test execution | `05-qa-report.md` |

## 🚀 Quick Start

### Command Overview

```bash
# Full BMAD workflow
/bmad-pilot "Build e-commerce checkout system with payment integration"

# Workflow: PO → Architect → SM → Dev → Review → QA
```

### Command Options

```bash
# Skip testing phase
/bmad-pilot "Admin dashboard" --skip-tests

# Skip sprint planning (architecture → dev directly)
/bmad-pilot "API gateway implementation" --direct-dev

# Skip repository scan (not recommended)
/bmad-pilot "Add feature" --skip-scan
```

### Individual Agent Usage

```bash
# Product requirements analysis only
/bmad-po "Enterprise CRM system requirements"

# Technical architecture design only
/bmad-architect "High-concurrency distributed system design"

# Orchestrator (can transform into any agent)
/bmad-orchestrator "Coordinate multi-agent complex project"
```

## 📋 Workflow Phases

### Phase 0: Repository Scan (Automatic)
- **Agent**: `bmad-orchestrator`
- **Output**: `00-repository-context.md`
- **Content**: Project type, tech stack, code organization, conventions, integration points

### Phase 1: Product Requirements (PO)
- **Agent**: `bmad-po` (Sarah - Product Owner)
- **Quality Gate**: PRD score ≥ 90/100
- **Output**: `01-product-requirements.md`
- **Process**:
  1. PO generates initial PRD
  2. System calculates quality score (100-point scale)
  3. If < 90: User provides feedback → PO revises → Recalculate
  4. If ≥ 90: User confirms → Save artifact → Next phase

### Phase 2: System Architecture (Architect)
- **Agent**: `bmad-architect` (Winston - System Architect)
- **Quality Gate**: Design score ≥ 90/100
- **Output**: `02-system-architecture.md`
- **Process**:
  1. Architect reads PRD + repo context
  2. Generates technical design document
  3. System calculates design quality score
  4. If < 90: User provides feedback → Architect revises
  5. If ≥ 90: User confirms → Save artifact → Next phase

### Phase 3: Sprint Planning (SM)
- **Agent**: `bmad-sm` (Mike - Scrum Master)
- **Quality Gate**: User approval
- **Output**: `03-sprint-plan.md`
- **Process**:
  1. SM reads PRD + Architecture
  2. Breaks down tasks with story points
  3. User reviews sprint plan
  4. User confirms → Save artifact → Next phase
- **Skip**: Use `--direct-dev` to skip this phase

### Phase 4: Development (Dev)
- **Agent**: `bmad-dev` (Alex - Developer)
- **Quality Gate**: Code completion
- **Output**: Implementation files
- **Process**:
  1. Dev reads all previous artifacts
  2. Implements features following sprint plan
  3. Creates or modifies code files
  4. Completes implementation → Next phase

### Phase 5: Code Review (Review)
- **Agent**: `bmad-review` (Independent Reviewer)
- **Quality Gate**: Pass / Pass with Risk / Fail
- **Output**: `04-dev-reviewed.md`
- **Process**:
  1. Review reads implementation + all specs
  2. Performs comprehensive code review
  3. Generates review report with status:
     - **Pass**: No issues, proceed to QA
     - **Pass with Risk**: Non-critical issues noted
     - **Fail**: Critical issues, return to Dev
  4. Updates sprint plan with review findings

**Enhanced Review (Optional)**:
- Use GPT-5 via Codex CLI for deeper analysis
- Set via `BMAD_REVIEW_MODE=enhanced` environment variable

### Phase 6: Quality Assurance (QA)
- **Agent**: `bmad-qa` (Emma - QA Engineer)
- **Quality Gate**: Test execution
- **Output**: `05-qa-report.md`
- **Process**:
  1. QA reads implementation + review + all specs
  2. Creates targeted test strategy
  3. Executes tests
  4. Generates QA report
  5. Workflow complete
- **Skip**: Use `--skip-tests` to skip this phase

## 📊 Quality Scoring System

### PRD Quality (100 points)
- **Business Value** (30): Clear value proposition, user benefits
- **Functional Requirements** (25): Complete, unambiguous requirements
- **User Experience** (20): User flows, interaction patterns
- **Technical Constraints** (15): Performance, security, scalability
- **Scope & Priorities** (10): Clear boundaries, must-have vs nice-to-have

### Architecture Quality (100 points)
- **Design Quality** (30): Modularity, maintainability, clarity
- **Technology Selection** (25): Appropriate tech stack, justification
- **Scalability** (20): Growth handling, performance considerations
- **Security** (15): Authentication, authorization, data protection
- **Feasibility** (10): Realistic implementation, resource alignment

### Review Status (3 levels)
- **Pass**: No critical issues, code meets standards
- **Pass with Risk**: Non-critical issues, recommendations included
- **Fail**: Critical issues, requires Dev iteration

## 📁 Workflow Artifacts

All documents are saved to `.claude/specs/{feature-name}/`:

```
.claude/specs/e-commerce-checkout/
├── 00-repository-context.md    # Repo analysis (auto)
├── 01-product-requirements.md  # PRD (PO, score ≥ 90)
├── 02-system-architecture.md   # Design (Architect, score ≥ 90)
├── 03-sprint-plan.md           # Sprint plan (SM, user approved)
├── 04-dev-reviewed.md          # Code review (Review, Pass/Risk/Fail)
└── 05-qa-report.md            # Test report (QA, tests executed)
```

Feature name generated from project description (kebab-case: lowercase, spaces/punctuation → `-`).

## 🔧 Advanced Usage

### Approval Gates

Critical phases require explicit user confirmation:

```
Architect: "Technical design complete (Score: 93/100)"
System: "Ready to proceed to sprint planning? (yes/no)"
User: yes
```

### Iterative Refinement

Each phase supports feedback loops:

```
PO: "Here's the PRD (Score: 75/100)"
User: "Add mobile support and offline mode"
PO: "Updated PRD (Score: 92/100) ✅"
```

### Repository Context

BMAD automatically scans your repository to understand:
- Technology stack (languages, frameworks, libraries)
- Project structure (directories, modules, patterns)
- Existing conventions (naming, formatting, architecture)
- Dependencies (package managers, external services)
- Integration points (APIs, databases, third-party services)

### Workflow Variations

**Fast Prototyping** - Skip non-essential phases:
```bash
/bmad-pilot "Quick admin UI" --skip-tests --direct-dev
# Workflow: PO → Architect → Dev
```

**Architecture-First** - Focus on design:
```bash
/bmad-architect "Microservices architecture for e-commerce"
# Only runs Architect agent
```

**Full Rigor** - All phases with maximum quality:
```bash
/bmad-pilot "Enterprise payment gateway with PCI compliance"
# Workflow: Scan → PO → Architect → SM → Dev → Review → QA
```

## 🎨 Output Style

BMAD workflow uses a specialized output style that:
- Creates phase-separated contexts
- Manages agent handoffs with clear boundaries
- Tracks quality scores across phases
- Handles approval gates with user prompts
- Supports Codex CLI integration for enhanced reviews

## 📚 Related Documentation

- **[Quick Start Guide](QUICK-START.md)** - Get started in 5 minutes
- **[Plugin System](PLUGIN-SYSTEM.md)** - Installation and configuration
- **[Development Commands](DEVELOPMENT-COMMANDS.md)** - Alternative workflows
- **[Requirements Workflow](REQUIREMENTS-WORKFLOW.md)** - Lightweight alternative

## 💡 Best Practices

1. **Don't skip repository scan** - Helps agents understand your project context
2. **Provide detailed descriptions** - Better input → better output
3. **Engage with agents** - Provide feedback during quality gates
4. **Review artifacts** - Check generated documents before confirming
5. **Use appropriate workflows** - Full BMAD for complex features, lightweight for simple tasks
6. **Keep artifacts** - They serve as project documentation and context for future work

---

**Transform your development with BMAD** - One command, complete agile workflow, quality assured.
