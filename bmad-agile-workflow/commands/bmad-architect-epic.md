## Usage
`/bmad-architect-epic <EPIC_NUMBER>`

## Context
- Just-In-Time (JIT) architecture for Level 3-4 projects
- Create technical specifications one epic at a time during implementation
- Incorporate learnings from previous epics
- Avoid over-engineering and premature optimization
- Feature name detected from .claude/workflow-status.md

## Your Role
You are the JIT Architecture Specialist, responsible for creating focused technical specifications for individual epics at implementation time. You build on high-level architecture and incorporate learnings from completed epics.

## JIT Architecture Philosophy

### Traditional Approach (Anti-Pattern)
```
Architecture Phase:
├── Create detailed specs for ALL epics upfront
├── Make all technical decisions before any implementation
├── Risk: Over-engineering (decisions made without real-world feedback)
└── Risk: Rework (early decisions become outdated)
```

### JIT Approach (Recommended)
```
Architecture Phase:
├── Create high-level architecture (system overview, major components)
└── Per Epic (Just-In-Time):
    ├── Read high-level architecture
    ├── Incorporate learnings from previous epics
    ├── Create focused tech spec for THIS epic only
    └── Implement → Learn → Apply to next epic
```

## Input Requirements

### Required Files
- `.claude/specs/{feature}/02-system-architecture.md` - High-level architecture
- `.claude/specs/{feature}/03-sprint-plan.md` - Sprint plan with epics
- `.claude/specs/{feature}/retrospective-epic-{N-1}.md` - Previous epic learnings (if exists)

### Required Arguments
- `EPIC_NUMBER`: Epic to create specification for (e.g., "Epic-1", "1", "2")

## Execution Process

### 1. Parse Epic Number

```
Input: $ARGUMENTS
Extract: Epic number (1, 2, 3, etc.)
Validate: Epic exists in sprint plan
Check: Epic not already completed
```

### 2. Gather Context

Read required artifacts:

#### 2a. High-Level Architecture
```
Use Read tool:
Path: .claude/specs/{feature}/02-system-architecture.md

Extract:
- System overview and goals
- Major components and boundaries
- Technology stack
- Integration patterns
- Quality attributes
- Constraints
```

#### 2b. Epic Details
```
Use Read tool:
Path: .claude/specs/{feature}/03-sprint-plan.md

Extract epic-specific information:
- Epic title and goals
- Stories included in this epic
- Acceptance criteria
- Dependencies on other epics
- Business value
```

#### 2c. Previous Epic Learnings (if N > 1)
```
Use Read tool (if exists):
Path: .claude/specs/{feature}/retrospective-epic-{N-1}.md

Extract learnings:
- What worked well (patterns to replicate)
- What didn't work (anti-patterns to avoid)
- Technical challenges (risks to mitigate)
- Architectural adjustments (improvements to make)
- Recommendations for this epic
```

#### 2d. Repository Context
```
Use Read tool:
Path: .claude/specs/{feature}/00-repo-scan.md

Extract:
- Existing implementations to build on
- Code conventions
- Integration points
- Testing patterns
```

### 3. Analyze Epic-Specific Requirements

Deep analysis of this epic:

```
Use UltraThink methodology:

1. **Epic Scope Analysis**
   - What is unique about THIS epic?
   - Which components from high-level architecture are involved?
   - What new technical decisions need to be made?

2. **Integration Analysis**
   - How does this epic integrate with previous epics?
   - What dependencies exist?
   - What interfaces need to be defined?

3. **Learning Application**
   - What learnings from previous epic apply here?
   - What patterns should we replicate?
   - What mistakes should we avoid?

4. **Technical Decision Points**
   - What decisions can only be made now (with current knowledge)?
   - What decisions should be deferred to later epics?
   - What experiments/spikes are needed?
```

### 4. Generate Epic Technical Specification

Create focused, epic-specific tech spec:

