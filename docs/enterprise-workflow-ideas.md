# 企业级 Claude Code 工作流方案

基于 Anthropic 官方最佳实践、GitHub Copilot 企业级功能、以及 showcase 项目的研究整理。

## 实施状态
- ✅ codeagent-wrapper multi-backend
- ✅ /gh-create-issue command
- ✅ /gh-implement command
- ✅ Hooks + Skills activation

## 核心工作流矩阵

| 工作流 | 触发方式 | 核心能力 | 企业应用场景 |
|--------|----------|----------|--------------|
| `/gh-create-issue` | Command | 多轮对话 → 结构化 Issue | 需求澄清、Bug 报告标准化 |
| `/gh-implement` | Command | Issue → 开发 → PR | 自动化开发闭环 |
| `/code-review` | Hook (PR) | AI 审查 + 人工确认 | 代码质量把控 |
| `/incident-debug` | Command | 日志分析 → 根因定位 | 生产问题排查 |
| `/migration` | Command | 批量代码迁移 | 技术债务清理 |
| `/security-audit` | Hook/Scheduled | 安全扫描 + 修复建议 | 安全合规 |
| `/onboarding` | Command | 代码库问答 | 新人培训 |

---

## 1. GitHub Issue 全生命周期工作流

### 1.1 `/gh-create-issue` - 需求创建

```
用户输入 → 多轮澄清 → 结构化 Issue → gh issue create
```

**流程设计：**
```markdown
---
description: Create structured GitHub issue through multi-round dialogue
argument-hint: Brief description of what you need (e.g., "user authentication feature")
---

You are a Requirements Analyst. Help create a well-structured GitHub issue.

## Phase 1: Initial Understanding
Ask 2-3 targeted questions to understand:
- What problem does this solve? (Why)
- Who benefits from this? (Who)
- What's the expected outcome? (What)

## Phase 2: Technical Scoping
Based on answers, clarify:
- Acceptance criteria (testable conditions)
- Technical constraints
- Dependencies on other features/teams
- Priority and urgency

## Phase 3: Issue Generation
Generate issue with structure:
- **Title**: [Type] Brief description
- **Problem Statement**: Why this matters
- **Proposed Solution**: High-level approach
- **Acceptance Criteria**: Checkbox list
- **Technical Notes**: Implementation hints
- **Labels**: auto-suggest based on content

## Phase 4: Confirmation & Creation
Show preview → User confirms → `gh issue create`
```

### 1.2 `/gh-implement` - Issue 实现

```
gh issue view → 理解 + 沟通 → /dev 开发 → gh issue comment → gh pr create
```

**流程设计：**
```markdown
---
description: Implement GitHub issue with full development lifecycle
argument-hint: Issue number (e.g., "123")
---

## Phase 1: Issue Analysis
1. `gh issue view $ARGUMENTS --json title,body,labels,comments`
2. Parse requirements and acceptance criteria
3. Identify affected files via codebase exploration

## Phase 2: Clarification (if needed)
If ambiguous, use AskUserQuestion to clarify:
- Implementation approach choices
- Scope boundaries
- Testing requirements

## Phase 3: Development
Invoke /dev workflow with parsed requirements:
- Codex analysis
- Task breakdown
- Parallel execution
- Coverage validation (≥90%)

## Phase 4: Progress Updates
After each milestone:
`gh issue comment $ARGUMENTS --body "✅ Completed: [milestone]"`

## Phase 5: PR Creation
`gh pr create --title "[#$ARGUMENTS] ..." --body "Closes #$ARGUMENTS\n\n..."`
```

---

## 2. 代码审查工作流

### 2.1 PR 自动审查 Hook

