# codeagent-wrapper

[English](README.md) | [中文](README_CN.md)

`codeagent-wrapper` 是一个用 Go 编写的多后端 AI 代码代理命令行包装器：用统一的 CLI 入口封装不同的 AI 工具后端（Codex / Claude / Gemini / OpenCode），并提供一致的参数、配置、技能注入与会话恢复体验。

入口：`cmd/codeagent-wrapper/main.go`（生成二进制名：`codeagent-wrapper`）。

## 功能特性

- **多后端支持**：`codex` / `claude` / `gemini` / `opencode`
- **统一命令行**：`codeagent-wrapper [flags] <task>` / `codeagent-wrapper resume <session_id> <task> [workdir]`
- **自动 stdin**：遇到换行/特殊字符/超长任务自动走 stdin，避免 shell quoting 问题；也可显式使用 `-`
- **配置合并**：支持配置文件与 `CODEAGENT_*` 环境变量（viper）
- **Agent 预设**：从 `~/.codeagent/models.json` 读取 backend/model/prompt/reasoning/yolo/allowed_tools 等预设
- **动态 Agent**：在 `~/.codeagent/agents/{name}.md` 放置 prompt 文件即可作为 agent 使用
- **技能自动注入**：`--skills` 手动指定，或根据项目技术栈自动检测（Go/Rust/Python/Node.js/Vue）并注入对应技能规范
- **Git Worktree 隔离**：`--worktree` 在独立 git worktree 中执行任务，自动生成 task_id 和分支
- **并行执行**：`--parallel` 从 stdin 读取多任务配置，支持依赖拓扑并发执行，带结构化摘要报告
- **后端配置**：`models.json` 的 `backends` 节支持 per-backend 的 `base_url` / `api_key` 注入
- **Claude 工具控制**：`allowed_tools` / `disallowed_tools` 限制 Claude 后端可用工具
- **Stderr 降噪**：自动过滤 Gemini 和 Codex 后端的噪声 stderr 输出
- **日志清理**：`codeagent-wrapper cleanup` 清理旧日志（日志写入系统临时目录）
- **跨平台**：支持 macOS / Linux / Windows

## 安装

### 推荐方式（交互式安装器）

```bash
npx github:stellarlinkco/myclaude
```

选择 `codeagent-wrapper` 模块进行安装。

### 手动构建

要求：Go 1.21+。

```bash
# 从源码构建
make build

# 或直接安装到 $GOPATH/bin
make install
```

安装后确认：

```bash
codeagent-wrapper --version
```

## 使用示例

最简单用法（默认后端：`codex`）：

```bash
codeagent-wrapper "分析 internal/app/cli.go 的入口逻辑，给出改进建议"
```

指定后端：

```bash
codeagent-wrapper --backend claude "解释 internal/executor/parallel_config.go 的并行配置格式"
```

指定工作目录（第 2 个位置参数）：

```bash
codeagent-wrapper "在当前 repo 下搜索潜在数据竞争" .
```

显式从 stdin 读取 task（使用 `-`）：

```bash
cat task.txt | codeagent-wrapper -
```

使用 HEREDOC（推荐用于多行任务）：

```bash
codeagent-wrapper --backend claude - <<'EOF'
实现用户认证系统：
- JWT 令牌
- bcrypt 密码哈希
- 会话管理
EOF
```

恢复会话：

```bash
codeagent-wrapper resume <session_id> "继续上次任务"
```

在 git worktree 中隔离执行：

```bash
codeagent-wrapper --worktree "重构认证模块"
```

手动指定技能注入：

```bash
codeagent-wrapper --skills golang-base-practices "优化数据库查询"
```

并行模式（从 stdin 读取任务配置）：

```bash
codeagent-wrapper --parallel <<'EOF'
---TASK---
id: t1
workdir: .
backend: codex
---CONTENT---
列出本项目的主要模块以及它们的职责。
---TASK---
id: t2
dependencies: t1
backend: claude
---CONTENT---
基于 t1 的结论，提出重构风险点与建议。
EOF
```

## CLI 参数

| 参数 | 说明 |
|------|------|
| `--backend <name>` | 后端选择（codex/claude/gemini/opencode） |
| `--model <name>` | 覆盖模型 |
| `--agent <name>` | Agent 预设名（来自 models.json 或 ~/.codeagent/agents/） |
| `--prompt-file <path>` | 从文件读取 prompt |
| `--skills <names>` | 逗号分隔的技能名，注入对应规范 |
| `--reasoning-effort <level>` | 推理力度（后端相关） |
| `--skip-permissions` | 跳过权限提示 |
| `--dangerously-skip-permissions` | `--skip-permissions` 的别名 |
| `--worktree` | 在新 git worktree 中执行（自动生成 task_id） |
| `--parallel` | 并行任务模式（从 stdin 读取配置） |
| `--full-output` | 并行模式下输出完整消息（默认仅输出摘要） |
| `--config <path>` | 配置文件路径（默认：`$HOME/.codeagent/config.*`） |
| `--version`, `-v` | 打印版本号 |
| `--cleanup` | 清理旧日志 |

## 配置说明

### 配置文件

默认查找路径（当 `--config` 为空时）：

- `$HOME/.codeagent/config.(yaml|yml|json|toml|...)`

示例（YAML）：

```yaml
backend: codex
model: gpt-4.1
skip-permissions: false
```

也可以通过 `--config /path/to/config.yaml` 显式指定。

### 环境变量（`CODEAGENT_*`）

