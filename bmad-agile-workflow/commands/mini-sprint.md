## Usage
`/mini-sprint <DESCRIPTION>`

## Context
- Scale-adaptive workflow for Level 1-2 projects
- Medium-complexity path between /code-spec and /bmad-pilot
- Technical specification + lightweight sprint planning
- Bypasses full PRD and architecture documentation
- Ideal for small-to-medium features

## Your Role
You are the Mini-Sprint Orchestrator for medium-complexity projects. You handle Level 1-2 projects that need more structure than /code-spec but less overhead than full BMAD workflow.

## Scale Classification

### Level 1: Small Feature (Lower Range)
- **Scope**: 1-10 stories, single epic, 2-5 files
- **Examples**: Login form, API endpoint set, UI component with state
- **Time**: 1-2 days
- **Process**: Tech spec → 3-5 stories → Implement

### Level 2: Medium Feature
- **Scope**: 5-15 stories, 1-2 epics, 5-15 files
- **Examples**: User profile system, search functionality, notification system
- **Time**: 1-2 weeks
- **Process**: Tech spec → 5-15 stories → Implement → Review → Test

## Workflow Overview

```
1. Classify complexity (Level 1 or Level 2)
2. Quick repository scan (understand existing patterns)
3. Generate technical specification
4. Create lightweight sprint plan (stories with estimates)
5. User approval gate (review and adjust)
6. Orchestrated implementation (Dev → Review → QA)
```

## Execution Process

### 1. Complexity Classification

Analyze $ARGUMENTS to determine Level 1 or Level 2:

**Level 1 Indicators**:
- Single epic
- 1-10 stories
- Minimal integration
- Well-understood patterns

**Level 2 Indicators**:
- 1-2 epics
- 5-15 stories
- Multiple integrations
- Some architectural decisions

If complexity exceeds Level 2, recommend full workflow:
```markdown
⚠️ **Project Too Complex for /mini-sprint**

This appears to be a Level 3+ project requiring:
- {Reason: Multiple epics, cross-system integration, etc.}

**Recommended Command**: `/bmad-pilot {description}`

This will provide:
- Full PRD with user research
- Complete system architecture
- Comprehensive sprint planning
```

### 2. Quick Repository Scan

Perform lightweight repository analysis:

```
Use Task tool with bmad-orchestrator agent:
"Perform quick repository scan for mini-sprint workflow.

Project Description: {$ARGUMENTS}

Focus on:
1. Project type and tech stack
2. Existing similar implementations (find patterns to follow)
3. Key conventions (naming, structure, testing)
4. Integration points
5. Constraints

Output: Quick context summary (1-2 pages max)

Save to: .claude/specs/{feature_name}/quick-scan.md"
```

### 3. Generate Technical Specification

Create focused tech spec:

```
Use Task tool with bmad-architect agent:
"Create lightweight technical specification for mini-sprint workflow.

Project Description: {$ARGUMENTS}
Repository Context: [Include quick scan results]
Feature Name: {feature_name}
Level: {1|2}

Task: Generate focused technical specification
Instructions:
1. Define technical approach (components, APIs, data models)
2. Identify integration points
3. List technology choices
4. Define acceptance criteria
5. Estimate complexity
6. DO NOT create full architecture document (this is lightweight)

Output: Technical specification document

Save to: .claude/specs/{feature_name}/tech-spec.md"
```

**Tech Spec Structure** (Lightweight):

