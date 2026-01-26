# Development Commands Reference

> Direct slash commands for daily coding tasks without workflow overhead

## 🎯 Overview

Development Essentials provides focused slash commands for common development tasks. Use these when you need direct implementation without the full workflow structure.

## 📋 Available Commands

### `/code` - Direct Implementation

Implement features, add functionality, or write code directly.

**Usage**:
```bash
/code "Add input validation for email fields"
/code "Implement pagination for user list API"
/code "Create database migration for orders table"
```

**Agent**: `code`

**Best for**:
- Clear, well-defined tasks
- Quick implementations
- Following existing patterns
- Adding straightforward features

### `/debug` - Systematic Debugging

Analyze and fix bugs with structured debugging approach.

**Usage**:
```bash
/debug "Login fails with 500 error on invalid credentials"
/debug "Memory leak in background worker process"
/debug "Race condition in order processing"
```

**Agent**: `debug`

**Approach**:
1. Reproduce the issue
2. Analyze root cause
3. Propose solution
4. Implement fix
5. Verify resolution

### `/test` - Testing Strategy

Create tests, improve test coverage, or test existing code.

**Usage**:
```bash
/test "Add unit tests for authentication service"
/test "Create integration tests for payment flow"
/test "Test edge cases for date parser"
```

**Agent**: `develop` (testing mode)

**Covers**:
- Unit tests
- Integration tests
- Edge cases
- Error scenarios
- Test data setup

### `/optimize` - Performance Tuning

Improve performance, reduce resource usage, or optimize algorithms.

**Usage**:
```bash
/optimize "Reduce database queries in dashboard endpoint"
/optimize "Speed up report generation process"
/optimize "Improve memory usage in data processing pipeline"
```

**Agent**: `develop` (optimization mode)

**Focus areas**:
- Algorithm efficiency
- Database query optimization
- Caching strategies
- Resource utilization
- Load time reduction

### `/bugfix` - Bug Resolution

Fix specific bugs with focused approach.

**Usage**:
```bash
/bugfix "Users can't reset password with special characters"
/bugfix "Session expires too quickly on mobile"
/bugfix "File upload fails for large files"
```

**Agent**: `bugfix`

**Process**:
1. Understand the bug
2. Locate problematic code
3. Implement fix
4. Add regression tests
5. Verify fix

### `/refactor` - Code Improvement

Improve code structure, readability, or maintainability without changing behavior.

**Usage**:
```bash
/refactor "Extract user validation logic into separate module"
/refactor "Simplify nested conditionals in order processing"
/refactor "Remove code duplication in API handlers"
```

**Agent**: `develop` (refactor mode)

**Goals**:
- Improve readability
- Reduce complexity
- Eliminate duplication
- Enhance maintainability
- Follow best practices

### `/review` - Code Validation

Review code for quality, security, and best practices.

**Usage**:
```bash
/review "Check authentication implementation for security issues"
/review "Validate API error handling patterns"
/review "Assess database schema design"
```

**Agent**: Independent reviewer

**Review criteria**:
- Code quality
- Security vulnerabilities
- Performance issues
- Best practices compliance
- Maintainability

### `/ask` - Technical Consultation

Get technical advice, design patterns, or implementation guidance.

**Usage**:
```bash
/ask "Best approach for real-time notifications in React"
/ask "How to handle database migrations in production"
/ask "Design pattern for plugin system"
```

**Agent**: Technical consultant

**Provides**:
- Architecture guidance
- Technology recommendations
- Design patterns
- Best practices
- Trade-off analysis

### `/docs` - Documentation

Generate or improve documentation.

**Usage**:
```bash
/docs "Create API documentation for user endpoints"
/docs "Add JSDoc comments to utility functions"
/docs "Write README for authentication module"
```

**Agent**: Documentation writer

**Creates**:
- Code comments
- API documentation
- README files
- Usage examples
- Architecture docs

### `/think` - Advanced Analysis

Deep reasoning and analysis for complex problems.

**Usage**:
```bash
/think "Analyze scalability bottlenecks in current architecture"
/think "Evaluate different approaches for data synchronization"
/think "Design migration strategy from monolith to microservices"
```

**Agent**: `gpt5` (deep reasoning)

**Best for**:
- Complex architectural decisions
- Multi-faceted problems
- Trade-off analysis
- Strategic planning
- System design

## 🔄 Command Workflows

### Simple Feature Development

```bash
# 1. Ask for guidance
/ask "Best way to implement rate limiting in Express"

# 2. Implement the feature
/code "Add rate limiting middleware to API routes"

# 3. Add tests
/test "Create tests for rate limiting behavior"

# 4. Review implementation
/review "Validate rate limiting implementation"
```

### Bug Investigation and Fix

```bash
# 1. Debug the issue
/debug "API returns 500 on concurrent requests"

# 2. Fix the bug
/bugfix "Add mutex lock to prevent race condition"

# 3. Add regression tests
/test "Test concurrent request handling"
```

### Code Quality Improvement

```bash
# 1. Review current code
/review "Analyze user service for improvements"

# 2. Refactor based on findings
/refactor "Simplify user validation logic"

# 3. Optimize performance
/optimize "Cache frequently accessed user data"

# 4. Update documentation
/docs "Document user service API"
```

## 🎯 When to Use What

### Use Direct Commands When:
- Task is clear and well-defined
- No complex planning needed
- Fast iteration is priority
- Working within existing patterns

### Use Requirements Workflow When:
- Feature has unclear requirements
- Need documented specifications
- Multiple implementation approaches possible
- Quality gates desired

### Use BMAD Workflow When:
- Complex business requirements
- Architecture design needed
- Sprint planning required
- Multiple stakeholders involved

## 💡 Best Practices

1. **Be Specific**: Provide clear, detailed descriptions
   - ❌ `/code "fix the bug"`
   - ✅ `/code "Fix null pointer exception in user login when email is missing"`

2. **One Task Per Command**: Keep commands focused
   - ❌ `/code "Add feature X, fix bug Y, refactor module Z"`
   - ✅ `/code "Add email validation to registration form"`

3. **Provide Context**: Include relevant details
   - ✅ `/debug "Login API returns 401 after password change, only on Safari"`

4. **Use Appropriate Command**: Match command to task type
   - Use `/bugfix` for bugs, not `/code`
   - Use `/refactor` for restructuring, not `/optimize`
   - Use `/think` for complex analysis, not `/ask`

5. **Chain Commands**: Break complex tasks into steps
   ```bash
   /ask "How to implement OAuth2"
   /code "Implement OAuth2 authorization flow"
   /test "Add OAuth2 integration tests"
   /review "Validate OAuth2 security"
   /docs "Document OAuth2 setup process"
   ```

## 🔌 Agent Configuration

All commands use specialized agents configured in:
- `agents/development-essentials/agents/`
- Agent prompt templates
- Tool access permissions
- Output formatting

## 📚 Related Documentation

- **[BMAD Workflow](BMAD-WORKFLOW.md)** - Full agile methodology
- **[Requirements Workflow](REQUIREMENTS-WORKFLOW.md)** - Lightweight workflow
- **[Quick Start Guide](QUICK-START.md)** - Get started quickly
- **[Plugin System](PLUGIN-SYSTEM.md)** - Installation and configuration

---

**Development Essentials** - Direct commands for productive coding without workflow overhead.
