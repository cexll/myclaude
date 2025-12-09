# GitHub Workflow Commands

Issue-to-PR automation with `codeagent-wrapper` integration.

## Overview

Two commands streamline GitHub workflows:
- `/gh-create-issue` - Create structured issues through guided dialogue
- `/gh-implement` - Implement issues with automated development lifecycle

## Prerequisites

```bash
# Install GitHub CLI
brew install gh

# Authenticate
gh auth login

# Verify
gh auth status
```

## Commands

### /gh-create-issue

Create structured GitHub issues through multi-round dialogue.

**Usage:**
```bash
/gh-create-issue "add user authentication"
```

**Process:**

1. **Initial Analysis** - Claude analyzes your brief description
2. **Clarification Questions** - Interactive Q&A to gather:
   - Technical requirements
   - Acceptance criteria
   - Dependencies
   - Priority/labels
3. **Issue Creation** - Generates structured issue with:
   - Clear title
   - Detailed description
   - Acceptance criteria checklist
   - Relevant labels
   - Assignee (optional)

**Example Session:**

```
User: /gh-create-issue "add dark mode"

Claude: I'll help create a structured issue for dark mode. Let me gather some details.

1. Scope: Just theme toggle, or also persist user preference?
2. Which components need dark mode support?
3. Design system: Using CSS variables or styled-components?

User:
1. Toggle + persist in localStorage
2. All components in src/components
3. CSS variables

Claude: Creating issue...

✅ Issue created: #123 "Implement dark mode with theme persistence"
URL: https://github.com/user/repo/issues/123
```

**Generated Issue Format:**

```markdown
# Implement dark mode with theme persistence

## Description
Add dark mode support across all components with user preference persistence.

## Requirements
- [ ] Add theme toggle component
- [ ] Implement CSS variables for light/dark themes
- [ ] Persist theme preference in localStorage
- [ ] Update all components in src/components to support dark mode
- [ ] Add theme toggle to app header

## Acceptance Criteria
- [ ] User can toggle between light and dark themes
- [ ] Theme preference persists across sessions
- [ ] All UI components render correctly in both themes
- [ ] No flash of unstyled content on page load

## Technical Notes
- Use CSS custom properties
- Store preference as `theme: 'light' | 'dark'` in localStorage
- Add `data-theme` attribute to root element

Labels: enhancement, ui
```

---

### /gh-implement

Implement GitHub issue with full development lifecycle.

**Usage:**
```bash
/gh-implement 123
```

**Phases:**

#### Phase 1: Issue Analysis
```bash
# Fetches issue details
gh issue view 123 --json title,body,labels,comments

# Parses:
- Requirements
- Acceptance criteria
- Technical constraints
- Related discussions
```

#### Phase 2: Clarification (if needed)
Claude asks questions about:
- Implementation approach
- Architecture decisions
- Testing strategy
- Edge cases

#### Phase 3: Development

**Option A: Simple scope** - Direct `codeagent-wrapper` call:
```bash
codeagent-wrapper --backend codex - <<'EOF'
Implement dark mode toggle based on issue #123:
- Add ThemeToggle component
- Implement CSS variables
- Add localStorage persistence
EOF
```

**Option B: Complex scope** - Use `/dev` workflow:
```bash
/dev "implement issue #123: dark mode with theme persistence"
```

**Coverage requirement:** ≥90% test coverage enforced

#### Phase 4: Progress Updates
```bash
# After each milestone
gh issue comment 123 --body "✅ Completed: ThemeToggle component"
gh issue comment 123 --body "✅ Completed: CSS variables setup"
gh issue comment 123 --body "✅ Completed: localStorage persistence"
```

#### Phase 5: PR Creation
```bash
gh pr create \
  --title "[#123] Implement dark mode with theme persistence" \
  --body "Closes #123

## Changes
- Added ThemeToggle component
- Implemented light/dark CSS variables
- Added localStorage persistence
- Updated all components for theme support

## Testing
- Unit tests: ThemeToggle, theme utilities
- Integration tests: theme persistence across page loads
- Coverage: 92%"
```

**Output:**
```
✅ PR created: #124
URL: https://github.com/user/repo/pull/124
```

---

## Examples

### Example 1: Bug Fix

```bash
# Create issue
/gh-create-issue "login form doesn't validate email"

# Implement
/gh-implement 125
```

**Process:**
1. Analysis: Parse bug report, identify validation logic
2. Clarification: Confirm expected validation rules
3. Development: Fix validation, add tests
4. Updates: Comment with fix details
5. PR: Link to issue, show test coverage

---

### Example 2: Feature Development

```bash
# Create issue
/gh-create-issue "add export to CSV feature"

# Implement
/gh-implement 126
```