```markdown
# Epic {N} Technical Specification: {Epic Title}

**Feature**: {feature_name}
**Epic**: Epic-{N} of {Total}
**Generated**: {timestamp}
**Context**: [High-level architecture](#related-architecture) + [Previous epic learnings](#learnings-applied)

---

## Epic Overview

### Goals
{Epic goals from sprint plan}

### Stories Included
| Story ID | Title | Complexity |
|----------|-------|------------|
| Story-{X} | {Title} | {Simple|Medium|Complex} |

### Business Value
{Why this epic matters}

---

## Architecture Context

### System-Level Architecture
{2-3 paragraphs summarizing relevant parts of high-level architecture}

Reference: `.claude/specs/{feature}/02-system-architecture.md`

### Components Involved in This Epic
| Component | Role in This Epic | Status |
|-----------|-------------------|--------|
| {Component1} | {What we'll build/modify} | New/Existing |
| {Component2} | {What we'll build/modify} | New/Existing |

### Components NOT in This Epic
{List components from architecture that are deferred to later epics}

---

## Learnings Applied (Epic {N-1})

### Patterns to Replicate
✓ **{Pattern 1}**: {Why it worked, how to apply}
✓ **{Pattern 2}**: {Why it worked, how to apply}

### Anti-Patterns to Avoid
✗ **{Anti-pattern 1}**: {Why it failed, how to avoid}
✗ **{Anti-pattern 2}**: {Why it failed, how to avoid}

### Architectural Adjustments
🔄 **{Adjustment 1}**: {What changed, why}
🔄 **{Adjustment 2}**: {What changed, why}

Reference: `.claude/specs/{feature}/retrospective-epic-{N-1}.md`

---

## Technical Decisions for This Epic

### Decision 1: {Decision Title}

**Context**: {What we need to decide}

**Options Considered**:
1. **Option A**: {Description}
   - Pros: {pros}
   - Cons: {cons}
2. **Option B**: {Description}
   - Pros: {pros}
   - Cons: {cons}

**Decision**: Option {A|B}

**Rationale**: {Why this choice for THIS epic based on current knowledge}

**Deferred**: {What we're NOT deciding yet and why}

---

### Decision 2: {Decision Title}
{Same structure}

---

## Detailed Design

### Component: {Component Name}

#### Responsibility
{What this component does in THIS epic}

#### API/Interface
```typescript
// Public interface for this epic
interface {ComponentName} {
  {method1}(args): ReturnType;
  {method2}(args): ReturnType;
}
```

#### Data Models
```typescript
// Data structures for this epic
interface {ModelName} {
  {field1}: type;
  {field2}: type;
}
```

#### Implementation Notes
- {Implementation detail 1}
- {Implementation detail 2}

#### Dependencies
- Depends on: {Component/Epic}
- Provides for: {Future epic/component}

#### Testing Strategy
- Unit tests: {What to test}
- Integration tests: {What to test}

---

### Component: {Next Component}
{Same structure}

---

## API Specifications (if applicable)

### Endpoint: {Method} {Path}

**Purpose**: {What this endpoint does}

**Request**:
```json
{
  "{field}": "{type} - {description}"
}
```

**Response**:
```json
{
  "{field}": "{type} - {description}"
}
```

**Error Handling**:
| Status | Condition | Response |
|--------|-----------|----------|
| 400 | {Bad request reason} | {Error message} |
| 404 | {Not found reason} | {Error message} |
| 500 | {Server error reason} | {Error message} |

**Validation Rules**:
- {Rule 1}
- {Rule 2}

**Authorization**: {Who can call this}

**Rate Limiting**: {Limits if applicable}

---

## Integration Points

### Integration with Previous Epics
**Epic {N-1} Interface**:
- {What we consume from previous epic}
- {How we connect}

### Integration with Existing System
- **{System/Component}**: {How we integrate}

### Interfaces for Future Epics
**Provided for Epic {N+1}**:
- {What we expose for future epics}
- {Interface contracts}

---

## Data Design

### Database Changes (if applicable)
```sql
-- New tables/schema for this epic
CREATE TABLE {table_name} (
  {field1} {type},
  {field2} {type}
);
```

### Data Migration (if applicable)
- Migration: {What data needs to migrate}
- Strategy: {How to migrate safely}
- Rollback: {How to rollback if needed}

---

## Non-Functional Requirements

### Performance
- **Response Time**: {Target}
- **Throughput**: {Target}
- **Optimization Strategy**: {How to achieve}

### Security
- **Authentication**: {Requirements}
- **Authorization**: {Requirements}
- **Data Protection**: {Encryption, sanitization}
- **Audit**: {What to log}

### Scalability
- **Current Load**: {Expected load for this epic}
- **Growth**: {How to scale in future epics}

### Reliability
- **Availability**: {Target uptime}
- **Error Handling**: {Strategy}
- **Monitoring**: {What to monitor}

---

## Implementation Strategy

### Story Implementation Order

**Phase 1: Foundation** (Stories {X}-{Y})
{What to build first and why}

**Phase 2: Core Features** (Stories {X}-{Y})
{What to build second and why}

**Phase 3: Integration** (Stories {X}-{Y})
{What to build last and why}

### Story-Specific Guidance

#### Story-{XXX}: {Title}
- **Technical Approach**: {Specific guidance}
- **Key Files**: {Files to modify/create}
- **Integration Points**: {What to connect}
- **Acceptance Validation**: {How to verify}

---

## Testing Strategy

### Unit Testing
- **Coverage Target**: {X}%
- **Critical Paths**: {What must be tested}
- **Test Patterns**: {Follow existing patterns from repo scan}

### Integration Testing
- **Integration Points**: {What to test}
- **Test Scenarios**: {Key scenarios}

### E2E Testing
- **User Flows**: {Flows to test}
- **Test Environment**: {How to set up}

### Performance Testing (if applicable)
- **Load Tests**: {What to test}
- **Benchmarks**: {Performance targets}

---

## Risks and Mitigations

| Risk | Probability | Impact | Mitigation | Owner |
|------|-------------|--------|------------|-------|
| {Risk 1} | {Low|Med|High} | {Low|Med|High} | {How to mitigate} | Epic {N} |
| {Risk 2} | {Low|Med|High} | {Low|Med|High} | {How to mitigate} | Epic {N} |

### Known Technical Challenges
1. **{Challenge 1}**
   - Issue: {What's challenging}
   - Approach: {How to handle}
   - Fallback: {Plan B}

---

## Epic Boundaries

### In Scope (This Epic)
- {Feature 1}
- {Feature 2}

### Out of Scope (Future Epics)
- {Feature 1} → Epic {N+1}
- {Feature 2} → Epic {N+2}

### Deferred Decisions
- **{Decision}**: Defer to Epic {N+1} (Reason: {why defer})

---

## Success Criteria

### Functional Criteria
- [ ] {Criterion 1}
- [ ] {Criterion 2}

### Technical Criteria
- [ ] All interfaces defined and documented
- [ ] Integration tests pass
- [ ] Performance targets met
- [ ] Security requirements implemented
- [ ] Code review approved

### Epic Definition of Done
- [ ] All stories completed
- [ ] Retrospective conducted
- [ ] Learnings documented
- [ ] Interfaces for next epic defined

---

## References

### Related Artifacts
- **High-Level Architecture**: `.claude/specs/{feature}/02-system-architecture.md`
- **Sprint Plan**: `.claude/specs/{feature}/03-sprint-plan.md`
- **Previous Retrospective**: `.claude/specs/{feature}/retrospective-epic-{N-1}.md`
- **Repository Scan**: `.claude/specs/{feature}/00-repo-scan.md`

### External Documentation
- {Link to external docs if applicable}

---

## Appendix: Technical Spikes (if needed)

### Spike 1: {Spike Title}
**Question**: {What we need to learn}
**Approach**: {How to investigate}
**Time Box**: {X hours}
**Decision Point**: {What decision this spike informs}

---

*Epic {N} Technical Specification*
*Generated: {timestamp}*
*Next: Implement stories in order, conduct retrospective, create spec for Epic {N+1}*
```

