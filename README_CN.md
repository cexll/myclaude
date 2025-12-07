# Claude Code 多智能体工作流系统

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Claude Code](https://img.shields.io/badge/Claude-Code-blue)](https://claude.ai/code)
[![Version](https://img.shields.io/badge/Version-5.0-green)](https://github.com/cexll/myclaude)

> AI 驱动的开发自动化 - Claude Code + Codex 协作

## 核心概念：Claude Code + Codex

本系统采用**双智能体架构**：

| 角色 | 智能体 | 职责 |
|------|-------|------|
| **编排者** | Claude Code | 规划、上下文收集、验证、用户交互 |
| **执行者** | Codex | 代码编辑、测试执行、文件操作 |

**为什么分离？**
- Claude Code 擅长理解上下文和编排复杂工作流
- Codex 擅长专注的代码生成和执行
- 两者结合效果优于单独使用

## 快速开始（windows上请在Powershell中执行）

```bash
git clone https://github.com/cexll/myclaude.git
cd myclaude
python3 install.py --install-dir ~/.claude
```

## 工作流概览

### 1. Dev 工作流（推荐）

**大多数开发任务的首选工作流。**

```bash
/dev "实现 JWT 用户认证"
```

**6 步流程：**
1. **需求澄清** - 交互式问答明确范围
2. **Codex 深度分析** - 代码库探索和架构决策
3. **开发计划生成** - 结构化任务分解和测试要求
4. **并行执行** - Codex 并发执行任务
5. **覆盖率验证** - 强制 ≥90% 测试覆盖率
6. **完成总结** - 文件变更和覆盖率报告

**核心特性：**
- Claude Code 编排，Codex 执行所有代码变更
- 自动任务并行化提升速度
- 强制 90% 测试覆盖率门禁
- 失败自动回滚

**适用场景：** 功能开发、重构、带测试的 bug 修复

---

### 2. BMAD 敏捷工作流

**包含 6 个专业智能体的完整企业敏捷方法论。**

```bash
/bmad-pilot "构建电商结账系统"
```

**智能体角色：**
| 智能体 | 职责 |
|-------|------|
| Product Owner | 需求与用户故事 |
| Architect | 系统设计与技术决策 |
| Tech Lead | Sprint 规划与任务分解 |
| Developer | 实现 |
| Code Reviewer | 质量保证 |
| QA Engineer | 测试与验证 |

**流程：**
```
需求 → 架构 → Sprint计划 → 开发 → 审查 → QA
 ↓      ↓       ↓         ↓      ↓      ↓
PRD.md DESIGN.md SPRINT.md Code REVIEW.md TEST.md
```

**适用场景：** 大型功能、团队协作、企业项目

---

### 3. 需求驱动工作流

**轻量级需求到代码流水线。**

```bash
/requirements-pilot "实现 API 限流"
```

**流程：**
1. 带质量评分的需求生成
2. 实现规划
3. 代码生成
4. 审查和测试

**适用场景：** 快速原型、明确定义的功能

---

### 4. 开发基础命令

**日常编码任务的直接命令。**

| 命令 | 用途 |
|------|------|
| `/code` | 实现功能 |
| `/debug` | 调试问题 |
| `/test` | 编写测试 |
| `/review` | 代码审查 |
| `/optimize` | 性能优化 |
| `/refactor` | 代码重构 |
| `/docs` | 编写文档 |

**适用场景：** 快速任务，无需工作流开销

---

## 安装

### 模块化安装（推荐）

```bash
# 安装所有启用的模块（默认：dev + essentials）
python3 install.py --install-dir ~/.claude

# 安装特定模块
python3 install.py --module dev

# 列出可用模块
python3 install.py --list-modules

# 强制覆盖现有文件
python3 install.py --force
```

### 可用模块

| 模块 | 默认 | 描述 |
|------|------|------|
| `dev` | ✓ 启用 | Dev 工作流 + Codex 集成 |
| `essentials` | ✓ 启用 | 核心开发命令 |
| `bmad` | 禁用 | 完整 BMAD 敏捷工作流 |
| `requirements` | 禁用 | 需求驱动工作流 |

### 安装内容

```
~/.claude/
├── CLAUDE.md              # 核心指令和角色定义
├── commands/              # 斜杠命令 (/dev, /code 等)
├── agents/                # 智能体定义
├── skills/
│   └── codex/
│       └── SKILL.md       # Codex 集成技能
└── installed_modules.json # 安装状态
```

### 配置

编辑 `config.json` 自定义：

```json
{
  "version": "1.0",
  "install_dir": "~/.claude",
  "modules": {
    "dev": {
      "enabled": true,
      "operations": [
        {"type": "merge_dir", "source": "dev-workflow"},
        {"type": "copy_file", "source": "memorys/CLAUDE.md", "target": "CLAUDE.md"},
        {"type": "copy_file", "source": "skills/codex/SKILL.md", "target": "skills/codex/SKILL.md"},
        {"type": "run_command", "command": "bash install.sh"}
      ]
    }
  }
}
```

**操作类型：**
| 类型 | 描述 |
|------|------|
| `merge_dir` | 合并子目录 (commands/, agents/) 到安装目录 |
| `copy_dir` | 复制整个目录 |
| `copy_file` | 复制单个文件到目标路径 |
| `run_command` | 执行 shell 命令 |

---

## Codex 集成

`codex` 技能使 Claude Code 能够将代码执行委托给 Codex CLI。

### 工作流中的使用

```bash
# 通过技能调用 Codex
codex-wrapper - <<'EOF'
在 @src/auth.ts 中实现 JWT 验证
EOF
```

### 并行执行

```bash
codex-wrapper --parallel <<'EOF'
---TASK---
id: backend_api
workdir: /project/backend
---CONTENT---
实现 /api/users 的 REST 端点

---TASK---
id: frontend_ui
workdir: /project/frontend
dependencies: backend_api
---CONTENT---
创建消费 API 的 React 组件
EOF
```

### 安装 Codex Wrapper

```bash
# 自动（通过 dev 模块）
python3 install.py --module dev

# 手动
bash install.sh
```

#### Windows 系统

Windows 系统会将 `codex-wrapper.exe` 安装到 `%USERPROFILE%\bin`。

```powershell
# PowerShell（推荐）
powershell -ExecutionPolicy Bypass -File install.ps1

# 批处理（cmd）
install.bat
```

**添加到 PATH**（如果安装程序未自动检测）：

```powershell
# PowerShell - 永久添加（当前用户）
[Environment]::SetEnvironmentVariable('PATH', "$HOME\bin;" + [Environment]::GetEnvironmentVariable('PATH','User'), 'User')

# PowerShell - 仅当前会话
$Env:PATH = "$HOME\bin;$Env:PATH"
```

```batch
REM cmd.exe - 永久添加（当前用户）
setx PATH "%USERPROFILE%\bin;%PATH%"
```

---

## 工作流选择指南

| 场景 | 推荐工作流 |
|------|----------|
| 带测试的新功能 | `/dev` |
| 快速 bug 修复 | `/debug` 或 `/code` |
| 大型多 Sprint 功能 | `/bmad-pilot` |
| 原型或 POC | `/requirements-pilot` |
| 代码审查 | `/review` |
| 性能问题 | `/optimize` |

---

## 故障排查

### 常见问题

**Codex wrapper 未找到：**
```bash
# 检查 PATH
echo $PATH | grep -q "$HOME/bin" || echo 'export PATH="$HOME/bin:$PATH"' >> ~/.zshrc

# 重新安装
bash install.sh
```

**权限被拒绝：**
```bash
python3 install.py --install-dir ~/.claude --force
```

**模块未加载：**
```bash
# 检查安装状态
cat ~/.claude/installed_modules.json

# 重新安装特定模块
python3 install.py --module dev --force
```

---

## 许可证

MIT License - 查看 [LICENSE](LICENSE)

## 支持

- **问题反馈**: [GitHub Issues](https://github.com/cexll/myclaude/issues)
- **文档**: [docs/](docs/)

---

**Claude Code + Codex = 更好的开发** - 编排遇见执行。