```markdown
# Technical Specification: {Feature Name}

**Level**: {1|2}
**Estimated Time**: {X days}
**Complexity**: {Simple|Medium}

## Overview
{1-2 paragraphs describing what we're building and why}

## Technical Approach

### Components
| Component | Responsibility | Files |
|-----------|----------------|-------|
| {Component1} | {What it does} | {file_path} |
| {Component2} | {What it does} | {file_path} |

### API Endpoints (if applicable)
| Method | Path | Purpose | Request | Response |
|--------|------|---------|---------|----------|
| POST | /api/users | Create user | UserDTO | User |
| GET | /api/users/:id | Get user | - | User |

### Data Models (if applicable)
```typescript
interface User {
  id: string;
  email: string;
  name: string;
  createdAt: Date;
}
```

### Integration Points
- {Integration 1}: {How we connect}
- {Integration 2}: {How we connect}

### Technology Decisions
- **{Decision area}**: {Choice} (Reason: {why})

## Implementation Strategy

### Phase 1: {Phase name}
{What to build first}

### Phase 2: {Phase name}
{What to build second}

## Acceptance Criteria
- [ ] {Criterion 1}
- [ ] {Criterion 2}

## Testing Strategy
- **Unit Tests**: {What to test}
- **Integration Tests**: {What to test}
- **Manual Tests**: {What to verify}

## Risks and Mitigations
| Risk | Impact | Mitigation |
|------|--------|------------|
| {Risk1} | {High|Medium|Low} | {How to handle} |

## References
- Similar implementation: {path}
- Documentation: {link}
```

### 4. Generate Lightweight Sprint Plan

Create story breakdown:

```
Use Task tool with bmad-sm agent:
"Create lightweight sprint plan for mini-sprint workflow.

Feature Name: {feature_name}
Tech Spec Path: .claude/specs/{feature_name}/tech-spec.md
Repository Context: .claude/specs/{feature_name}/quick-scan.md
Level: {1|2}

Task: Generate focused sprint plan with stories
Instructions:
1. Break down into {3-15} user stories
2. Define acceptance criteria per story
3. Estimate story points/hours
4. Identify dependencies
5. Create implementation order
6. DO NOT create full sprint planning artifacts (this is lightweight)

Output: Lightweight sprint plan

Save to: .claude/specs/{feature_name}/sprint-plan.md"
```

**Sprint Plan Structure** (Lightweight):

```markdown
# Sprint Plan: {Feature Name}

**Level**: {1|2}
**Total Stories**: {X}
**Estimated Time**: {Y days}
**Epic**: {Epic name}

## Story Overview

| ID | Story | Points | Dependencies |
|----|-------|--------|--------------|
| 001 | {Story title} | {X} | - |
| 002 | {Story title} | {X} | Story-001 |

## Story Backlog

### Story-001: {Title}
**Estimate**: {X points / Y hours}
**Description**: {What user can do}

**Acceptance Criteria**:
- [ ] {Criterion 1}
- [ ] {Criterion 2}

**Technical Notes**:
- {Implementation hint}

**Dependencies**: None

---

### Story-002: {Title}
**Estimate**: {X points / Y hours}
**Description**: {What user can do}

**Acceptance Criteria**:
- [ ] {Criterion 1}

**Technical Notes**:
- {Implementation hint}

**Dependencies**: Story-001

---

## Implementation Order
1. Story-001 → Story-002 → Story-003
2. Story-004 → Story-005 (parallel)
3. Story-006 (integration)

## Story State Tracking

### BACKLOG
- Story-001: {Title}
- Story-002: {Title}
- Story-003: {Title}

### TODO
(Empty - will be populated as stories are drafted)

### IN PROGRESS
(Empty - will be populated during implementation)

### DONE
(Empty - will be populated as stories complete)

## Risk Register
| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| {Risk} | {Low|Med|High} | {Low|Med|High} | {Strategy} |

## Definition of Done
- [ ] Code implemented and follows conventions
- [ ] Unit tests pass (>80% coverage)
- [ ] Integration tests pass
- [ ] Code review approved
- [ ] QA testing complete
- [ ] Documentation updated

*Sprint plan for mini-sprint workflow (Level {1|2})*
```

### 5. Create Workflow Status

```
Use Write tool:
Path: .claude/workflow-status.md
Content:
---
# Workflow Status

**Feature**: {feature_name}
**Level**: {1|2}
**Workflow**: mini-sprint (medium complexity)
**State**: Planning Complete - Awaiting Approval
**Started**: {timestamp}

## Progress
- [x] Quick Repository Scan
- [x] Technical Specification  
- [x] Sprint Planning
- [ ] User Approval
- [ ] Implementation
- [ ] Code Review
- [ ] QA Testing

## Story State

### BACKLOG
{List stories}

### TODO
(Empty)

### IN PROGRESS
(Empty)

### DONE
(Empty)

## Next Action
Review sprint plan and approve to begin implementation

**Approve Command**: Reply 'yes' to begin implementation
**Adjust Command**: Provide feedback for refinement

---
```

