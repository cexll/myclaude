# Claude Code 多智能体工作流系统

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Claude Code](https://img.shields.io/badge/Claude-Code-blue)](https://claude.ai/code)
[![Version](https://img.shields.io/badge/Version-3.2-green)](https://github.com/cexll/myclaude)
[![Plugin Ready](https://img.shields.io/badge/Plugin-Ready-purple)](https://docs.claude.com/en/docs/claude-code/plugins)

> 企业级敏捷开发自动化与 AI 驱动的多智能体编排

[English](README.md) | [文档](docs/)

## 🚀 快速开始

### 安装

**插件系统（推荐）**
```bash
/plugin github.com/cexll/myclaude
```

**传统安装**
```bash
git clone https://github.com/cexll/myclaude.git
cd myclaude
make install
```

### 基本使用

```bash
# 完整敏捷工作流
/bmad-pilot "构建用户认证系统，支持 OAuth2 和多因素认证"

# 轻量级开发
/requirements-pilot "实现 JWT 令牌刷新"

# 直接开发命令
/code "添加 API 限流功能"
```

## 📦 插件模块

| 插件 | 描述 | 主要命令 |
|------|------|---------|
| **[bmad-agile-workflow](docs/BMAD-WORKFLOW.md)** | 完整 BMAD 方法论，包含6个专业智能体 | `/bmad-pilot` |
| **[requirements-driven-workflow](docs/REQUIREMENTS-WORKFLOW.md)** | 精简的需求到代码工作流 | `/requirements-pilot` |
| **[development-essentials](docs/DEVELOPMENT-COMMANDS.md)** | 核心开发斜杠命令 | `/code` `/debug` `/test` `/optimize` |
| **[advanced-ai-agents](docs/ADVANCED-AGENTS.md)** | GPT-5 深度推理集成 | 智能体: `gpt5` |
| **[requirements-clarity](docs/REQUIREMENTS-CLARITY.md)** | 自动需求澄清，100分制质量评分 | 自动激活技能 |

## 💡 使用场景

**BMAD 工作流** - 完整敏捷流程自动化
- 产品需求 → 架构设计 → 冲刺规划 → 开发实现 → 代码审查 → 质量测试
- 90% 阈值质量门控
- 自动生成文档

**Requirements 工作流** - 快速原型开发
- 需求生成 → 实现 → 审查 → 测试
- 轻量级实用主义

**开发命令** - 日常编码
- 直接实现、调试、测试、优化
- 无工作流开销

**需求澄清** - 自动化需求工程
- 自动检测模糊需求并启动澄清流程
- 100分制质量评分系统
- 生成完整的产品需求文档

## 🎯 核心特性

- **🤖 角色化智能体**: 每个开发阶段的专业 AI 智能体
- **📊 质量门控**: 自动质量评分，迭代优化
- **✅ 确认节点**: 关键工作流阶段的用户确认
- **📁 持久化产物**: 所有规格保存至 `.claude/specs/`
- **🔌 插件系统**: 原生 Claude Code 插件支持
- **🔄 灵活工作流**: 选择完整敏捷或轻量开发
- **🎯 需求澄清**: 自动化需求澄清与质量评分

## 📚 文档

- **[BMAD 工作流指南](docs/BMAD-WORKFLOW.md)** - 完整方法论和智能体角色
- **[Requirements 工作流](docs/REQUIREMENTS-WORKFLOW.md)** - 轻量级开发流程
- **[开发命令参考](docs/DEVELOPMENT-COMMANDS.md)** - 斜杠命令说明
- **[插件系统](docs/PLUGIN-SYSTEM.md)** - 安装与配置
- **[快速上手](docs/QUICK-START.md)** - 5分钟入门

## 🛠️ 安装方式

**方式1: 插件安装**（一条命令）
```bash
/plugin install bmad-agile-workflow
```

**方式2: Make 命令**（选择性安装）
```bash
make deploy-bmad          # 仅 BMAD 工作流
make deploy-requirements  # 仅 Requirements 工作流
make deploy-all          # 全部安装
```

**方式3: 手动安装**
- 复制 `/commands/*.md` 到 `~/.config/claude/commands/`
- 复制 `/agents/*.md` 到 `~/.config/claude/agents/`

运行 `make help` 查看所有选项。

## 📄 许可证

MIT 许可证 - 查看 [LICENSE](LICENSE)

## 🙋 支持

- **问题反馈**: [GitHub Issues](https://github.com/cexll/myclaude/issues)
- **文档**: [docs/](docs/)
- **插件指南**: [PLUGIN_README.md](PLUGIN_README.md)

---

**使用 AI 驱动的自动化转型您的开发流程** - 一条命令，完整工作流，质量保证。
