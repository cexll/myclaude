# Claude Code 多智能体工作流系统

[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL_v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![Claude Code](https://img.shields.io/badge/Claude-Code-blue)](https://claude.ai/code)
[![Version](https://img.shields.io/badge/Version-5.2-green)](https://github.com/cexll/myclaude)

> AI 驱动的开发自动化 - 多后端执行架构 (Codex/Claude/Gemini)

## 核心概念：多后端架构

本系统采用**双智能体架构**与可插拔 AI 后端：

| 角色 | 智能体 | 职责 |
|------|-------|------|
| **编排者** | Claude Code | 规划、上下文收集、验证、用户交互 |
| **执行者** | codeagent-wrapper | 代码编辑、测试执行（Codex/Claude/Gemini 后端）|

**为什么分离？**
- Claude Code 擅长理解上下文和编排复杂工作流
- 专业后端（Codex 擅长代码、Claude 擅长推理、Gemini 擅长原型）专注执行
- 通过 `--backend codex|claude|gemini` 匹配模型与任务

## 快速开始（windows上请在Powershell中执行）

```bash
git clone https://github.com/cexll/myclaude.git
cd myclaude
python3 install.py --install-dir ~/.claude
```

## 工作流概览

### 0. OmO 多智能体编排器（复杂任务推荐）

**基于风险信号智能路由任务到专业智能体的多智能体编排系统。**

```bash
/omo "分析并修复这个认证 bug"
```

**智能体层级：**
| 智能体 | 角色 | 后端 | 模型 |
|-------|------|------|------|
| `oracle` | 技术顾问 | Claude | claude-opus-4-5 |
| `librarian` | 外部研究 | Claude | claude-sonnet-4-5 |
| `explore` | 代码库搜索 | OpenCode | grok-code |
| `develop` | 代码实现 | Codex | gpt-5.2 |
| `frontend-ui-ux-engineer` | UI/UX 专家 | Gemini | gemini-3-pro |
| `document-writer` | 文档撰写 | Gemini | gemini-3-flash |

**路由信号（非固定流水线）：**
- 代码位置不明确 → `explore`
- 外部库/API → `librarian`
- 高风险/多文件变更 → `oracle`
- 需要实现 → `develop` / `frontend-ui-ux-engineer`

**常用配方：**
- 解释代码：`explore`
- 位置已知的小修复：直接 `develop`
- Bug 修复，位置未知：`explore → develop`
- 跨模块重构：`explore → oracle → develop`
- 外部 API 集成：`explore + librarian → oracle → develop`

**适用场景：** 复杂 bug 调查、多文件重构、架构决策

---

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
├── bin/
│   └── codeagent-wrapper    # 主可执行文件
├── CLAUDE.md                # 核心指令和角色定义
├── commands/                # 斜杠命令 (/dev, /code 等)
├── agents/                  # 智能体定义
├── skills/
│   └── codex/
│       └── SKILL.md         # Codex 集成技能
├── config.json              # 配置文件
└── installed_modules.json   # 安装状态
```

### 自定义安装目录

默认情况下，myclaude 安装到 `~/.claude`。您可以使用 `INSTALL_DIR` 环境变量自定义安装目录：

```bash
# 安装到自定义目录
INSTALL_DIR=/opt/myclaude bash install.sh

# 相应更新您的 PATH
export PATH="/opt/myclaude/bin:$PATH"
```

**目录结构：**
- `$INSTALL_DIR/bin/` - codeagent-wrapper 可执行文件
- `$INSTALL_DIR/skills/` - 技能定义
- `$INSTALL_DIR/config.json` - 配置文件
- `$INSTALL_DIR/commands/` - 斜杠命令定义
- `$INSTALL_DIR/agents/` - 智能体定义

**注意：** 使用自定义安装目录时，请确保将 `$INSTALL_DIR/bin` 添加到您的 `PATH` 环境变量中。

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
codeagent-wrapper - <<'EOF'
在 @src/auth.ts 中实现 JWT 验证
EOF
```

### 并行执行

```bash
codeagent-wrapper --parallel <<'EOF'
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

Windows 系统会将 `codeagent-wrapper.exe` 安装到 `%USERPROFILE%\bin`。

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
REM cmd.exe - 永久添加（当前用户）（建议使用上面的 PowerShell 方法）
REM 警告：此命令会展开 %PATH% 包含系统 PATH，导致重复
REM 注意：使用 reg add 而非 setx 以避免 1024 字符截断限制
reg add "HKCU\Environment" /v Path /t REG_EXPAND_SZ /d "%USERPROFILE%\bin;%PATH%" /f
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
# 安装程序会自动添加 PATH，检查是否已添加
if [[ ":$PATH:" != *":$HOME/.claude/bin:"* ]]; then
    echo "PATH not configured. Reinstalling..."
    bash install.sh
fi

# 或手动添加（幂等性命令）
[[ ":$PATH:" != *":$HOME/.claude/bin:"* ]] && echo 'export PATH="$HOME/.claude/bin:$PATH"' >> ~/.zshrc
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

## 常见问题 (FAQ)

### Q1: `codeagent-wrapper` 执行时报错 "Unknown event format"

**问题描述：**
执行 `codeagent-wrapper` 时出现错误：
```
Unknown event format: {"type":"turn.started"}
Unknown event format: {"type":"assistant", ...}
```

**解决方案：**
这是日志事件流的显示问题，不影响实际功能执行。预计在下个版本中修复。如需排查其他问题，可忽略此日志输出。

**相关 Issue：** [#96](https://github.com/cexll/myclaude/issues/96)

---

### Q2: Gemini 无法读取 `.gitignore` 忽略的文件

**问题描述：**
使用 `codeagent-wrapper --backend gemini` 时，无法读取 `.claude/` 等被 `.gitignore` 忽略的目录中的文件。

**解决方案：**
- **方案一：** 在项目根目录的 `.gitignore` 中取消对 `.claude/` 的忽略
- **方案二：** 确保需要读取的文件不在 `.gitignore` 忽略列表中

**相关 Issue：** [#75](https://github.com/cexll/myclaude/issues/75)

---

### Q3: `/dev` 命令并行执行特别慢

**问题描述：**
使用 `/dev` 命令开发简单功能耗时过长（超过30分钟），无法了解任务执行状态。

**解决方案：**
1. **检查日志：** 查看 `C:\Users\User\AppData\Local\Temp\codeagent-wrapper-*.log` 分析瓶颈
2. **调整后端：**
   - 尝试使用 `gpt-5.1-codex-max` 等更快的模型
   - 在 WSL 环境下运行速度可能更快
3. **工作区选择：** 使用独立的代码仓库而非包含多个子项目的 monorepo

**相关 Issue：** [#77](https://github.com/cexll/myclaude/issues/77)

---

### Q4: 新版 Go 实现的 Codex 权限不足

**问题描述：**
升级到新版 Go 实现的 Codex 后，出现权限不足的错误。

**解决方案：**
在 `~/.codex/config.yaml` 中添加以下配置（Windows: `c:\user\.codex\config.toml`）：
```yaml
model = "gpt-5.1-codex-max"
model_reasoning_effort = "high"
model_reasoning_summary = "detailed"
approval_policy = "never"
sandbox_mode = "workspace-write"
disable_response_storage = true
network_access = true
```

**关键配置说明：**
- `approval_policy = "never"` - 移除审批限制
- `sandbox_mode = "workspace-write"` - 允许工作区写入权限
- `network_access = true` - 启用网络访问

**相关 Issue：** [#31](https://github.com/cexll/myclaude/issues/31)

---

### Q5: 执行时遇到权限拒绝或沙箱限制

**问题描述：**
运行 codeagent-wrapper 时出现权限错误或沙箱限制。

**解决方案：**
设置以下环境变量：
```bash
export CODEX_BYPASS_SANDBOX=true
export CODEAGENT_SKIP_PERMISSIONS=true
```

或添加到 shell 配置文件（`~/.zshrc` 或 `~/.bashrc`）：
```bash
echo 'export CODEX_BYPASS_SANDBOX=true' >> ~/.zshrc
echo 'export CODEAGENT_SKIP_PERMISSIONS=true' >> ~/.zshrc
```

**注意：** 这些设置会绕过安全限制，请仅在可信环境中使用。

---

**仍有疑问？** 请访问 [GitHub Issues](https://github.com/cexll/myclaude/issues) 搜索或提交新问题。

---

## 许可证

AGPL-3.0 License - 查看 [LICENSE](LICENSE)

## 支持

- **问题反馈**: [GitHub Issues](https://github.com/cexll/myclaude/issues)
- **文档**: [docs/](docs/)

---

**Claude Code + Codex = 更好的开发** - 编排遇见执行。
