## Usage
`/bmad-sm-draft-story <STORY_ID>`

## Context
- Story state machine transition: BACKLOG → TODO
- Generate detailed story draft ready for user approval
- Feature detected from .claude/workflow-status.md
- Part of 4-state story lifecycle

## Your Role
You are the Story Drafter, responsible for moving stories from BACKLOG to TODO state by creating detailed story specifications ready for development approval.

## Story State Machine

```
BACKLOG → TODO → IN PROGRESS → DONE
   ↑        ↑          ↑          ↑
   |        |          |          |
 Planned  Drafted  Approved  Completed
```

### State Definitions

**BACKLOG**: Ordered list of stories to be drafted
- Status: Planned but not detailed
- Action needed: Draft story details

**TODO**: Single story ready for drafting (or drafted, awaiting approval)
- Status: Detailed specification created
- Action needed: User review and approval

**IN PROGRESS**: Single story approved for development
- Status: Currently being implemented
- Action needed: Development and testing

**DONE**: Completed stories with dates and points
- Status: Implementation complete, tested, reviewed
- Action needed: None (archived)

## Execution Process

### 1. Validate Story ID

```
Input: $ARGUMENTS (e.g., "Story-003" or "003")
Extract: Story number
Validate: Story exists in sprint plan BACKLOG section
Check: Story not already in TODO, IN PROGRESS, or DONE
```

### 2. Read Sprint Plan

```
Use Read tool:
Path: .claude/specs/{feature}/03-sprint-plan.md

Extract:
- Story title and description from BACKLOG
- Epic association
- Initial estimate
- Dependencies
```

### 3. Read Context Documents

```
Use Read tool:
Paths:
- .claude/specs/{feature}/01-product-requirements.md (user needs)
- .claude/specs/{feature}/02-system-architecture.md (technical context)
- .claude/specs/{feature}/00-repo-scan.md (implementation patterns)
```

### 4. Generate Detailed Story Specification

Create comprehensive story draft:

```markdown
## Story-{ID}: {Title}

**Epic**: {Epic name}
**State**: TODO (Draft - Awaiting Approval)
**Estimated Points**: {X}
**Dependencies**: {Story-XXX, Story-YYY or None}
**Priority**: {High|Medium|Low}

### User Story
As a {user_type}
I want {functionality}
So that {business_value}

### Description
{Detailed description of what needs to be implemented, 2-3 paragraphs}

### Acceptance Criteria
- [ ] {Specific, testable criterion 1}
- [ ] {Specific, testable criterion 2}
- [ ] {Specific, testable criterion 3}
- [ ] {Specific, testable criterion 4}

### Technical Implementation Notes

#### Components to Modify/Create
- **{Component 1}** ({path/to/file}): {What to do}
- **{Component 2}** ({path/to/file}): {What to do}

#### API Changes (if applicable)
- **Endpoint**: {METHOD} {/api/path}
  - Request: {schema}
  - Response: {schema}

#### Database Changes (if applicable)
- **Table**: {table_name}
  - Changes: {what_to_modify}

#### Integration Points
- {Integration 1}: {How to connect}
- {Integration 2}: {How to connect}

### Implementation Steps
1. {Step 1}
2. {Step 2}
3. {Step 3}

### Testing Requirements

#### Unit Tests
- {Test case 1}
- {Test case 2}

#### Integration Tests
- {Test case 1}

#### Manual Testing
- [ ] {Manual test scenario 1}
- [ ] {Manual test scenario 2}

### Edge Cases & Error Handling
- **Edge Case 1**: {Scenario} → {Expected behavior}
- **Error Condition 1**: {Condition} → {Error handling}

### Definition of Done
- [ ] Code implemented following conventions
- [ ] Unit tests written and passing (>80% coverage)
- [ ] Integration tests passing
- [ ] Code reviewed and approved
- [ ] Acceptance criteria validated
- [ ] Documentation updated
- [ ] No new warnings or errors

### Dependencies
**Depends On**:
- {Story-XXX}: {Why this dependency}

**Blocks**:
- {Story-YYY}: {What this story provides}

### Estimated Effort
- **Story Points**: {X}
- **Time Estimate**: {Y hours}
- **Confidence**: {High|Medium|Low}

### Risks
- **{Risk 1}**: {Description} (Mitigation: {strategy})

### References
- PRD Section: {Section reference}
- Architecture Section: {Section reference}
- Similar Implementation: {path/to/example}

---

*Story drafted by bmad-sm-draft-story*
*Ready for user approval to move to IN PROGRESS*
```

### 5. Update Sprint Plan State Machine

```
Use Edit tool:
Path: .claude/specs/{feature}/03-sprint-plan.md

Change:
Find story in BACKLOG section
Move to TODO section with "Status: Draft - Awaiting Approval"

Example:
From:
### BACKLOG
- Story-003: User Profile Edit

To:
### TODO
- Story-003: User Profile Edit (Status: Draft - Awaiting Approval)

Also update the detailed story section in sprint plan with full specification
```

### 6. Save Story Draft

```
Use Write tool:
Path: .claude/specs/{feature}/story-{ID}-draft.md
Content: Detailed story specification
```

