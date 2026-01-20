# codeagent-wrapper

`codeagent-wrapper` 是一个用 Go 编写的“多后端 AI 代码代理”命令行包装器：用统一的 CLI 入口封装不同的 AI 工具后端（Codex / Claude / Gemini / Opencode），并提供一致的参数、配置与会话恢复体验。

入口：`cmd/codeagent/main.go`（生成二进制名：`codeagent`）。

## 功能特性

- 多后端支持：`codex` / `claude` / `gemini` / `opencode`
- 统一命令行：`codeagent [flags] <task>` / `codeagent resume <session_id> <task> [workdir]`
- 自动 stdin：遇到换行/特殊字符/超长任务自动走 stdin，避免 shell quoting 地狱；也可显式使用 `-`
- 配置合并：支持配置文件与 `CODEAGENT_*` 环境变量（viper）
- Agent 预设：从 `~/.codeagent/models.json` 读取 backend/model/prompt 等预设
- 并行执行：`--parallel` 从 stdin 读取多任务配置，支持依赖拓扑并发执行
- 日志清理：`codeagent cleanup` 清理旧日志（日志写入系统临时目录）

## 安装

要求：Go 1.21+。

在仓库根目录执行：

```bash
go install ./cmd/codeagent
```

安装后确认：

```bash
codeagent version
```

## 使用示例

最简单用法（默认后端：`codex`）：

```bash
codeagent "分析 internal/app/cli.go 的入口逻辑，给出改进建议"
```

指定后端：

```bash
codeagent --backend claude "解释 internal/executor/parallel_config.go 的并行配置格式"
```

指定工作目录（第 2 个位置参数）：

```bash
codeagent "在当前 repo 下搜索潜在数据竞争" .
```

显式从 stdin 读取 task（使用 `-`）：

```bash
cat task.txt | codeagent -
```

恢复会话：

```bash
codeagent resume <session_id> "继续上次任务"
```

并行模式（从 stdin 读取任务配置；禁止位置参数）：

```bash
codeagent --parallel <<'EOF'
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

- `CODEAGENT_BACKEND`（`codex|claude|gemini|opencode`）
- `CODEAGENT_MODEL`
- `CODEAGENT_AGENT`
- `CODEAGENT_PROMPT_FILE`
- `CODEAGENT_REASONING_EFFORT`
- `CODEAGENT_SKIP_PERMISSIONS`
- `CODEAGENT_FULL_OUTPUT`（并行模式 legacy 输出）
- `CODEAGENT_MAX_PARALLEL_WORKERS`（0 表示不限制，上限 100）

### Agent 预设（`~/.codeagent/models.json`）

可在 `~/.codeagent/models.json` 定义 agent → backend/model/prompt 等映射，用 `--agent <name>` 选择：

```json
{
  "default_backend": "opencode",
  "default_model": "opencode/grok-code",
  "agents": {
    "develop": {
      "backend": "codex",
      "model": "gpt-4.1",
      "prompt_file": "~/.codeagent/prompts/develop.md",
      "description": "Code development"
    }
  }
}
```

## 支持的后端

该项目本身不内置模型能力，依赖你本机安装并可在 `PATH` 中找到对应 CLI：

- `codex`：执行 `codex e ...`（默认会添加 `--dangerously-bypass-approvals-and-sandbox`；如需关闭请设置 `CODEX_BYPASS_SANDBOX=false`）
- `claude`：执行 `claude -p ... --output-format stream-json`（默认会跳过权限提示；如需开启请设置 `CODEAGENT_SKIP_PERMISSIONS=false`）
- `gemini`：执行 `gemini ... -o stream-json`（可从 `~/.gemini/.env` 加载环境变量）
- `opencode`：执行 `opencode run --format json`

## 开发

```bash
make build
make test
make lint
make clean
```