**Process:**
1. Analysis: Understand data structure, export requirements
2. Clarification: Which data fields? File naming? Encoding?
3. Development:
   - Backend: CSV generation endpoint
   - Frontend: Export button + download handler
   - Tests: Unit + integration
4. Updates: Milestone comments (backend done, frontend done, tests done)
5. PR: Full feature description with screenshots

---

### Example 3: Refactoring

```bash
# Create issue
/gh-create-issue "refactor authentication module"

# Implement
/gh-implement 127
```

**Process:**
1. Analysis: Review current auth code, identify issues
2. Clarification: Scope (just refactor vs add features)?
3. Development:
   - Modularize auth logic
   - Extract reusable utilities
   - Add missing tests
   - Update documentation
4. Updates: Component-by-component progress
5. PR: Before/after comparison, test coverage improvement

---

## Workflow Integration

### With /dev Workflow

```bash
# Create issue first
/gh-create-issue "implement real-time notifications"

# Then implement with /dev
/gh-implement 128

# Claude will:
# 1. Analyze issue #128
# 2. Trigger /dev workflow internally
# 3. Execute with 90% coverage requirement
# 4. Post progress updates
# 5. Create PR
```

### With Parallel Tasks

For complex features, `/gh-implement` may use parallel execution:

```bash
# Internally executes:
codeagent-wrapper --parallel <<'EOF'
---TASK---
id: backend_notifications
workdir: /project/backend
---CONTENT---
implement notifications API with WebSocket

---TASK---
id: frontend_notifications
workdir: /project/frontend
dependencies: backend_notifications
---CONTENT---
build Notifications UI component

---TASK---
id: tests_notifications
workdir: /project
dependencies: backend_notifications, frontend_notifications
---CONTENT---
add E2E tests for notification flow
EOF
```

---

## Configuration

### Issue Templates

Create `.github/ISSUE_TEMPLATE/feature.md`:

```markdown
---
name: Feature Request
about: Suggest a new feature
labels: enhancement
---

## Description
<!-- Clear description of the feature -->

## Requirements
<!-- Specific requirements -->

## Acceptance Criteria
<!-- Checklist of criteria -->
```

### PR Templates

Create `.github/PULL_REQUEST_TEMPLATE.md`:

```markdown
## Related Issue
Closes #

## Changes
<!-- List of changes -->

## Testing
<!-- Test coverage and manual testing -->

## Screenshots (if applicable)
<!-- Before/after screenshots -->
```

---

## Best Practices

1. **Clear issue descriptions** - More context = better implementation
2. **Incremental commits** - Easier to review and rollback
3. **Test-driven** - Write tests before/during implementation
4. **Milestone updates** - Keep issue comments up-to-date
5. **Detailed PRs** - Explain why, not just what

---

## Troubleshooting

**Issue not found:**
```bash
# Verify issue exists
gh issue view 123

# Check repository
gh repo view
```

**PR creation failed:**
```bash
# Ensure branch is pushed
git push -u origin feature-branch

# Check if PR already exists
gh pr list --head feature-branch
```

**Authentication error:**
```bash
# Re-authenticate
gh auth login

# Check token scopes
gh auth status
```

---

## Advanced Usage

### Custom Labels

```bash
# Add labels during issue creation
gh issue create \
  --title "Feature: dark mode" \
  --body "..." \
  --label "enhancement,ui,priority:high"
```

### Multiple Assignees

```bash
# Assign to team members
gh issue create \
  --title "..." \
  --assignee @user1,@user2
```

### Milestone Assignment

```bash
# Add to milestone
gh issue create \
  --title "..." \
  --milestone "v2.0"
```

---

## Integration with CI/CD

### Auto-close on merge

```yaml
# .github/workflows/pr-merge.yml
name: Close Issues on PR Merge
on:
  pull_request:
    types: [closed]

jobs:
  close-issues:
    if: github.event.pull_request.merged == true
    runs-on: ubuntu-latest
    steps:
      - name: Close linked issues
        run: gh issue close ${{ github.event.pull_request.number }}
```

### Coverage Check

```yaml
# .github/workflows/coverage.yml
name: Coverage Check
on: [pull_request]

jobs:
  coverage:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Run tests with coverage
        run: go test -coverprofile=coverage.out ./...
      - name: Check coverage threshold
        run: |
          coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          if (( $(echo "$coverage < 90" | bc -l) )); then
            echo "Coverage $coverage% is below 90% threshold"
            exit 1
          fi
```

---

## Further Reading

- [GitHub CLI Manual](https://cli.github.com/manual/)
- [Codeagent-Wrapper Guide](./CODEAGENT-WRAPPER.md)
- [Hooks Documentation](./HOOKS.md)
- [Development Workflow](../README.md)
