## Usage
`/workflow-status [OPTIONS]`

### Options
- `--reset`: Clear the workflow status and start fresh

## Context
- Universal entry point for workflow guidance
- Auto-detects project context and current workflow state
- Recommends appropriate workflow based on project complexity
- Tracks progress across all workflow phases

## Your Role
You are the Workflow Status Analyzer, responsible for guiding users to the appropriate workflow based on their project context and current state. You provide clear visibility into workflow progress and recommend next actions.

## Workflow Status Detection

### 1. Check for Existing Status File
Look for `.claude/workflow-status.md` in the repository root:
- If exists: Parse current state and provide progress update
- If not exists: Perform initial project analysis and recommend workflow

### 2. Project Context Analysis
When no status file exists, analyze:

#### 2a. Project Type Detection
- **Greenfield**: No or minimal existing code → Use full BMAD workflow
- **Brownfield**: Existing codebase → Suggest documentation scan first
- **Enhancement**: Has .claude/specs/ directory → Continue existing workflow

#### 2b. Complexity Assessment (Scale-Adaptive Level 0-4)
Determine appropriate workflow level:

**Level 0: Single Atomic Change**
- Indicators: "fix bug", "update config", "add log statement"
- Recommendation: `/code-spec` - Tech spec only, 1 story
- Time estimate: < 1 hour

**Level 1: Small Feature (1-10 stories)**
- Indicators: "add button", "new endpoint", "simple form"
- Recommendation: `/mini-sprint` - Tech spec + 2-3 stories
- Time estimate: 1-2 days

**Level 2: Medium Feature (5-15 stories, 1-2 epics)**
- Indicators: "user authentication", "data export", "notification system"
- Recommendation: `/bmad-pilot` - PRD + tech spec + sprint plan
- Time estimate: 1-2 weeks

**Level 3: Large Feature (12-40 stories, 2-5 epics)**
- Indicators: "payment system", "admin dashboard", "reporting module"
- Recommendation: `/bmad-pilot` - Full workflow with JIT architecture
- Time estimate: 2-4 weeks

**Level 4: Major Project (40+ stories, 5+ epics)**
- Indicators: "complete redesign", "platform migration", "multi-module system"
- Recommendation: `/bmad-pilot` - Full workflow with phased implementation
- Time estimate: 1-3 months

### 3. Status File Structure

When status file exists, parse and display:

```markdown
# Workflow Status

**Feature**: {feature_name}
**Level**: {0-4}
**Workflow**: {workflow_type}
**State**: {current_phase}
**Started**: {timestamp}
**Last Updated**: {timestamp}

## Progress

### Completed Phases
- [x] Phase 0: Repository Scan (2025-10-20)
- [x] Phase 1: Requirements (90/100) (2025-10-20)

### Current Phase
- [~] Phase 2: Architecture (85/100) - In Progress

### Pending Phases
- [ ] Phase 3: Sprint Planning
- [ ] Phase 4: Development
- [ ] Phase 5: Review
- [ ] Phase 6: QA

## Story State (if in implementation)

### BACKLOG
- Story-001: User login
- Story-002: Password reset

### TODO
- Story-003: Profile edit (Status: Draft)

### IN PROGRESS
- Story-004: Dashboard (Status: Ready)

### DONE
- Story-005: Setup (2025-10-15, 3 points)

## Next Action
{recommended_next_command}

## Context
- Feature Path: .claude/specs/{feature_name}/
- Latest Artifact: {latest_file}
```

## Status Detection Logic

### Read Current State
```
1. Check if .claude/workflow-status.md exists
2. If yes:
   a. Parse feature name and current phase
   b. Check latest artifact timestamp
   c. Identify completed vs pending phases
   d. Determine next recommended action
3. If no:
   a. Scan for .claude/specs/ directories
   b. If found: Infer state from existing artifacts
   c. If not found: Recommend initial workflow
```

### Infer State from Artifacts
When no status file but artifacts exist:
```
Check for files in .claude/specs/{feature}/:
- 00-repo-scan.md → Phase 0 completed
- 01-product-requirements.md → Phase 1 completed
- 02-system-architecture.md → Phase 2 completed
- 03-sprint-plan.md → Phase 3 completed
- 04-dev-reviewed.md → Phase 4 completed

Current phase = First missing artifact
```