**触发点：** PR 创建或更新时

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Bash(gh pr create:*)",
        "hooks": [{
          "type": "command",
          "command": "$CLAUDE_PROJECT_DIR/.claude/hooks/auto-review.sh"
        }]
      }
    ]
  }
}
```

**审查维度（参考 Anthropic 博客）：**
- 代码风格一致性
- 潜在 bug 和边界条件
- 安全漏洞（OWASP Top 10）
- 性能影响
- 文档完整性
- 测试覆盖率

### 2.2 `/review-pr` Command

```markdown
---
description: Comprehensive PR review with actionable feedback
argument-hint: PR number or URL
---

1. Fetch PR details: `gh pr view $ARGUMENTS --json files,commits,body`
2. Read changed files with context (±50 lines)
3. Analyze against:
   - Repository coding standards (CLAUDE.md)
   - Security best practices
   - Performance implications
   - Test coverage
4. Generate review with:
   - Summary of changes
   - 🟢 Approved / 🟡 Changes Requested / 🔴 Blocked
   - Specific line comments
   - Suggested improvements
5. Post review: `gh pr review $ARGUMENTS --body "..." [--approve|--request-changes]`
```

---

## 3. 生产问题排查工作流

### 3.1 `/incident-debug`

```markdown
---
description: Debug production incidents from logs and traces
argument-hint: Error message, log file path, or incident ID
---

## Phase 1: Context Gathering
- Parse provided logs/error messages
- Search codebase for related code paths
- Check recent deployments: `gh release list --limit 5`

## Phase 2: Root Cause Analysis
Use Codex for deep analysis:
- Stack trace interpretation
- Data flow tracing
- Dependency chain analysis

## Phase 3: Solution Proposal
- Immediate mitigation steps
- Long-term fix plan
- Regression test suggestions

## Phase 4: Documentation
Generate incident report:
- Timeline
- Root cause
- Impact assessment
- Resolution steps
- Prevention measures
```

---

## 4. 大规模迁移工作流

### 4.1 `/migration` - 批量代码迁移

**适用场景：**
- 框架升级（React 17→18, Vue 2→3）
- API 版本迁移
- 依赖库替换
- 代码模式重构

```markdown
---
description: Batch code migration with validation
argument-hint: Migration type and scope (e.g., "React class to hooks in src/components")
---

## Phase 1: Scope Analysis
1. Use Codex to identify all affected files
2. Generate migration task list (file by file)
3. Estimate complexity per file

## Phase 2: Parallel Execution (Headless Mode)
For each file, run:
```bash
claude -p "Migrate $FILE from [old] to [new]. Verify with tests." \
  --allowedTools Edit Bash(npm test:*)
```

## Phase 3: Validation
- Run full test suite
- Type checking
- Lint verification

## Phase 4: Report
- Success/failure per file
- Manual review required files
- Rollback instructions
```

---

## 5. 安全审计工作流

### 5.1 `/security-audit`

```markdown
---
description: Security vulnerability scanning and remediation
---

## Scan Categories
1. **Dependency vulnerabilities**: `npm audit` / `pip-audit`
2. **SAST**: Code pattern analysis for OWASP Top 10
3. **Secrets detection**: Hardcoded credentials
4. **Configuration**: Insecure defaults

## Output Format
- Severity: Critical/High/Medium/Low
- Location: File:Line
- Description: What's wrong
- Remediation: How to fix
- Auto-fix available: Yes/No

## Auto-remediation
For auto-fixable issues:
1. Generate fix via Codex
2. Run tests
3. Create PR with security label
```

---

## 6. 新人培训工作流

### 6.1 Codebase Q&A（Anthropic 推荐）

直接使用 Claude Code 进行代码库问答，无需特殊配置：

**常见问题类型：**
- "这个项目的架构是什么？"
- "如何添加新的 API 端点？"
- "日志系统是怎么工作的？"
- "这个函数为什么这样设计？"（结合 git history）

### 6.2 `/onboarding` Command

```markdown
---
description: Interactive codebase onboarding for new team members
---

## Phase 1: Overview
- Read README, CLAUDE.md, package.json
- Summarize tech stack and architecture

## Phase 2: Key Flows
For each major feature:
- Entry point
- Data flow
- Key files