### 5. Save Epic Technical Specification

```
Use Write tool:
Path: .claude/specs/{feature}/tech-spec-epic-{N}.md
Content: Generated epic technical specification
```

### 6. Update Workflow Status

```
Use Edit tool:
Path: .claude/workflow-status.md

Update:
- Current phase: Epic {N} Implementation
- Add epic tech spec to completed artifacts
```

### 7. Generate Summary

```markdown
# Epic {N} Technical Specification Complete ✓

**Epic**: Epic-{N} - {Title}
**Feature**: {feature_name}
**Specification**: `.claude/specs/{feature}/tech-spec-epic-{N}.md`

## Specification Summary
{2-3 sentence summary of technical approach}

## Key Technical Decisions
1. **{Decision 1}**: {Choice} (Rationale: {why})
2. **{Decision 2}**: {Choice} (Rationale: {why})

## Components in This Epic
- {Component 1}: {Role}
- {Component 2}: {Role}

## Learnings Applied from Epic {N-1}
✓ {Pattern replicated}
✗ {Anti-pattern avoided}
🔄 {Architectural adjustment}

## Stories Ready for Implementation
- Story-{XXX}: {Title}
- Story-{XXX}: {Title}
({X} stories total)

## Integration Points
- Previous epic: {What we consume}
- Next epic: {What we provide}

## Risks Identified
- {Risk 1}: {Mitigation}
- {Risk 2}: {Mitigation}

---

**Ready to implement Epic {N}?**

**Start Implementation**:
```bash
/bmad-dev-epic {N}
```

**Review Specification**:
```bash
# View: .claude/specs/{feature}/tech-spec-epic-{N}.md
```

**What Comes After**:
1. Implement all stories in Epic {N}
2. Conduct retrospective: `/bmad-retrospective {N}`
3. Create spec for Epic {N+1}: `/bmad-architect-epic {N+1}`

---

*JIT Architecture: Decide at the last responsible moment, learn from implementation*
```

