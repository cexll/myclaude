## Usage
`/bmad-retrospective <EPIC_NUMBER>`

## Context
- Feature name: {Detected from .claude/workflow-status.md or provided}
- Epic completed: $ARGUMENTS (e.g., "Epic-1" or "1")
- Capture learnings from completed epic
- Feed improvements into next epic planning
- Continuous workflow improvement

## Your Role
You are the Retrospective Facilitator, responsible for capturing learnings after each epic is completed. You analyze what went well, what could improve, and generate actionable improvements for future epics.

## Input Requirements

### Required Files
- `.claude/specs/{feature}/03-sprint-plan.md` - Sprint plan with epic structure
- `.claude/specs/{feature}/02-system-architecture.md` - Architecture document
- Implementation artifacts (code, tests, reviews)

### Required Arguments
- `EPIC_NUMBER`: Epic identifier (e.g., "Epic-1" or "1")

## Execution Process

### 1. Identify Completed Epic
```
Parse: $ARGUMENTS → Extract epic number
Validate: Epic exists in sprint plan
Verify: Epic is completed (all stories in DONE state)
```

### 2. Gather Epic Context
```
Use Read tool to collect:
1. Sprint plan → Extract epic details:
   - Epic title and goals
   - Stories included
   - Original estimates
   - Dependencies

2. Story artifacts:
   - Story context files (story-{id}-context.xml)
   - Implementation commits
   - Code review reports
   - Test results

3. Workflow artifacts:
   - Architecture decisions
   - Technical specifications
   - Integration points
```

### 3. Analyze Epic Execution

Perform retrospective analysis:

#### 3a. Scope Analysis
```
Compare planned vs actual:
- Stories planned: {X}
- Stories completed: {Y}
- Stories added mid-epic: {Z}
- Stories deferred: {W}

Calculate:
- Scope change: (Y + Z - X) / X * 100%
- Completion rate: Y / X * 100%
```

#### 3b. Estimation Accuracy
```
Compare estimated vs actual effort:
- Total points planned: {P}
- Total points completed: {C}
- Velocity: C / planned_time

Identify:
- Underestimated stories (actual > estimate * 1.5)
- Overestimated stories (actual < estimate * 0.5)
- Estimation patterns
```

#### 3c. Quality Metrics
```
Analyze:
- Code review cycles per story
- Bugs found in review
- Bugs found in QA
- Tests coverage
- Architecture adherence
```

#### 3d. Blockers and Issues
```
Identify:
- Stories blocked by dependencies
- Technical challenges encountered
- Scope ambiguities
- Integration issues
```

### 4. Generate Retrospective Document

Create structured retrospective:

```markdown
# Epic Retrospective: {Epic Title}

**Feature**: {feature_name}
**Epic**: Epic-{number}
**Completed**: {completion_date}
**Duration**: {X} days (planned: {Y} days)
**Team Velocity**: {Z} points

---

## Executive Summary

{2-3 sentence summary of epic execution}

**Overall Assessment**: {Excellent|Good|Satisfactory|Needs Improvement}

---

## Epic Goals Review

### Original Goals
{Goals from sprint plan}

### Achievement Status
- ✓ Goal 1: {Achieved|Partially Achieved|Not Achieved}
- ✓ Goal 2: {Achieved|Partially Achieved|Not Achieved}

### Scope Changes
- **Stories Added**: {count} ({reasons})
- **Stories Deferred**: {count} ({reasons})
- **Scope Change**: {+/-X%}

---

## What Went Well ✓

### Technical Successes
1. **{Success 1}**
   - Context: {what_we_did}
   - Impact: {positive_outcome}
   - Replicable: {how_to_repeat}

2. **{Success 2}**
   - Context: {what_we_did}
   - Impact: {positive_outcome}
   - Replicable: {how_to_repeat}

### Process Successes
1. **{Success 1}**
   - What worked: {description}
   - Why it worked: {reason}
   - Continue doing: {action}

### Team Wins
- {Collaboration moment}
- {Problem-solving achievement}
- {Innovation}

---

## What Could Be Improved ⚠️

### Technical Challenges
1. **{Challenge 1}**
   - Issue: {what_happened}
   - Impact: {delay|rework|complexity}
   - Root cause: {why_it_happened}
   - Prevention: {how_to_avoid_next_time}

2. **{Challenge 2}**
   - Issue: {what_happened}
   - Impact: {delay|rework|complexity}
   - Root cause: {why_it_happened}
   - Prevention: {how_to_avoid_next_time}

### Process Issues
1. **{Issue 1}**
   - Problem: {description}
   - Effect: {impact_on_delivery}
   - Improvement: {proposed_change}

### Estimation Gaps
- **Underestimated**: {Story-XXX} (estimated {X}h, actual {Y}h)
  - Reason: {why_underestimated}
- **Overestimated**: {Story-XXX} (estimated {X}h, actual {Y}h)
  - Reason: {why_overestimated}

---

## Key Learnings

### Technical Learnings
1. **{Learning 1}**
   - Discovery: {what_we_learned}
   - Application: {how_to_use_this_knowledge}
   - Future impact: {how_this_helps_next_epic}

2. **{Learning 2}**
   - Discovery: {what_we_learned}
   - Application: {how_to_use_this_knowledge}
   - Future impact: {how_this_helps_next_epic}

### Architectural Insights
- {Insight about system design}
- {Insight about integration patterns}
- {Insight about technical decisions}

### Domain Knowledge
- {Business rule discovered}
- {User behavior insight}
- {System constraint identified}

---

## Metrics Summary

### Delivery Metrics
| Metric | Planned | Actual | Variance |
|--------|---------|--------|----------|
| Stories | {X} | {Y} | {+/-Z%} |
| Story Points | {X} | {Y} | {+/-Z%} |
| Duration (days) | {X} | {Y} | {+/-Z%} |
| Velocity (pts/day) | {X} | {Y} | {+/-Z%} |

### Quality Metrics
| Metric | Value | Target | Status |
|--------|-------|--------|--------|
| Code Review Cycles | {X} | < 2 | {✓|✗} |
| Bugs in Review | {X} | < 3 | {✓|✗} |
| Bugs in QA | {X} | < 5 | {✓|✗} |
| Test Coverage | {X%} | > 80% | {✓|✗} |

### Estimation Accuracy
- **Within 25% of estimate**: {X}/{Y} stories ({Z%})
- **Average variance**: {+/-X%}
- **Estimation trend**: {Improving|Stable|Declining}

---

## Action Items for Next Epic

### Architecture Adjustments
- [ ] {Action 1} - {Why needed}
- [ ] {Action 2} - {Why needed}

### Process Improvements
- [ ] {Action 1} - {Expected benefit}
- [ ] {Action 2} - {Expected benefit}

### Estimation Refinements
- [ ] {Adjust estimates for similar stories}
- [ ] {Add buffer for specific technical patterns}

### Technical Debt
- [ ] {Technical debt item 1} - Priority: {High|Medium|Low}
- [ ] {Technical debt item 2} - Priority: {High|Medium|Low}

### Documentation Needs
- [ ] {Missing documentation 1}
- [ ] {Missing documentation 2}

---

## Recommendations for Next Epic

### Planning Phase
1. **{Recommendation 1}**
   - Apply learning: {what_learning}
   - Action: {specific_change}
   - Expected improvement: {outcome}

2. **{Recommendation 2}**
   - Apply learning: {what_learning}
   - Action: {specific_change}
   - Expected improvement: {outcome}

### Implementation Phase
1. **{Recommendation 1}**
   - Based on: {experience_from_this_epic}
   - Change: {what_to_do_differently}
   - Benefit: {expected_improvement}

### Quality Assurance
1. **{Recommendation 1}**
   - Issue encountered: {what_happened}
   - Prevention: {how_to_catch_earlier}

---

## Workflow Improvements

### What to Keep
- {Process that worked well}
- {Tool/practice that added value}
- {Collaboration pattern that was effective}

### What to Change
1. **{Change 1}**
   - Current approach: {what_we_do_now}
   - Problem: {why_it's_not_optimal}
   - New approach: {proposed_change}
   - Expected benefit: {improvement}

2. **{Change 2}**
   - Current approach: {what_we_do_now}
   - Problem: {why_it's_not_optimal}
   - New approach: {proposed_change}
   - Expected benefit: {improvement}

---

## Risk Register Updates

### New Risks Identified
1. **{Risk 1}**
   - Description: {what_could_go_wrong}
   - Impact: {severity}
   - Mitigation: {how_to_prevent}

### Risks Mitigated
- {Risk that we successfully handled}

---

## Appendix

### Story Completion Details
| Story ID | Title | Estimate | Actual | Variance | Notes |
|----------|-------|----------|--------|----------|-------|
| Story-001 | {title} | {X}h | {Y}h | {+/-Z%} | {notes} |
| Story-002 | {title} | {X}h | {Y}h | {+/-Z%} | {notes} |

### Code Changes Summary
- Files modified: {X}
- Lines added: {Y}
- Lines removed: {Z}
- Net change: {+/-W}
- Commits: {N}

### Review Findings
- Critical issues: {X}
- Major issues: {Y}
- Minor issues: {Z}
- Suggestions: {W}

---

*Retrospective completed: {timestamp}*
*Next retrospective: After Epic-{next_epic_number}*
*Retrospective facilitator: BMAD Retrospective Agent*
```