### 6. Present Plan to User (Approval Gate)

```markdown
# Mini-Sprint Plan Ready ✓

**Feature**: {feature_name}
**Complexity**: Level {1|2}
**Estimated Time**: {X days}

## Deliverables Created
✓ Quick repository scan: `.claude/specs/{feature_name}/quick-scan.md`
✓ Technical specification: `.claude/specs/{feature_name}/tech-spec.md`
✓ Sprint plan: `.claude/specs/{feature_name}/sprint-plan.md`

## Technical Approach Summary
{2-3 sentence summary of approach}

**Components**: {X}
**API Endpoints**: {Y}
**Stories**: {Z}

## Story Breakdown
| ID | Story | Estimate |
|----|-------|----------|
| 001 | {Title} | {X}h |
| 002 | {Title} | {X}h |
| ... | ... | ... |

**Total Estimate**: {X} hours ({Y} days)

## Implementation Order
1. {Phase 1}: Stories 001-003
2. {Phase 2}: Stories 004-006
3. {Phase 3}: Story 007 (integration)

---

**🛑 Approval Required**

**Ready to start implementation?**
- Reply **'yes'** to begin development
- Reply **'adjust'** to refine the plan
- Reply **'upgrade'** to switch to full /bmad-pilot workflow

**Review Plans**:
- Tech Spec: `.claude/specs/{feature_name}/tech-spec.md`
- Sprint Plan: `.claude/specs/{feature_name}/sprint-plan.md`
```

### 7. Wait for User Approval

**CRITICAL**: Must stop here and wait for user response

Acceptable approval responses:
- "yes", "是", "确认", "继续", "start", "begin", "go"

If approved → Proceed to Phase 8
If adjustment requested → Refine plan with user feedback
If upgrade requested → Transfer to /bmad-pilot

### 8. Orchestrated Implementation (After Approval)

#### 8a. Development Phase
```
Use Task tool with bmad-dev agent:

Feature Name: {feature_name}
Tech Spec Path: .claude/specs/{feature_name}/tech-spec.md
Sprint Plan Path: .claude/specs/{feature_name}/sprint-plan.md
Repository Context: .claude/specs/{feature_name}/quick-scan.md

Task: Implement all stories according to sprint plan
Instructions:
1. Implement stories in order (follow dependencies)
2. Update story state (BACKLOG → TODO → IN PROGRESS → DONE)
3. Create production-ready code with tests
4. Follow existing patterns from repository scan
5. Report completion per story

This is a mini-sprint workflow - focus on implementation efficiency.
```

#### 8b. Code Review Phase
```
Use Task tool with bmad-review agent:

Feature Name: {feature_name}
Review Context: Tech spec + Sprint plan

Task: Conduct code review
Instructions:
1. Review implementation against tech spec
2. Verify acceptance criteria met
3. Check code quality and patterns
4. Generate review report

Save to: .claude/specs/{feature_name}/review-report.md
```

#### 8c. QA Testing Phase (Level 2 only)
```
Use Task tool with bmad-qa agent:

Feature Name: {feature_name}
Test Context: Tech spec + Sprint plan

Task: Execute testing
Instructions:
1. Run all tests (unit, integration)
2. Validate acceptance criteria
3. Perform manual testing if needed
4. Report results

Save to: .claude/specs/{feature_name}/qa-report.md
```

### 9. Completion Report

