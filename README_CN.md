# Claude Code 多智能体工作流系统

[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL_v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![Claude Code](https://img.shields.io/badge/Claude-Code-blue)](https://claude.ai/code)
[![Version](https://img.shields.io/badge/Version-6.x-green)](https://github.com/cexll/myclaude)

> AI 驱动的开发自动化 - 多后端执行架构 (Codex/Claude/Gemini/OpenCode)

## 快速开始

```bash
npx github:cexll/myclaude
```

## 模块概览

| 模块 | 描述 | 文档 |
|------|------|------|
| [do](skills/do/README.md) | **推荐** - 5 阶段功能开发 + codeagent 编排 | `/do` 命令 |
| [omo](skills/omo/README.md) | 多智能体编排 + 智能路由 | `/omo` 命令 |
| [bmad](agents/bmad/README.md) | BMAD 敏捷工作流 + 6 个专业智能体 | `/bmad-pilot` 命令 |
| [requirements](agents/requirements/README.md) | 轻量级需求到代码流水线 | `/requirements-pilot` 命令 |
| [essentials](agents/development-essentials/README.md) | 11 个核心开发命令：ask、bugfix、code、debug、docs、enhance-prompt、optimize、refactor、review、test、think | `/code`, `/debug` 等 |
| [sparv](skills/sparv/README.md) | SPARV 工作流 (Specify→Plan→Act→Review→Vault) | `/sparv` 命令 |
| course | 课程开发（组合 dev + product-requirements + test-cases） | 组合模块 |
| claudekit | ClaudeKit：do 技能 + 全局钩子（pre-bash、inject-spec、log-prompt）| 组合模块 |

### 可用技能

可通过 `npx github:cexll/myclaude --list` 单独安装技能（模块内置技能如 do、omo、sparv 见上表）：

| 技能 | 描述 |
|------|------|
| browser | 浏览器自动化测试和数据提取 |
| codeagent | codeagent-wrapper 多后端 AI 代码任务调用 |
| codex | Codex 后端直接执行 |
| dev | 轻量级端到端开发工作流 |
| gemini | Gemini 后端直接执行 |
| product-requirements | 交互式 PRD 生成（含质量评分）|
| prototype-prompt-generator | 结构化 UI/UX 原型提示词生成 |
| skill-install | 从 GitHub 安装技能（含安全扫描）|
| test-cases | 从需求生成全面测试用例 |

## 核心架构

| 角色 | 智能体 | 职责 |
|------|-------|------|
| **编排者** | Claude Code | 规划、上下文收集、验证 |
| **执行者** | codeagent-wrapper | 代码编辑、测试执行（Codex/Claude/Gemini/OpenCode 后端）|

## 工作流详解

### do 工作流（推荐）

5 阶段功能开发，通过 codeagent-wrapper 编排多个智能体。**大多数功能开发任务的首选工作流。**

```bash
/do "添加用户登录功能"
```

**5 阶段：**
| 阶段 | 名称 | 目标 |
|------|------|------|
| 1 | Understand | 并行探索理解需求和映射代码库 |
| 2 | Clarify | 解决阻塞性歧义（条件触发）|
| 3 | Design | 产出最小变更实现方案 |
| 4 | Implement + Review | 构建功能并审查 |
| 5 | Complete | 记录构建结果 |

**智能体：**
- `code-explorer` - 代码追踪、架构映射
- `code-architect` - 设计方案、文件规划
- `code-reviewer` - 代码审查、简化建议
- `develop` - 实现代码、运行测试

---

### OmO 多智能体编排器

基于风险信号智能路由任务到专业智能体。

```bash
/omo "分析并修复这个认证 bug"
```

**智能体层级：**
| 智能体 | 角色 | 后端 |
|-------|------|------|
| `oracle` | 技术顾问 | Claude |
| `librarian` | 外部研究 | Claude |
| `explore` | 代码库搜索 | OpenCode |
| `develop` | 代码实现 | Codex |
| `frontend-ui-ux-engineer` | UI/UX 专家 | Gemini |
| `document-writer` | 文档撰写 | Gemini |

**常用配方：**
- 解释代码：`explore`
- 位置已知的小修复：直接 `develop`
- Bug 修复（位置未知）：`explore → develop`
- 跨模块重构：`explore → oracle → develop`

---

### SPARV 工作流

极简 5 阶段工作流：Specify → Plan → Act → Review → Vault。

```bash
/sparv "实现订单导出功能"
```

**核心规则：**
- **10 分规格门**：得分 0-10，必须 >=9 才能进入 Plan
- **2 动作保存**：每 2 次工具调用写入 journal.md
- **3 失败协议**：连续 3 次失败后停止并上报
- **EHRB**：高风险操作需明确确认

**评分维度（各 0-2 分）：**
1. Value - 为什么做，可验证的收益
2. Scope - MVP + 不在范围内的内容
3. Acceptance - 可测试的验收标准
4. Boundaries - 错误/性能/兼容/安全边界
5. Risk - EHRB/依赖/未知 + 处理方式

---

### BMAD 敏捷工作流

完整企业敏捷方法论 + 6 个专业智能体。

```bash
/bmad-pilot "构建电商结账系统"
```

**智能体角色：**
| 智能体 | 职责 |
|-------|------|
| Product Owner | 需求与用户故事 |
| Architect | 系统设计与技术决策 |
| Scrum Master | Sprint 规划与任务分解 |
| Developer | 实现 |
| Code Reviewer | 质量保证 |
| QA Engineer | 测试与验证 |

**审批门：**
- PRD 完成后（90+ 分）需用户审批
- 架构完成后（90+ 分）需用户审批

---

### 需求驱动工作流

轻量级需求到代码流水线。

```bash
/requirements-pilot "实现 API 限流"
```

**100 分质量评分：**
- 功能清晰度：30 分
- 技术具体性：25 分
- 实现完整性：25 分
- 业务上下文：20 分

---

### 开发基础命令

日常编码任务的直接命令。

| 命令 | 用途 |
|------|------|
| `/code` | 实现功能 |
| `/debug` | 调试问题 |
| `/test` | 编写测试 |
| `/review` | 代码审查 |
| `/optimize` | 性能优化 |
| `/refactor` | 代码重构 |
| `/docs` | 编写文档 |
| `/ask` | 提问和咨询 |
| `/bugfix` | Bug 修复 |
| `/enhance-prompt` | 提示词优化 |
| `/think` | 深度思考分析 |

---

## 安装

```bash
# 交互式安装器（推荐）
npx github:cexll/myclaude

# 列出可安装项（module:* / skill:* / codeagent-wrapper）
npx github:cexll/myclaude --list

# 检测已安装 modules 并从 GitHub 更新
npx github:cexll/myclaude --update

# 指定安装目录 / 强制覆盖
npx github:cexll/myclaude --install-dir ~/.claude --force
```

`--update` 会在目标安装目录（默认 `~/.claude`，优先读取 `installed_modules.json`）检测已安装 modules，并从 GitHub 拉取最新发布版本覆盖更新。

### 模块配置

编辑 `config.json` 启用/禁用模块：

```json
{
  "modules": {
    "bmad": { "enabled": false },
    "requirements": { "enabled": false },
    "essentials": { "enabled": false },
    "omo": { "enabled": false },
    "sparv": { "enabled": false },
    "do": { "enabled": true },
    "course": { "enabled": false }
  }
}
```

## 工作流选择指南

| 场景 | 推荐 |
|------|------|
| 功能开发（默认） | `/do` |
| Bug 调查 + 修复 | `/omo` |
| 大型企业项目 | `/bmad-pilot` |
| 快速原型 | `/requirements-pilot` |
| 简单任务 | `/code`, `/debug` |

## 后端 CLI 要求

| 后端 | 必需功能 |
|------|----------|
| Codex | `codex e`, `--json`, `-C`, `resume` |
| Claude | `--output-format stream-json`, `-r` |
| Gemini | `-o stream-json`, `-y`, `-r` |
| OpenCode | `opencode`, stdin 模式 |

## 故障排查

**Codex wrapper 未找到：**
```bash
# 选择：codeagent-wrapper
npx github:cexll/myclaude
```

**模块未加载：**
```bash
cat ~/.claude/installed_modules.json
npx github:cexll/myclaude --force
```

## FAQ

| 问题 | 解决方案 |
|------|----------|
| "Unknown event format" | 日志显示问题，可忽略 |
| Gemini 无法读取 .gitignore 文件 | 从 .gitignore 移除或使用其他后端 |
| Codex 权限拒绝 | 在 ~/.codex/config.yaml 设置 `approval_policy = "never"` |

更多问题请访问 [GitHub Issues](https://github.com/cexll/myclaude/issues)。

## 许可证

AGPL-3.0 - 查看 [LICENSE](LICENSE)

### 商业授权

如需商业授权（无需遵守 AGPL 义务），请联系：evanxian9@gmail.com

## 支持

- [GitHub Issues](https://github.com/cexll/myclaude/issues)