### 7. Update Workflow Status

```
Use Edit tool:
Path: .claude/workflow-status.md

Update story state section:
Move story from BACKLOG to TODO
```

### 8. Present Draft to User

```markdown
# Story Draft Complete ✓

**Story**: Story-{ID} - {Title}
**Epic**: {Epic name}
**State**: TODO (Awaiting Approval)
**Estimated Effort**: {X} points ({Y} hours)

## Summary
{1-2 sentence summary of story}

## Acceptance Criteria
- {Criterion 1}
- {Criterion 2}
- {Criterion 3}

## Implementation Approach
{2-3 sentence summary of technical approach}

**Components**: {X} components affected
**API Changes**: {Y} endpoints
**Tests Required**: {Z} test cases

---

**🛑 User Approval Required**

**Review Draft**:
`.claude/specs/{feature}/story-{ID}-draft.md`

**Approve Story**:
```bash
/bmad-sm-approve-story {ID}
```

This will move the story to IN PROGRESS and start implementation.

**Adjust Story**:
Provide feedback for refinement, then re-run `/bmad-sm-draft-story {ID}`

**Dependencies**:
{List dependencies if any - these must be completed first}

---

*Story moved from BACKLOG → TODO*
*Awaiting your approval to begin development*
```

## Story Draft Quality Criteria

### Good Story Draft Checklist
- [ ] Clear user value (As a... I want... So that...)
- [ ] 3-5 specific, testable acceptance criteria
- [ ] Technical implementation is actionable (file paths, specific changes)
- [ ] Edge cases and error handling defined
- [ ] Testing requirements specified
- [ ] Dependencies identified
- [ ] Estimated appropriately (4-40 hours)
- [ ] References to PRD/Architecture included

## Error Handling

### Story Not in BACKLOG
```markdown
❌ **Error**: Story-{ID} not found in BACKLOG

**Current State**: {current_state}

**Resolution**:
- If in TODO: Already drafted, use `/bmad-sm-approve-story {ID}`
- If in IN PROGRESS: Already being developed
- If in DONE: Already completed
- If not exists: Check story ID in sprint plan
```

### Missing Prerequisites
```markdown
❌ **Error**: Cannot draft story - missing dependencies

**Dependency Status**:
- Story-{X}: {BACKLOG|TODO|IN PROGRESS} (must be DONE)

**Recommendation**:
Complete dependency first: `/bmad-sm-draft-story {X}`
Then return to: `/bmad-sm-draft-story {ID}`
```

### Story Already Drafted
```markdown
⚠️ **Warning**: Story-{ID} already in TODO state

**Current Status**: {Status description}

**Options**:
- Approve existing draft: `/bmad-sm-approve-story {ID}`
- Re-draft with updates: Continue with current command
- View draft: `.claude/specs/{feature}/story-{ID}-draft.md`
```

## Integration with Story Context

### Relationship to /bmad-sm-context

**Draft Story** (this command):
- Moves story BACKLOG → TODO
- Creates detailed specification
- Focuses on WHAT to build

**Story Context** (/bmad-sm-context):
- Called after approval (IN PROGRESS state)
- Creates implementation guidance (XML)
- Focuses on HOW to build

**Workflow**:
```
1. /bmad-sm-draft-story {ID}  → Creates draft (BACKLOG → TODO)
2. User approves                → Story moves to IN PROGRESS
3. /bmad-sm-context {ID}        → Creates context XML
4. /bmad-dev-story {ID}         → Implements story
```

## Success Criteria
- Story specification created with all required sections
- Story moved from BACKLOG to TODO in sprint plan
- Story draft file saved
- Workflow status updated
- User presented with clear approval options
- Quality criteria met

## Example Output

```markdown
# Story Draft Complete ✓

**Story**: Story-007 - User Avatar Upload
**Epic**: User Profile Management
**State**: TODO (Awaiting Approval)
**Estimated Effort**: 8 points (12 hours)

## Summary
Enable users to upload and manage profile avatars with automatic image optimization and validation.

## Acceptance Criteria
- User can upload image files (JPG, PNG) up to 5MB
- Image automatically resized to 300x300px
- Preview shown before confirmation
- Old avatar deleted when new one uploaded
- Avatar displayed across all user interactions

## Implementation Approach
Create AvatarUpload component using existing ImageUpload pattern from repository. Add POST /api/users/avatar endpoint with multipart form handling. Store avatars in S3 (existing integration). Update User model to include avatar_url field.

**Components**: 3 components (AvatarUpload UI, AvatarController API, AvatarService storage)
**API Changes**: 2 endpoints (POST /api/users/avatar, DELETE /api/users/avatar)
**Tests Required**: 8 test cases (upload validation, resize, delete, error handling)

---

**🛑 User Approval Required**

**Review Draft**:
`.claude/specs/user-profile/story-007-draft.md`

**Approve Story**:
```bash
/bmad-sm-approve-story 007
```

This will move the story to IN PROGRESS and start implementation.

**Dependencies**:
- Story-005 (User Profile API): DONE ✓
- Story-006 (S3 Integration): DONE ✓

---

*Story moved from BACKLOG → TODO*
*Awaiting your approval to begin development*
```