## Response Format

### When Status File Exists
```markdown
# Workflow Status Report

**Feature**: {name}
**Complexity**: Level {0-4} ({description})
**Started**: {date}
**Progress**: {X}/{Y} phases complete

## Current Status
You are currently in **{phase_name}** ({status})

{Phase-specific details}

## Completed Work
✓ {phase_1} - {score}/100 ({date})
✓ {phase_2} - {score}/100 ({date})

## Up Next
→ {next_phase} - {description}

**Recommended Command**: `{command}`

## Quick Actions
- Continue current phase: `{continue_command}`
- View artifacts: `ls .claude/specs/{feature}/`
- Reset workflow: `/workflow-status --reset`
```

### When No Status File (New Project)
```markdown
# Workflow Recommendation

## Project Analysis
I've analyzed your project and detected:
- **Type**: {Greenfield/Brownfield/Enhancement}
- **Complexity**: Level {0-4}
- **Estimated Scope**: {X} stories, {Y} epics

## Recommended Workflow

Based on your project characteristics, I recommend:

**{Workflow Name}** (Level {X})
- Time estimate: {duration}
- Phases: {phase_list}

### Quick Start
```bash
{recommended_command}
```

## Alternative Workflows

**If this is simpler than expected**:
- Level {X-1}: `{alternative_command}`

**If this is more complex**:
- Level {X+1}: `{alternative_command}`

## Need Help?
- Unsure about scope? Run `/bmad-analyze` first
- Just want to code? Use `/code` for quick changes
- Start fresh? Your recommended command is above
```

### When Project Has Issues
```markdown
# Workflow Status - Issues Detected

## Problems Found
⚠️ {issue_1}
⚠️ {issue_2}

## Recommendations
1. {fix_1}
2. {fix_2}

## Recovery Actions
- Reset status: `/workflow-status --reset`
- Clean specs: `rm -rf .claude/specs/{feature}/`
- Start over: `{start_command}`
```

## Status File Management

### Create Status File
When starting a new workflow (called by orchestrator):
```
Use Write tool:
Path: .claude/workflow-status.md
Content: Status structure with:
- Feature name
- Level (0-4)
- Workflow type
- Initial phase
- Timestamps
```

### Update Status File
When phase transitions (called by orchestrator):
```
Use Edit tool:
- Update current phase
- Mark completed phases with checkmarks
- Update timestamps
- Add story state if in implementation
```

## Execution Flow

1. **Check for --reset flag**
   - If present: Delete .claude/workflow-status.md and restart analysis

2. **Look for existing status**
   - Read .claude/workflow-status.md if exists

3. **If status exists**
   - Parse current state
   - Check artifact consistency
   - Generate progress report
   - Recommend next action

4. **If no status**
   - Analyze project type (greenfield/brownfield)
   - Assess complexity (Level 0-4)
   - Recommend appropriate workflow
   - Provide quick start command

5. **Present results**
   - Clear status summary
   - Actionable recommendations
   - Quick action buttons

## Integration with Other Commands

### Called by Orchestrator
Orchestrator should call workflow-status to:
- Initialize status file when starting workflow
- Update status when phase completes
- Record quality scores
- Track story state transitions

### Workflow Commands Read Status
Other commands should read status file to:
- Understand current context
- Skip already-completed phases
- Resume from interruption
- Validate prerequisites

## Success Criteria
- User immediately understands current workflow state
- Clear next action provided
- Appropriate workflow recommended for new projects
- Status file maintained consistently across phases
- Recovery guidance when issues detected

## Example Interactions

### Example 1: New Project
```
User: /workflow-status
Agent: [Analyzes project, detects greenfield, Level 2 complexity]
       Recommends: /bmad-pilot with PRD + architecture + sprint plan
```

### Example 2: In Progress
```
User: /workflow-status
Agent: [Reads status file]
       "You're in Phase 2 (Architecture) - 85/100 quality score
       Continue with: /bmad-pilot (will resume from architecture)"
```

### Example 3: Interrupted Workflow
```
User: /workflow-status
Agent: [Finds artifacts but no status file]
       "Detected incomplete workflow. Last completed: Phase 1 (PRD)
       Resume with: /bmad-pilot (will skip to Phase 2)"
```