## Key JIT Principles

### 1. Last Responsible Moment
- Make technical decisions when you have enough information
- Don't make decisions that will likely change
- Defer decisions to later epics when possible

### 2. Incorporate Learnings
- Every epic teaches us something
- Apply successful patterns from previous epics
- Avoid repeating mistakes

### 3. Right-Sized Specification
- Detailed enough for THIS epic
- Not over-specified for future epics
- Focus on actionable information

### 4. Adaptive Architecture
- Architecture evolves based on learnings
- High-level architecture provides direction
- Epic-specific specs provide details

## Benefits of JIT Architecture

### Prevents Over-Engineering
- Don't design features that might change
- Focus on current epic needs
- Adapt based on real feedback

### Incorporates Learning
- Apply lessons from Epic 1 to Epic 2
- Patterns emerge from implementation
- Architecture improves iteratively

### Reduces Rework
- Decisions made with better information
- Less chance of outdated decisions
- More agile architecture

### Maintains Focus
- Dev team focuses on one epic at a time
- Clearer context and scope
- Less cognitive overload

## Error Handling

### Epic Not Found
```markdown
❌ **Error**: Epic-{N} not found in sprint plan

**Available Epics**:
{List epics from sprint plan}

**Usage**: `/bmad-architect-epic <EPIC_NUMBER>`
```

### Epic Already Completed
```markdown
⚠️ **Warning**: Epic-{N} already completed

**Status**: All stories in DONE state
**Retrospective**: {exists|missing}

**Recommendation**:
- If starting new epic: `/bmad-architect-epic {N+1}`
- If epic needs rework: Review retrospective first
```

### Missing Prerequisites
```markdown
❌ **Error**: Missing required artifacts

**Missing**:
- {Missing artifact 1}
- {Missing artifact 2}

**Resolution**: Run `/bmad-pilot` to generate high-level architecture first

**Note**: JIT architecture requires high-level architecture as foundation
```

## Success Criteria
- Epic technical specification generated
- Learnings from previous epic incorporated
- Technical decisions made at appropriate level
- Integration points defined
- Risks identified and mitigated
- Specification saved correctly
- Ready for story implementation

## Example Output

```markdown
# Epic 2 Technical Specification Complete ✓

**Epic**: Epic-2 - Payment Processing
**Feature**: e-commerce-checkout
**Specification**: `.claude/specs/e-commerce-checkout/tech-spec-epic-2.md`

## Specification Summary
Implement payment processing with Stripe integration, supporting credit cards and digital wallets. Build on Epic 1's cart system. Use payment gateway abstraction to allow future payment providers.

## Key Technical Decisions
1. **Payment Provider**: Stripe (Rationale: Well-documented API, Epic 1 showed we need robust error handling)
2. **Payment Abstraction**: Gateway pattern (Rationale: Epic 1 retrospective recommended flexibility for future providers)
3. **Transaction Storage**: Separate payments DB (Rationale: Epic 1 showed cart DB getting large, separate for scalability)

## Components in This Epic
- PaymentGateway: Abstract payment interface
- StripeAdapter: Stripe-specific implementation
- PaymentService: Business logic for payments
- TransactionLogger: Audit trail (learned from Epic 1 debugging needs)

## Learnings Applied from Epic 1
✓ **Gateway Pattern**: Worked well for cart persistence, replicate for payments
✗ **Inline Error Handling**: Made Epic 1 code messy, use centralized error handler
🔄 **Logging Strategy**: Epic 1 lacked audit trail, add comprehensive logging here

## Stories Ready for Implementation
- Story-010: Payment gateway abstraction
- Story-011: Stripe integration
- Story-012: Payment UI components
- Story-013: Transaction logging
- Story-014: Error handling and retries
(5 stories total)

## Integration Points
- Previous epic: Consume Cart interface from Epic 1
- Next epic: Provide Payment confirmation for Epic 3 (order fulfillment)

## Risks Identified
- Stripe API changes (Low probability): Pin to specific API version
- Payment failures (Medium impact): Implement retry logic with exponential backoff

---

**Ready to implement Epic 2?**

**Start Implementation**:
```bash
/bmad-dev-epic 2
```

**Review Specification**:
`.claude/specs/e-commerce-checkout/tech-spec-epic-2.md`

**What Comes After**:
1. Implement Stories 010-014
2. Conduct retrospective: `/bmad-retrospective 2`
3. Create spec for Epic 3: `/bmad-architect-epic 3`

---

*JIT Architecture: Leveraging Epic 1 learnings for better Epic 2 design*
```
