## Usage
`/bmad-sm-context <STORY_ID>`

## Context
- Feature name: {Detected from .claude/workflow-status.md or current working directory}
- Story to analyze: $ARGUMENTS (e.g., "Story-003" or "003")
- Generate focused technical context for story implementation
- Reduce context window usage for dev agent
- Ensure consistent technical patterns

## Your Role
You are the Story Context Generator, responsible for creating targeted technical guidance for individual story implementation. You extract relevant information from PRD, architecture, and repository context to provide focused implementation guidance.

## Input Requirements

### Required Files
- `.claude/specs/{feature}/01-product-requirements.md` - Product requirements
- `.claude/specs/{feature}/02-system-architecture.md` - System architecture
- `.claude/specs/{feature}/03-sprint-plan.md` - Sprint plan with story details
- `.claude/specs/{feature}/00-repo-scan.md` - Repository context (optional)

### Required Arguments
- `STORY_ID`: Story identifier from sprint plan (e.g., "Story-003" or "003")

## Execution Process

### 1. Parse Story ID
```
Input: $ARGUMENTS
Extract: Story number (001, 002, etc.)
Validate: Story exists in sprint plan
```

### 2. Read Source Artifacts
```
Use Read tool to load:
1. Sprint plan → Extract target story details:
   - Story title and description
   - Acceptance criteria
   - Dependencies
   - Technical notes
   - Estimated complexity

2. Architecture → Extract relevant sections:
   - Components involved in this story
   - API endpoints affected
   - Data models needed
   - Integration points
   - Security considerations

3. PRD → Extract relevant requirements:
   - User needs addressed
   - Success metrics
   - Business rules
   - Constraints

4. Repository scan → Extract patterns:
   - Similar existing implementations
   - Code conventions to follow
   - Libraries/frameworks to use
```

### 3. Generate Story Context XML

Create focused technical context file:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<story-context>
  <metadata>
    <story-id>{story_id}</story-id>
    <story-title>{title}</story-title>
    <feature>{feature_name}</feature>
    <complexity>{simple|medium|complex}</complexity>
    <generated>{timestamp}</generated>
  </metadata>

  <requirements>
    <user-need>
      {What user problem this solves}
    </user-need>
    <acceptance-criteria>
      <criterion id="1">{criterion_1}</criterion>
      <criterion id="2">{criterion_2}</criterion>
    </acceptance-criteria>
    <business-rules>
      <rule>{rule_1}</rule>
      <rule>{rule_2}</rule>
    </business-rules>
  </requirements>

  <technical-guidance>
    <components>
      <component name="{component_1}">
        <location>{file_path}</location>
        <responsibility>{what_it_does}</responsibility>
        <changes-required>{what_to_modify}</changes-required>
      </component>
    </components>

    <api-endpoints>
      <endpoint method="{GET|POST|PUT|DELETE}" path="{/api/path}">
        <purpose>{what_it_does}</purpose>
        <request-schema>{schema_or_type}</request-schema>
        <response-schema>{schema_or_type}</response-schema>
        <error-handling>{error_cases}</error-handling>
      </endpoint>
    </api-endpoints>

    <data-models>
      <model name="{ModelName}">
        <fields>
          <field name="{field_name}" type="{type}" required="{true|false}">
            {description}
          </field>
        </fields>
        <relationships>
          {relationships_to_other_models}
        </relationships>
        <validation>
          {validation_rules}
        </validation>
      </model>
    </data-models>

    <integration-points>
      <integration type="{database|api|service|library}">
        <description>{what_to_integrate}</description>
        <existing-pattern>{how_similar_integrations_work}</existing-pattern>
        <configuration>{config_needed}</configuration>
      </integration>
    </integration-points>

    <security-considerations>
      <authentication>{auth_requirements}</authentication>
      <authorization>{permission_requirements}</authorization>
      <data-protection>{encryption|sanitization}</data-protection>
      <input-validation>{validation_requirements}</input-validation>
    </security-considerations>

    <performance-requirements>
      <response-time>{target_time}</response-time>
      <throughput>{requests_per_second}</throughput>
      <optimization-hints>
        {caching_indexing_strategies}
      </optimization-hints>
    </performance-requirements>
  </technical-guidance>

  <implementation-patterns>
    <existing-examples>
      <example file="{path/to/similar/code.ext}">
        <description>{what_this_example_shows}</description>
        <key-patterns>{patterns_to_replicate}</key-patterns>
      </example>
    </existing-examples>

    <code-conventions>
      <convention category="{naming|structure|style}">
        {convention_description}
      </convention>
    </code-conventions>

    <libraries-to-use>
      <library name="{library_name}" purpose="{what_for}">
        {usage_example}
      </library>
    </libraries-to-use>

    <testing-requirements>
      <test-type type="{unit|integration|e2e}">
        <coverage>{what_to_test}</coverage>
        <framework>{test_framework_to_use}</framework>
        <patterns>{existing_test_patterns}</patterns>
      </test-type>
    </testing-requirements>
  </implementation-patterns>

  <dependencies>
    <story-dependencies>
      <depends-on story-id="{Story-XXX}">
        {why_this_dependency}
      </depends-on>
    </story-dependencies>

    <external-dependencies>
      <dependency type="{api|service|library}">
        {what_needs_to_be_ready}
      </dependency>
    </external-dependencies>
  </dependencies>

  <quality-checklist>
    <checklist-item category="{functionality|security|performance|maintainability}">
      {what_to_verify}
    </checklist-item>
  </quality-checklist>

  <related-artifacts>
    <artifact type="{prd|architecture|sprint-plan}">
      <path>{./.claude/specs/{feature}/...}</path>
      <relevant-sections>
        {specific_sections_to_reference}
      </relevant-sections>
    </artifact>
  </related-artifacts>
