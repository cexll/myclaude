---
name: gemini
description: Execute Gemini CLI for AI-powered code analysis and generation. Use when you need to leverage Google's Gemini models for complex reasoning tasks.
---

# Gemini CLI Integration

## Overview

Execute Gemini CLI commands with support for multiple models and flexible prompt input. Integrates Google's Gemini AI models into Claude Code workflows.

## When to Use

- Complex reasoning tasks requiring advanced AI capabilities
- Code generation and analysis with Gemini models
- Tasks requiring Google's latest AI technology
- Alternative perspective on code problems

## Usage

**推荐方式**（使用 uv run，自动管理 Python 环境）：
```bash
uv run ~/.claude/skills/gemini/scripts/gemini.py -m <model> -p "<prompt>" [working_dir]
```

**备选方式**（直接执行或使用 Python）：
```bash
~/.claude/skills/gemini/scripts/gemini.py -m <model> -p "<prompt>" [working_dir]
# 或
python3 ~/.claude/skills/gemini/scripts/gemini.py -m <model> -p "<prompt>" [working_dir]
```

## Environment Variables

- **GEMINI_MODEL**: Override default model (default: `gemini-3-pro-preview`)
  - Example: `export GEMINI_MODEL=gemini-3`
- **GEMINI_TIMEOUT**: Override timeout in milliseconds (default: 7200000 = 2 hours)
  - Example: `export GEMINI_TIMEOUT=3600000` for 1 hour

## Timeout Control

- **Built-in**: Script enforces 2-hour timeout by default
- **Override**: Set `GEMINI_TIMEOUT` environment variable (in milliseconds)
- **Bash tool**: Always set `timeout: 7200000` parameter for double protection

### Parameters

- `-m, --model` (optional): Model to use (default: gemini-3-pro-preview)
  - `gemini-3-pro-preview`: Latest flagship model
- `-p, --prompt` (required): Task prompt or question
- `working_dir` (optional): Working directory (default: current)

### Return Format

Plain text output from Gemini:

```text
Model response text here...
```

Error format (stderr):

```text
ERROR: Error message
```

### Invocation Pattern

When calling via Bash tool, always include the timeout parameter:

```yaml
Bash tool parameters:
- command: uv run ~/.claude/skills/gemini/scripts/gemini.py -m gemini-3-pro-preview -p "<prompt>"
- timeout: 7200000
- description: <brief description of the task>
```

Alternatives:

```yaml
# Direct execution (simplest)
- command: ~/.claude/skills/gemini/scripts/gemini.py -m gemini-3-pro-preview -p "<prompt>"

# Using python3
- command: python3 ~/.claude/skills/gemini/scripts/gemini.py -m gemini-3-pro-preview -p "<prompt>"
```

### Examples

**Basic query:**

```bash
# Recommended: via uv run
uv run ~/.claude/skills/gemini/scripts/gemini.py -m gemini-3-pro-preview -p "explain quantum computing"
# timeout: 7200000

# Alternative: direct execution
~/.claude/skills/gemini/scripts/gemini.py -m gemini-3-pro-preview -p "explain quantum computing"
```

**Code analysis:**

```bash
uv run ~/.claude/skills/gemini/scripts/gemini.py -m gemini-3-pro-preview -p "review this code for security issues: $(cat app.py)"
# timeout: 7200000
```

**With specific working directory:**

```bash
uv run ~/.claude/skills/gemini/scripts/gemini.py -m gemini-3-pro-preview -p "analyze project structure" "/path/to/project"
# timeout: 7200000
```

**Using fast model:**

```bash
uv run ~/.claude/skills/gemini/scripts/gemini.py -m gemini-3-pro-preview -p "quick code suggestion"
# timeout: 7200000
```

**Using python3 directly (alternative):**

```bash
python3 ~/.claude/skills/gemini/scripts/gemini.py -m gemini-3-pro-preview -p "your prompt here"
```

## Notes

- **Recommended**: Use `uv run` for automatic Python environment management (requires uv installed)
- **Alternative**: Direct execution `./gemini.py` (uses system Python via shebang)
- Python implementation using standard library (zero dependencies)
- Cross-platform compatible (Windows/macOS/Linux)
- PEP 723 compliant (inline script metadata)
- Requires Gemini CLI installed and authenticated
- Supports all Gemini model variants
- Output is streamed directly from Gemini CLI