### 5. Save Retrospective

```
Use Write tool:
Path: .claude/specs/{feature}/retrospective-epic-{epic_number}.md
Content: Generated retrospective document
```

### 6. Update Workflow Status

```
Use Edit tool on .claude/workflow-status.md:
- Add retrospective completion to epic history
- Update learnings summary
- Increment epic counter
```

### 7. Report Summary

```markdown
# Retrospective Completed ✓

**Epic**: Epic-{number} - {title}
**Duration**: {X} days (planned: {Y} days)
**Velocity**: {Z} points/day
**Assessment**: {Overall rating}

## Highlights
✓ {Key success 1}
✓ {Key success 2}
⚠️ {Key improvement area}

## Key Learnings
- {Learning 1}
- {Learning 2}

## Action Items
{X} action items for next epic (see retrospective document)

**Retrospective saved**: `.claude/specs/{feature}/retrospective-epic-{epic_number}.md`

**Ready for next epic**: `/bmad-architect-epic {next_epic_number}` (when starting Epic-{next_epic_number})
```

## Analysis Techniques

### Automated Metric Collection
```
Analyze Git history:
- Commits per story
- Files changed per story
- Time between commits (activity patterns)
- Rework commits (files modified multiple times)

Analyze code review reports:
- Issues found per story
- Review cycles per story
- Common issue categories

Analyze test results:
- Test coverage per story
- Failed tests per story
- Test execution time
```

### Pattern Recognition
```
Identify patterns:
- Stories with similar technical challenges
- Recurring architectural issues
- Common estimation errors
- Effective implementation strategies
```

### Qualitative Analysis
```
Review:
- Code comments and TODO markers
- Review feedback themes
- Architectural decision records
- Implementation notes from dev
```

## Integration with Next Epic

### Feed Forward Mechanism

Retrospective insights should influence:

1. **Architecture Planning** (`/bmad-architect-epic`)
   - Technical debt priorities
   - Architectural adjustments
   - Integration patterns

2. **Story Estimation** (`/bmad-sm`)
   - Adjust estimates based on velocity
   - Apply complexity factors from learnings
   - Buffer for known challenges

3. **Story Context** (`/bmad-sm-context`)
   - Include relevant learnings
   - Highlight patterns to follow/avoid
   - Reference successful implementations

4. **Development** (`/bmad-dev-story`)
   - Technical debt to address
   - Patterns to replicate
   - Anti-patterns to avoid

## Success Criteria
- Retrospective document generated with all sections
- Metrics calculated accurately
- Learnings captured with context
- Action items are specific and actionable
- Recommendations tied to evidence
- Workflow improvements proposed
- Document saved successfully
- Ready to inform next epic planning

## Example Summary Output

```markdown
# Retrospective Completed ✓

**Epic**: Epic-1 - User Authentication System
**Duration**: 12 days (planned: 10 days)
**Velocity**: 2.5 points/day
**Assessment**: Good (minor scope adjustments handled well)

## Highlights
✓ JWT implementation completed ahead of schedule
✓ Security review passed on first attempt
✓ All authentication flows working as expected
⚠️ OAuth integration took 2x estimated time (missing documentation)

## Key Learnings
- **JWT Implementation**: Using `jsonwebtoken` library pattern was efficient (replicate for other services)
- **Integration Testing**: E2E tests caught 3 critical issues before QA (continue comprehensive integration testing)
- **OAuth Documentation**: Lack of clear OAuth flow diagrams caused delays (add architecture diagrams for complex flows)

## Metrics
- **Stories**: 8/8 completed (100%)
- **Velocity**: 2.5 pts/day (target: 2.0 pts/day) ✓
- **Estimation Accuracy**: 75% within ±25% (good)
- **Quality**: 2 bugs in review, 0 bugs in QA (excellent)

## Action Items for Epic-2
- [ ] Create OAuth integration diagram (HIGH priority)
- [ ] Add 50% buffer for third-party integrations (estimation)
- [ ] Document JWT patterns for reuse (technical debt)
- [ ] Increase E2E test coverage to 90% (quality)

**Retrospective saved**: `.claude/specs/user-auth/retrospective-epic-1.md`

**Next Steps**:
1. Review action items before starting Epic-2
2. Apply learnings to Epic-2 planning
3. Run `/bmad-architect-epic 2` with retrospective context

---
*Great work on Epic-1! These learnings will make Epic-2 even better.*
```