## Phase 3: Development Setup
- Environment setup steps
- Common commands
- Testing workflow

## Phase 4: Q&A Mode
"Ask me anything about this codebase!"
```

---

## 7. codeagent-wrapper 多后端架构

### 设计方案

```go
// codeagent-wrapper architecture
type AgentBackend interface {
    Name() string
    Execute(ctx context.Context, task TaskSpec, timeout int) TaskResult
    HealthCheck() error
}

type CodexBackend struct{}    // OpenAI Codex
type ClaudeBackend struct{}   // Claude CLI (claude -p)
type GeminiBackend struct{}   // Gemini API

// 命令行接口
// codeagent-wrapper [--backend=codex|claude|gemini] "task" [workdir]
// codeagent-wrapper --parallel --backend=claude < tasks.txt
```

### 后端选择策略

| 任务类型 | 推荐后端 | 原因 |
|----------|----------|------|
| 代码生成/重构 | Codex | 代码专精 |
| 复杂推理/规划 | Claude | 推理能力强 |
| 快速原型 | Gemini | 速度快、成本低 |
| 并行批量任务 | 混合 | 负载均衡 |

---

## 8. Hooks + Skills 协作模式

### 推荐配置

```json
{
  "hooks": {
    "UserPromptSubmit": [
      {
        "hooks": [{
          "type": "command",
          "command": "$CLAUDE_PROJECT_DIR/.claude/hooks/skill-activation-prompt.sh"
        }]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "Edit|MultiEdit|Write",
        "hooks": [{
          "type": "command",
          "command": "$CLAUDE_PROJECT_DIR/.claude/hooks/post-tool-tracker.sh"
        }]
      },
      {
        "matcher": "Bash(gh pr create:*)",
        "hooks": [{
          "type": "command",
          "command": "$CLAUDE_PROJECT_DIR/.claude/hooks/auto-review-trigger.sh"
        }]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {"type": "command", "command": "$CLAUDE_PROJECT_DIR/.claude/hooks/test-runner.sh"},
          {"type": "command", "command": "$CLAUDE_PROJECT_DIR/.claude/hooks/coverage-check.sh"}
        ]
      }
    ]
  }
}
```

### skill-rules.json 扩展

```json
{
  "skills": {
    "gh-workflow": {
      "type": "domain",
      "enforcement": "suggest",
      "priority": "high",
      "promptTriggers": {
        "keywords": ["issue", "pr", "pull request", "github", "gh"],
        "intentPatterns": ["(create|implement|review).*?(issue|pr|pull)"]
      }
    },
    "incident-response": {
      "type": "domain",
      "enforcement": "suggest",
      "priority": "critical",
      "promptTriggers": {
        "keywords": ["error", "bug", "incident", "production", "debug", "crash"],
        "intentPatterns": ["(fix|debug|investigate).*?(error|bug|issue)"]
      }
    }
  }
}
```

---

## 9. 实施优先级建议

### Phase 1: 基础设施（1-2 周）
1. ✅ codeagent-wrapper 已完成
2. 🔄 codeagent-wrapper 多后端改造
3. 🆕 基础 hooks 配置

### Phase 2: 核心工作流（2-3 周）
1. `/gh-create-issue` command
2. `/gh-implement` command
3. `/code-review` command

### Phase 3: 高级功能（3-4 周）
1. skill-rules.json + activation hook
2. `/migration` 批量迁移
3. `/security-audit` 安全审计

### Phase 4: 企业级增强
1. 多 Claude 实例协作
2. CI/CD 集成（headless mode）
3. 监控和分析仪表板

---

## 参考资料

- [Anthropic Claude Code Best Practices](https://www.anthropic.com/engineering/claude-code-best-practices)
- [GitHub Copilot Coding Agent](https://docs.github.com/en/copilot/using-github-copilot/using-copilot-coding-agent-to-work-on-tasks)
- [claude-code-infrastructure-showcase](https://github.com/hellogithub/claude-code-infrastructure-showcase)