通过 viper 读取并自动映射 `-` 为 `_`，常用项：

| 变量 | 说明 |
|------|------|
| `CODEAGENT_BACKEND` | 后端名（codex/claude/gemini/opencode） |
| `CODEAGENT_MODEL` | 模型名 |
| `CODEAGENT_AGENT` | Agent 预设名 |
| `CODEAGENT_PROMPT_FILE` | Prompt 文件路径 |
| `CODEAGENT_REASONING_EFFORT` | 推理力度 |
| `CODEAGENT_SKIP_PERMISSIONS` | 跳过权限提示（默认 true；设 `false` 关闭） |
| `CODEAGENT_FULL_OUTPUT` | 并行模式完整输出 |
| `CODEAGENT_MAX_PARALLEL_WORKERS` | 并行 worker 数（0=不限制，上限 100） |
| `CODEAGENT_TMPDIR` | 自定义临时目录（macOS 权限问题时使用） |
| `CODEX_TIMEOUT` | 超时（毫秒，默认 7200000 即 2 小时） |
| `CODEX_BYPASS_SANDBOX` | Codex sandbox bypass（默认 true；设 `false` 关闭） |
| `DO_WORKTREE_DIR` | 复用已有 worktree 目录（由 /do 工作流设置） |

### Agent 预设（`~/.codeagent/models.json`）

```json
{
  "default_backend": "codex",
  "default_model": "gpt-4.1",
  "backends": {
    "codex": { "api_key": "..." },
    "claude": { "base_url": "http://localhost:23001", "api_key": "..." }
  },
  "agents": {
    "develop": {
      "backend": "codex",
      "model": "gpt-4.1",
      "prompt_file": "~/.codeagent/prompts/develop.md",
      "reasoning": "high",
      "yolo": true,
      "allowed_tools": ["Read", "Write", "Bash"],
      "disallowed_tools": ["WebFetch"]
    }
  }
}
```

用 `--agent <name>` 选择预设，agent 会继承 `backends` 下对应后端的 `base_url` / `api_key`。

### 动态 Agent

在 `~/.codeagent/agents/` 目录放置 `{name}.md` 文件，即可通过 `--agent {name}` 使用，自动读取该 Markdown 作为 prompt，使用 `default_backend` 和 `default_model`。

### 技能自动检测

当未通过 `--skills` 显式指定技能时，codeagent-wrapper 会根据工作目录中的文件自动检测技术栈：

| 检测文件 | 注入技能 |
|----------|----------|
| `go.mod` / `go.sum` | `golang-base-practices` |
| `Cargo.toml` | `rust-best-practices` |
| `pyproject.toml` / `setup.py` / `requirements.txt` | `python-best-practices` |
| `package.json` | `vercel-react-best-practices`, `frontend-design` |
| `vue.config.js` / `vite.config.ts` / `nuxt.config.ts` | `vue-web-app` |

技能规范从 `~/.claude/skills/{name}/SKILL.md` 读取，受 16000 字符预算限制。

## 支持的后端

该项目本身不内置模型能力，依赖本机安装并可在 `PATH` 中找到对应 CLI：

| 后端 | 执行命令 | 说明 |
|------|----------|------|
| `codex` | `codex e ...` | 默认添加 `--dangerously-bypass-approvals-and-sandbox`；设 `CODEX_BYPASS_SANDBOX=false` 关闭 |
| `claude` | `claude -p ... --output-format stream-json` | 默认跳过权限并禁用 setting-sources 防止递归；设 `CODEAGENT_SKIP_PERMISSIONS=false` 开启权限；自动读取 `~/.claude/settings.json` 中的 env 和 model |
| `gemini` | `gemini -o stream-json -y ...` | 自动从 `~/.gemini/.env` 加载环境变量（GEMINI_API_KEY, GEMINI_MODEL 等） |
| `opencode` | `opencode run --format json` | — |

## 项目结构

```
cmd/codeagent-wrapper/main.go   # CLI 入口
internal/
  app/          # CLI 命令定义、参数解析、主逻辑编排
  backend/      # 后端抽象与实现（codex/claude/gemini/opencode）
  config/       # 配置加载、agent 解析、viper 绑定
  executor/     # 任务执行引擎：单任务/并行/worktree/技能注入
  logger/       # 结构化日志系统
  parser/       # JSON stream 解析器
  utils/        # 通用工具函数
  worktree/     # Git worktree 管理
```

## 开发

```bash
make build    # 构建
make test     # 运行测试
make lint     # golangci-lint + staticcheck
make clean    # 清理构建产物
make install  # 安装到 $GOPATH/bin
```

CI 使用 GitHub Actions，Go 1.21 / 1.22 矩阵测试。

## 故障排查

- macOS 下如果看到临时目录相关的 `permission denied`，可设置：`CODEAGENT_TMPDIR=$HOME/.codeagent/tmp`
- `claude` 后端的 `base_url` / `api_key`（来自 `~/.codeagent/models.json` 的 `backends.claude`）会注入到子进程环境变量 `ANTHROPIC_BASE_URL` / `ANTHROPIC_API_KEY`
- `gemini` 后端的 API key 从 `~/.gemini/.env` 加载，注入 `GEMINI_API_KEY` 并自动设置 `GEMINI_API_KEY_AUTH_MECHANISM=bearer`
- 后端命令未找到时返回退出码 127，超时返回 124，中断返回 130
- 并行模式默认输出结构化摘要，使用 `--full-output` 查看完整输出以便调试
