# Quick Start Guide

> Get started with Claude Code Multi-Agent Workflow System in 5 minutes

## 🚀 Installation (2 minutes)

### Option 1: Plugin System (Fastest)

```bash
# Install everything with one command
/plugin marketplace add cexll/myclaude
```

### Option 2: Make Install

```bash
git clone https://github.com/cexll/myclaude.git
cd myclaude
make install
```

### Option 3: Selective Install

```bash
# Install only what you need
/plugin install bmad-agile-workflow       # Full agile workflow
/plugin install development-essentials    # Daily coding commands
```

## 🎯 Your First Workflow (3 minutes)

### Try BMAD Workflow

Complete agile development automation:

```bash
/bmad-pilot "Build a simple todo list API with CRUD operations"
```

**What happens**:
1. **Product Owner** generates requirements (PRD)
2. **Architect** designs system architecture
3. **Scrum Master** creates sprint plan
4. **Developer** implements code
5. **Reviewer** performs code review
6. **QA** runs tests

All documents saved to `.claude/specs/todo-list-api/`

### Try Requirements Workflow

Fast prototyping:

```bash
/requirements-pilot "Add user authentication to existing API"
```

**What happens**:
1. Generate functional requirements
2. Implement code
3. Review implementation
4. Create tests

### Try Direct Commands

Quick coding without workflow:

```bash
# Implement a feature
/code "Add input validation for email fields"

# Debug an issue
/debug "API returns 500 on missing parameters"

# Add tests
/test "Create unit tests for validation logic"
```

## 📋 Common Use Cases

### 1. New Feature Development

**Complex Feature** (use BMAD):
```bash
/bmad-pilot "User authentication system with OAuth2, MFA, and role-based access control"
```

**Simple Feature** (use Requirements):
```bash
/requirements-pilot "Add pagination to user list endpoint"
```

**Tiny Feature** (use direct command):
```bash
/code "Add created_at timestamp to user model"
```

### 2. Bug Fixing

**Complex Bug** (use debug):
```bash
/debug "Memory leak in background job processor"
```

**Simple Bug** (use bugfix):
```bash
/bugfix "Login button not working on mobile Safari"
```

### 3. Code Quality

**Full Review**:
```bash
/review "Review authentication module for security issues"
```

**Refactoring**:
```bash
/refactor "Simplify user validation logic and remove duplication"
```

**Optimization**:
```bash
/optimize "Reduce database queries in dashboard API"
```

## 🎨 Workflow Selection Guide

```
┌─────────────────────────────────────────────────────────┐
│                    Choose Your Workflow                 │
└─────────────────────────────────────────────────────────┘

Complex Business Feature + Architecture Needed
              ↓
      🏢 Use BMAD Workflow
   /bmad-pilot "description"
   • 6 specialized agents
   • Quality gates (PRD ≥90, Design ≥90)
   • Complete documentation
   • Sprint planning included

────────────────────────────────────────────────────────

Clear Requirements + Fast Iteration Needed
              ↓
    ⚡ Use Requirements Workflow
 /requirements-pilot "description"
   • 4 phases: Requirements → Code → Review → Test
   • Quality gate (Requirements ≥90)
   • Minimal documentation
   • Direct to implementation

────────────────────────────────────────────────────────

Well-Defined Task + No Workflow Overhead
              ↓
      🔧 Use Direct Commands
  /code | /debug | /test | /optimize
   • Single-purpose commands
   • Immediate execution
   • No documentation overhead
   • Perfect for daily tasks
```

## 💡 Tips for Success

### 1. Be Specific

**❌ Bad**:
```bash
/bmad-pilot "Build an app"
```

**✅ Good**:
```bash
/bmad-pilot "Build a task management API with user authentication, task CRUD, 
task assignment, and real-time notifications via WebSocket"
```

### 2. Provide Context

Include relevant technical details:
```bash
/code "Add Redis caching to user profile endpoint, cache TTL 5 minutes, 
invalidate on profile update"
```

### 3. Engage with Agents

During BMAD workflow, provide feedback at quality gates:

```
PO: "Here's the PRD (Score: 85/100)"
You: "Add mobile app support and offline mode requirements"
PO: "Updated PRD (Score: 94/100) ✅"
```

### 4. Review Generated Artifacts

Check documents before confirming:
- `.claude/specs/{feature}/01-product-requirements.md`
- `.claude/specs/{feature}/02-system-architecture.md`
- `.claude/specs/{feature}/03-sprint-plan.md`

### 5. Chain Commands for Complex Tasks

Break down complex work:
```bash
/ask "Best approach for implementing real-time chat"
/bmad-pilot "Real-time chat system with message history and typing indicators"
/test "Add integration tests for chat message delivery"
/docs "Document chat API endpoints and WebSocket events"
```

## 🎓 Learning Path

**Day 1**: Try direct commands
```bash
/code "simple task"
/test "add some tests"
/review "check my code"
```

**Day 2**: Try Requirements workflow
```bash
/requirements-pilot "small feature"
```

**Week 2**: Try BMAD workflow
```bash
/bmad-pilot "larger feature"
```

**Week 3**: Combine workflows
```bash
# Use BMAD for planning
/bmad-pilot "new module" --direct-dev

# Use Requirements for sprint tasks
/requirements-pilot "individual task from sprint"

# Use commands for daily work
/code "quick fix"
/test "add test"
```

## 📚 Next Steps

### Explore Documentation

- **[BMAD Workflow Guide](BMAD-WORKFLOW.md)** - Deep dive into full agile workflow
- **[Requirements Workflow Guide](REQUIREMENTS-WORKFLOW.md)** - Learn lightweight development
- **[Development Commands Reference](DEVELOPMENT-COMMANDS.md)** - All command details
- **[Plugin System Guide](PLUGIN-SYSTEM.md)** - Plugin management

### Try Advanced Features

**BMAD Options**:
```bash
# Skip testing for prototype
/bmad-pilot "prototype" --skip-tests

# Skip sprint planning for quick dev
/bmad-pilot "feature" --direct-dev

# Skip repo scan (if context exists)
/bmad-pilot "feature" --skip-scan
```

**Individual Agents**:
```bash
# Just requirements
/bmad-po "feature requirements"

# Just architecture
/bmad-architect "system design"

# Just orchestration
/bmad-orchestrator "complex project coordination"
```

### Check Quality

Run tests and validation:
```bash
make test-bmad              # Test BMAD workflow
make test-requirements      # Test Requirements workflow
```

## 🆘 Troubleshooting

**Commands not found**?
```bash
# Verify installation
/plugin list

# Reinstall if needed
make install
```

**Agents not working**?
```bash
# Check agent configuration
ls ~/.config/claude/agents/

# Redeploy agents
make deploy-agents
```

**Output styles missing**?
```bash
# Deploy output styles
cp output-styles/*.md ~/.config/claude/output-styles/
```

## 📞 Get Help

- **Issues**: [GitHub Issues](https://github.com/cexll/myclaude/issues)
- **Documentation**: [docs/](.)
- **Examples**: Check `.claude/specs/` after running workflows
- **Make Help**: Run `make help` for all commands

---

**You're ready!** Start with `/code "your first task"` and explore from there.
