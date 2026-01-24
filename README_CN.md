# Claude Code 多智能体工作流系统

[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL_v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![Claude Code](https://img.shields.io/badge/Claude-Code-blue)](https://claude.ai/code)
[![Version](https://img.shields.io/badge/Version-6.x-green)](https://github.com/cexll/myclaude)

> AI 驱动的开发自动化 - 多后端执行架构 (Codex/Claude/Gemini/OpenCode)

## 快速开始

```bash
git clone https://github.com/cexll/myclaude.git
cd myclaude
python3 install.py --install-dir ~/.claude
```

## 模块概览

| 模块 | 描述 | 文档 |
|------|------|------|
| [do](skills/do/README.md) | **推荐** - 7 阶段功能开发 + codeagent 编排 | `/do` 命令 |
| [dev](dev-workflow/README.md) | 轻量级开发工作流 + Codex 集成 | `/dev` 命令 |
| [omo](skills/omo/README.md) | 多智能体编排 + 智能路由 | `/omo` 命令 |
| [bmad](bmad-agile-workflow/README.md) | BMAD 敏捷工作流 + 6 个专业智能体 | `/bmad-pilot` 命令 |
| [requirements](requirements-driven-workflow/README.md) | 轻量级需求到代码流水线 | `/requirements-pilot` 命令 |
| [essentials](development-essentials/README.md) | 核心开发命令和工具 | `/code`, `/debug` 等 |
| [sparv](skills/sparv/README.md) | SPARV 工作流 (Specify→Plan→Act→Review→Vault) | `/sparv` 命令 |
| course | 课程开发（组合 dev + product-requirements + test-cases） | 组合模块 |

## 核心架构

| 角色 | 智能体 | 职责 |
|------|-------|------|
| **编排者** | Claude Code | 规划、上下文收集、验证 |
| **执行者** | codeagent-wrapper | 代码编辑、测试执行（Codex/Claude/Gemini/OpenCode 后端）|

## 工作流详解

### do 工作流（推荐）

7 阶段功能开发，通过 codeagent-wrapper 编排多个智能体。**大多数功能开发任务的首选工作流。**

```bash
/do "添加用户登录功能"
```

**7 阶段：**
| 阶段 | 名称 | 目标 |
|------|------|------|
| 1 | Discovery | 理解需求 |
| 2 | Exploration | 映射代码库模式 |
| 3 | Clarification | 解决歧义（**强制**）|
| 4 | Architecture | 设计实现方案 |
| 5 | Implementation | 构建功能（**需审批**）|
| 6 | Review | 捕获缺陷 |
| 7 | Summary | 记录结果 |

**智能体：**
- `code-explorer` - 代码追踪、架构映射
- `code-architect` - 设计方案、文件规划
- `code-reviewer` - 代码审查、简化建议
- `develop` - 实现代码、运行测试

---

### Dev 工作流

轻量级开发工作流，适合简单功能开发。

```bash
/dev "实现 JWT 用户认证"
```

**6 步流程：**
1. 需求澄清 - 交互式问答
2. Codex 深度分析 - 代码库探索
3. 开发计划生成 - 结构化任务分解
4. 并行执行 - Codex 并发执行
5. 覆盖率验证 - 强制 ≥90%
6. 完成总结 - 报告生成

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

---

## 安装

```bash
# 安装所有启用的模块
python3 install.py --install-dir ~/.claude

# 安装特定模块
python3 install.py --module dev

# 列出可用模块
python3 install.py --list-modules

# 强制覆盖
python3 install.py --force
```

### 模块配置

编辑 `config.json` 启用/禁用模块：

```json
{
  "modules": {
    "dev": { "enabled": true },
    "bmad": { "enabled": false },
    "requirements": { "enabled": false },
    "essentials": { "enabled": false },
    "omo": { "enabled": false },
    "sparv": { "enabled": false },
    "do": { "enabled": false },
    "course": { "enabled": false }
  }
}
```

## 工作流选择指南

| 场景 | 推荐 |
|------|------|
| 功能开发（默认） | `/do` |
| 轻量级功能 | `/dev` |
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

## 故障排查

**Codex wrapper 未找到：**
```bash
bash install.sh
```

**模块未加载：**
```bash
cat ~/.claude/installed_modules.json
python3 install.py --module <name> --force
```

## FAQ

| 问题 | 解决方案 |
|------|----------|
| "Unknown event format" | 日志显示问题，可忽略 |
| Gemini 无法读取 .gitignore 文件 | 从 .gitignore 移除或使用其他后端 |
| `/dev` 执行慢 | 检查日志，尝试更快模型，使用单一仓库 |
| Codex 权限拒绝 | 在 ~/.codex/config.yaml 设置 `approval_policy = "never"` |

更多问题请访问 [GitHub Issues](https://github.com/cexll/myclaude/issues)。

## 许可证

AGPL-3.0 - 查看 [LICENSE](LICENSE)

## 支持

- [GitHub Issues](https://github.com/cexll/myclaude/issues)
- [文档](docs/)