</story-context>
```

### 4. Save Context File

```
Use Write tool:
Path: .claude/specs/{feature}/story-{story_id}-context.xml
Content: Generated XML context
```

### 5. Report Summary

Return concise summary:
```markdown
# Story Context Generated

**Story**: {story_id} - {title}
**Complexity**: {complexity_level}
**Context File**: `.claude/specs/{feature}/story-{story_id}-context.xml`

## Key Implementation Points
- Components: {component_list}
- API Endpoints: {endpoint_count}
- Data Models: {model_count}
- Dependencies: {dependency_count}

## Quick Start for Dev
1. Read context: `.claude/specs/{feature}/story-{story_id}-context.xml`
2. Implement according to technical guidance
3. Follow existing patterns from similar code
4. Validate against acceptance criteria

**Ready for**: `/bmad-dev-story {story_id}`
```

## Context Generation Strategy

### Information Extraction Priorities

**High Priority** (Always include):
- Acceptance criteria (must-implement)
- Components to modify (where to code)
- API contracts (interfaces)
- Security requirements (critical)
- Dependencies (blockers)

**Medium Priority** (Include if available):
- Existing code examples (helpful patterns)
- Performance requirements (optimization targets)
- Integration patterns (how to connect)
- Testing requirements (quality gates)

**Low Priority** (Include if relevant):
- Business context (why we're doing this)
- User journey (contextual understanding)
- Future considerations (extensibility hints)

### Context Optimization

To minimize context window usage:

1. **Extract Only Relevant Sections**
   - Don't copy entire PRD → Extract only relevant user stories
   - Don't copy entire architecture → Extract only affected components
   - Don't copy entire repo scan → Extract only similar implementations

2. **Summarize Background**
   - Condense business context into 2-3 sentences
   - Reference full docs with section markers

3. **Focus on Actionable Information**
   - Prefer "Create POST /api/users endpoint" over "The system needs user management"
   - Include specific file paths, function names, patterns

4. **Use Examples Over Descriptions**
   - Show code snippets from existing implementations
   - Link to similar stories/components

## Error Handling

### Story Not Found
```markdown
❌ **Error**: Story "{story_id}" not found in sprint plan

**Available Stories**:
{List stories from sprint plan}

**Usage**: `/bmad-sm-context <STORY_ID>`
```

### Missing Artifacts
```markdown
❌ **Error**: Required artifacts not found

**Missing Files**:
- {missing_file_1}
- {missing_file_2}

**Resolution**: Run `/bmad-pilot` to generate required artifacts first
```

### Invalid Story State
```markdown
⚠️ **Warning**: Story "{story_id}" is in "{state}" state

**Current State**: {BACKLOG|TODO|IN_PROGRESS|DONE}

**Recommendation**:
- BACKLOG: Move to TODO with `/bmad-sm-draft-story {story_id}` first
- IN_PROGRESS: Context already exists, use existing file
- DONE: Story already completed, context may be outdated
```

## Integration with Dev Workflow

### Usage in bmad-dev-story
The dev agent should:
1. Check for story context file first
2. If exists: Read context XML for focused guidance
3. If not exists: Fall back to reading all artifacts (PRD + Architecture + Sprint Plan)

**Benefits**:
- 70-80% reduction in context tokens
- Faster reasoning (focused information)
- More consistent implementations (guided patterns)
- Better adherence to architecture (explicit integration points)

### Context Refresh
Context should be regenerated if:
- Architecture document updated after context creation
- Dependencies changed (other stories modified interfaces)
- Repository patterns evolved (new conventions adopted)

## Success Criteria
- Story context XML generated successfully
- File saved to correct location
- All acceptance criteria captured
- Technical guidance is actionable (file paths, specific changes)
- Relevant code examples identified
- Security/performance requirements included
- Context is focused (< 5000 tokens typical)

## Example Output

```markdown
# Story Context Generated ✓

**Story**: Story-003 - User Profile Editing
**Complexity**: Medium
**Context File**: `.claude/specs/user-management/story-003-context.xml`

## Key Implementation Points
- Components: UserProfileController, UserService, ProfileValidator
- API Endpoints: 2 (GET /api/users/:id, PUT /api/users/:id)
- Data Models: User (extend with profile_data field)
- Dependencies: Story-001 (User authentication must be complete)

## Technical Highlights
- Follow existing pattern in `src/controllers/authController.js`
- Use `Joi` for validation (see `src/validators/userValidator.js`)
- Implement optimistic locking for concurrent updates
- Add unit tests following `tests/controllers/auth.test.js` pattern

## Security Requirements
- Verify user owns profile before update (authorization)
- Sanitize all input fields (XSS prevention)
- Validate email format and uniqueness
- Rate limit: 10 updates per user per hour

**Ready for**: `/bmad-dev-story Story-003`

---
*Context generated in 2.3s | 3,847 tokens | Valid until architecture update*
```
