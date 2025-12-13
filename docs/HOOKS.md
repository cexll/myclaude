# Claude Code Hooks Guide

Hooks are shell scripts or commands that execute in response to Claude Code events.

## Available Hook Types

### 1. UserPromptSubmit
Runs after user submits a prompt, before Claude processes it.

**Use cases:**
- Auto-activate skills based on keywords
- Add context injection
- Log user requests

### 2. PostToolUse
Runs after Claude uses a tool.

**Use cases:**
- Validate tool outputs
- Run additional checks (linting, formatting)
- Log tool usage

### 3. Stop
Runs when Claude Code session ends.

**Use cases:**
- Cleanup temporary files
- Generate session reports
- Commit changes automatically

## Configuration

Hooks are configured in `.claude/settings.json`:

```json
{
  "hooks": {
    "UserPromptSubmit": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "$CLAUDE_PROJECT_DIR/hooks/skill-activation-prompt.sh"
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "$CLAUDE_PROJECT_DIR/hooks/post-tool-check.sh"
          }
        ]
      }
    ]
  }
}
```

## Creating Custom Hooks

### Example: Pre-Commit Hook

**File:** `hooks/pre-commit.sh`

```bash
#!/bin/bash
set -e

# Get staged files
STAGED_FILES=$(git diff --cached --name-only --diff-filter=ACM)

# Run tests on Go files
GO_FILES=$(echo "$STAGED_FILES" | grep '\.go$' || true)
if [ -n "$GO_FILES" ]; then
  go test ./... -short || exit 1
fi

# Validate JSON files
JSON_FILES=$(echo "$STAGED_FILES" | grep '\.json$' || true)
if [ -n "$JSON_FILES" ]; then
  for file in $JSON_FILES; do
    jq empty "$file" || exit 1
  done
fi

echo "✅ Pre-commit checks passed"
```

**Register in settings.json:**

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "$CLAUDE_PROJECT_DIR/hooks/pre-commit.sh"
          }
        ]
      }
    ]
  }
}
```

### Example: Auto-Format Hook

**File:** `hooks/auto-format.sh`

```bash
#!/bin/bash

# Format Go files
find . -name "*.go" -exec gofmt -w {} \;

# Format JSON files
find . -name "*.json" -exec jq --indent 2 . {} \; -exec mv {} {}.tmp \; -exec mv {}.tmp {} \;

echo "✅ Files formatted"
```

## Environment Variables

Hooks have access to:
- `$CLAUDE_PROJECT_DIR` - Project root directory
- `$PWD` - Current working directory
- All shell environment variables

## Best Practices

1. **Keep hooks fast** - Slow hooks block Claude Code
2. **Handle errors gracefully** - Return non-zero on failure
3. **Use absolute paths** - Reference `$CLAUDE_PROJECT_DIR`
4. **Make scripts executable** - `chmod +x hooks/script.sh`
5. **Test independently** - Run hooks manually first
6. **Document behavior** - Add comments explaining logic

## Debugging Hooks

Enable verbose logging:

```bash
# Add to your hook
set -x  # Print commands
set -e  # Exit on error
```

Test manually:

```bash
cd /path/to/project
./hooks/your-hook.sh
echo $?  # Check exit code
```

## Built-in Hooks

This repository includes:

| Hook | File | Purpose |
|------|------|---------|
| Skill Activation | `skill-activation-prompt.sh` | Auto-suggest skills |
| Pre-commit | `pre-commit.sh` | Code quality checks |

## Disabling Hooks

Remove hook configuration from `.claude/settings.json` or set empty array:

```json
{
  "hooks": {
    "UserPromptSubmit": []
  }
}
```

## Troubleshooting

**Hook not running?**
- Check `.claude/settings.json` syntax
- Verify script is executable: `ls -l hooks/`
- Check script path is correct

**Hook failing silently?**
- Add `set -e` to script
- Check exit codes: `echo $?`
- Add logging: `echo "debug" >> /tmp/hook.log`

## Further Reading

- [Claude Code Hooks Documentation](https://docs.anthropic.com/claude-code/hooks)
- [Bash Scripting Guide](https://www.gnu.org/software/bash/manual/)