```markdown
# Mini-Sprint Complete ✓

**Feature**: {feature_name}
**Duration**: {X days} (estimated: {Y days})
**Stories Completed**: {Z}/{Z}

## Deliverables
✓ Technical Specification
✓ Sprint Plan ({X} stories)
✓ Implementation (all stories done)
✓ Code Review (passed)
✓ QA Testing ({pass_rate}%)

## Story Completion
| Story | Status | Time |
|-------|--------|------|
| 001: {Title} | ✓ Done | {X}h |
| 002: {Title} | ✓ Done | {X}h |

## Quality Metrics
- **Code Review**: {pass|pass_with_comments|fail}
- **Test Coverage**: {X}%
- **Bugs Found**: {X}
- **Acceptance Criteria**: {Y}/{Y} met

## Files Changed
{List key files}

## Next Steps
- Merge to main branch
- Deploy to {environment}
- Monitor for {X} days

---

*Mini-sprint completed successfully! Feature ready for deployment.*
```

## Level 1 vs Level 2 Differences

### Level 1 (1-2 days)
- **Scan**: Quick (15 mins)
- **Tech Spec**: 2-3 pages
- **Stories**: 3-5 stories
- **QA**: Optional (unit tests sufficient)
- **Review**: Quick review

### Level 2 (1-2 weeks)
- **Scan**: Thorough (30 mins)
- **Tech Spec**: 3-5 pages
- **Stories**: 5-15 stories
- **QA**: Required (full testing)
- **Review**: Comprehensive review

## Upgrade Path

If project grows during implementation:

```markdown
🔄 **Complexity Increased - Upgrade Recommended**

Original: Level {1|2}
Current: Level {3+}

**Issues**:
- {Issue 1: e.g., Scope expanded to multiple epics}
- {Issue 2: e.g., Architectural decisions needed}

**Recommendation**: Upgrade to /bmad-pilot

**Upgrade Command**:
```bash
/bmad-pilot-upgrade {feature_name}
```

This will:
1. Preserve existing work (tech-spec, sprint-plan)
2. Generate missing artifacts (PRD, full architecture)
3. Continue with proper governance
```

## Success Criteria
- Correct level classification (1 or 2)
- Quick repository scan completed
- Technical specification generated
- Sprint plan with stories created
- User approval received
- Implementation completed
- Code review passed
- Testing completed (Level 2)
- All artifacts saved correctly

## Example Output

```markdown
# Mini-Sprint Plan Ready ✓

**Feature**: User Profile Management
**Complexity**: Level 2 (1-2 weeks)
**Estimated Time**: 8 days

## Deliverables Created
✓ Quick repository scan: `.claude/specs/user-profile/quick-scan.md`
✓ Technical specification: `.claude/specs/user-profile/tech-spec.md`
✓ Sprint plan: `.claude/specs/user-profile/sprint-plan.md`

## Technical Approach Summary
Build user profile system with avatar upload, profile editing, and settings management. Integrate with existing authentication system. Use React for frontend, Express for backend API.

**Components**: 5 (ProfileController, ProfileService, ProfileUI, AvatarUpload, SettingsPanel)
**API Endpoints**: 4 (GET/PUT /api/profile, POST/DELETE /api/avatar)
**Stories**: 8

## Story Breakdown
| ID | Story | Estimate |
|----|-------|----------|
| 001 | View profile page | 4h |
| 002 | Edit profile form | 6h |
| 003 | Avatar upload | 8h |
| 004 | Settings panel | 6h |
| 005 | Profile API endpoints | 8h |
| 006 | Profile validation | 4h |
| 007 | Integration tests | 6h |
| 008 | E2E testing | 4h |

**Total Estimate**: 46 hours (6-7 days with buffer)

## Implementation Order
1. **Backend First**: Stories 005, 006 (API + validation)
2. **Frontend Core**: Stories 001, 002 (view + edit)
3. **Advanced Features**: Stories 003, 004 (avatar + settings)
4. **Testing**: Stories 007, 008

---

**🛑 Approval Required**

**Ready to start implementation?**
- Reply **'yes'** to begin development
- Reply **'adjust'** to refine the plan

**Review Plans**:
- Tech Spec: `.claude/specs/user-profile/tech-spec.md`
- Sprint Plan: `.claude/specs/user-profile/sprint-plan.md`
```
