# System Architecture

## Overview

Multi-agent AI development system with Claude Code as orchestrator and pluggable execution backends.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         User                                 │
└──────────────────┬──────────────────────────────────────────┘
                   │
                   │ /dev, /gh-implement, etc.
                   ▼
┌─────────────────────────────────────────────────────────────┐
│                   Claude Code (Orchestrator)                 │
│  ┌─────────────────────────────────────────────────────────┐│
│  │ - Planning & context gathering                          ││
│  │ - Requirements clarification                            ││
│  │ - Task breakdown                                        ││
│  │ - Verification & reporting                              ││
│  └─────────────────────────────────────────────────────────┘│
└──────────────────┬──────────────────────────────────────────┘
                   │
                   │ via codeagent-wrapper
                   ▼
┌─────────────────────────────────────────────────────────────┐
│              Codeagent-Wrapper (Execution Layer)             │
│  ┌──────────────────────────────────────────────────────────┤
│  │ Backend Interface                                        │
│  ├──────────────┬──────────────┬──────────────┐            │
│  │ Codex        │ Claude       │ Gemini       │            │
│  │ Backend      │ Backend      │ Backend      │            │
│  └──────────────┴──────────────┴──────────────┘            │
│                                                              │
│  ┌──────────────────────────────────────────────────────────┤
│  │ Features:                                                │
│  │ - Multi-backend execution                                │
│  │ - JSON stream parsing                                    │
│  │ - Session management                                     │
│  │ - Parallel task execution                                │
│  │ - Timeout handling                                       │
│  └──────────────────────────────────────────────────────────┘
└──────────────────┬──────────────────────────────────────────┘
                   │
                   │ CLI invocations
                   ▼
┌─────────────────────────────────────────────────────────────┐
│                    AI CLI Backends                           │
│  ┌──────────────┬──────────────┬──────────────┐            │
│  │ Codex CLI    │ Claude CLI   │ Gemini CLI   │            │
│  │              │              │              │            │
│  │ Code editing │ Reasoning    │ Fast proto   │            │
│  └──────────────┴──────────────┴──────────────┘            │
└─────────────────────────────────────────────────────────────┘
```

## Component Architecture

### 1. Orchestrator Layer (Claude Code)

**Responsibilities:**
- User interaction and requirements gathering
- Context analysis and exploration
- Task planning and breakdown
- Workflow coordination
- Verification and reporting

**Key Workflows:**
```
/dev
├── Requirements clarification (AskUserQuestion)
├── Codex analysis (Task tool → Explore agent)
├── Dev plan generation (Task tool → dev-plan-generator)
├── Parallel execution (codeagent-wrapper --parallel)
├── Coverage validation (≥90%)
└── Completion summary

/gh-implement <issue>
├── Issue analysis (gh issue view)
├── Clarification (if needed)
├── Development (codeagent-wrapper or /dev)
├── Progress updates (gh issue comment)
└── PR creation (gh pr create)
```

### 2. Execution Layer (Codeagent-Wrapper)

**Architecture:**

```go
┌─────────────────────────────────────────────────────────┐
│                    Main Entry Point                      │
│  - Parse CLI arguments                                   │
│  - Detect mode (new/resume/parallel)                     │
│  - Select backend                                        │
└──────────────────┬──────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────┐
│                   Backend Selection                       │
│  func SelectBackend(name string) Backend                 │
│  ┌──────────────┬──────────────┬──────────────┐         │
│  │ CodexBackend │ ClaudeBackend│ GeminiBackend│         │
│  └──────────────┴──────────────┴──────────────┘         │
└──────────────────┬──────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────┐
│                   Executor                                │
│  func RunCodexTask(cfg *Config) (string, error)          │
│  ┌──────────────────────────────────────────────────────┤
│  │ 1. Build command args via Backend.BuildArgs()        │
│  │ 2. Start process with timeout                        │
│  │ 3. Stream stdout/stderr                              │
│  │ 4. Parse JSON stream via ParseJSONStream()           │
│  │ 5. Extract session ID                                │
│  │ 6. Handle signals (SIGINT, SIGTERM)                  │
│  └──────────────────────────────────────────────────────┘
└──────────────────┬──────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────┐
│                   Parser                                  │
│  func ParseJSONStream(r io.Reader) (string, string)      │
│  ┌──────────────────────────────────────────────────────┤
│  │ Detects format:                                      │
│  │ - Codex: {"type":"thread.started","thread_id":...}  │
│  │ - Claude: {"type":"...","subtype":"result"}         │
│  │ - Gemini: {"type":"...","role":"assistant"}         │
│  │                                                      │
│  │ Extracts:                                            │
│  │ - Agent messages                                     │
│  │ - Session IDs                                        │
│  └──────────────────────────────────────────────────────┘
└─────────────────────────────────────────────────────────┘
```

**Backend Interface:**

```go
type Backend interface {
    Name() string
    Command() string
    BuildArgs(cfg *Config, targetArg string) []string
}

// Codex: codex e --skip-git-repo-check -C <workdir> --json <task>
// Claude: claude -p --dangerously-skip-permissions --output-format stream-json --verbose <task>
// Gemini: gemini -o stream-json -y -p <task>
```

**Key Files:**
- `main.go` - Entry point and orchestration
- `config.go` - CLI argument parsing
- `backend.go` - Backend interface and implementations
- `executor.go` - Process execution and stream handling
- `parser.go` - JSON stream parsing (multi-format)
- `logger.go` - Async logging with ring buffer
- `utils.go` - Helper functions

### 3. Hooks System

**Architecture:**

```
┌─────────────────────────────────────────────────────────┐
│                  Claude Code Events                       │
│  UserPromptSubmit │ PostToolUse │ Stop                   │
└──────────────────┬──────────────────────────────────────┘
                   │
                   │ reads
                   ▼
┌─────────────────────────────────────────────────────────┐
│              .claude/settings.json                        │
│  {                                                        │
│    "hooks": {                                             │
│      "UserPromptSubmit": [                                │
│        {                                                  │
│          "hooks": [                                       │
│            {                                              │
│              "type": "command",                           │
│              "command": "$CLAUDE_PROJECT_DIR/hooks/..."   │
│            }                                              │
│          ]                                                │
│        }                                                  │
│      ]                                                    │
│    }                                                      │
│  }                                                        │
└──────────────────┬──────────────────────────────────────┘
                   │
                   │ executes
                   ▼
┌─────────────────────────────────────────────────────────┐
│                   Hook Scripts                            │
│  ┌────────────────────────────────────────────────────┐  │
│  │ skill-activation-prompt.sh                         │  │
│  │ - Reads skills/skill-rules.json                    │  │
│  │ - Matches user prompt against triggers             │  │
│  │ - Injects skill suggestions                        │  │
│  └────────────────────────────────────────────────────┘  │
│  ┌────────────────────────────────────────────────────┐  │
│  │ pre-commit.sh                                      │  │
│  │ - Validates staged files                           │  │
│  │ - Runs tests                                       │  │
│  │ - Formats code                                     │  │
│  └────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

### 4. Skills System

**Structure:**

```
skills/
├── codex/
│   └── SKILL.md         # Codex CLI integration
├── codeagent/
│   └── SKILL.md         # Multi-backend wrapper
├── gemini/
│   └── SKILL.md         # Gemini CLI integration
└── skill-rules.json     # Auto-activation rules
```

**skill-rules.json Format:**

```json
{
  "rules": [
    {
      "trigger": {
        "pattern": "implement|build|create feature",
        "type": "regex"
      },
      "skill": "codeagent",
      "priority": 1,
      "suggestion": "Use codeagent skill for code implementation"
    }
  ]
}
```

## Data Flow

### Example: /dev Workflow

```
1. User: /dev "add user authentication"
   │
   ▼
2. Claude Code:
   │ ├─ Clarifies requirements (AskUserQuestion)
   │ ├─ Analyzes codebase (Explore agent)
   │ └─ Generates dev-plan.md
   │
   ▼
3. Claude Code invokes: codeagent-wrapper --parallel <<EOF
   ---TASK---
   id: auth_backend_1701234567
   workdir: /project/backend
   ---CONTENT---
   implement JWT authentication in @src/auth

   ---TASK---
   id: auth_frontend_1701234568
   workdir: /project/frontend
   dependencies: auth_backend_1701234567
   ---CONTENT---
   implement login form consuming /api/auth

   ---TASK---
   id: auth_tests_1701234569
   workdir: /project
   dependencies: auth_backend_1701234567, auth_frontend_1701234568
   ---CONTENT---
   add integration tests for auth flow
   EOF
   │
   ▼
4. Codeagent-Wrapper:
   │ ├─ Parses parallel config
   │ ├─ Topological sort (resolves dependencies)
   │ ├─ Executes tasks concurrently:
   │ │  ├─ Task 1: codex e --json "implement JWT..."
   │ │  ├─ Task 2: waits for Task 1, then codex e --json "implement login..."
   │ │  └─ Task 3: waits for Tasks 1&2, then codex e --json "add tests..."
   │ └─ Aggregates results
   │
   ▼
5. Claude Code:
   │ ├─ Validates coverage (≥90%)
   │ ├─ Runs final tests
   │ └─ Reports summary
   │
   ▼
6. User receives:
   ✅ Authentication implemented
   📊 Coverage: 92%
   📁 Files modified: 8
   🧪 Tests: 24 passed
```

## Module System

Installation system uses modular architecture:

```
config.json
├── dev module (enabled)
│   ├── merge_dir: dev-workflow → ~/.claude
│   ├── copy_file: memorys/CLAUDE.md
│   ├── copy_file: skills/codex/SKILL.md
│   └── run_command: install codeagent-wrapper binary
│
├── gh module (enabled)
│   ├── merge_dir: github-workflow → ~/.claude
│   ├── copy_file: skills/codeagent/SKILL.md
│   ├── copy_dir: hooks → ~/.claude/hooks
│   └── merge_json: hooks-config.json → settings.json
│
└── essentials module (enabled)
    └── merge_dir: development-essentials → ~/.claude
```

**Installation Flow:**

```bash
python3 install.py --module dev,gh

1. Load config.json
2. Validate against config.schema.json
3. Select modules: dev, gh
4. Execute operations:
   ├─ dev:
   │  ├─ Merge dev-workflow/commands → ~/.claude/commands
   │  ├─ Copy CLAUDE.md → ~/.claude/CLAUDE.md
   │  ├─ Copy codex skill → ~/.claude/skills/codex/
   │  └─ Run install.sh (compile codeagent-wrapper)
   │
   └─ gh:
      ├─ Merge github-workflow/commands → ~/.claude/commands
      ├─ Copy codeagent skill → ~/.claude/skills/codeagent/
      ├─ Copy hooks → ~/.claude/hooks
      └─ Merge hooks-config.json → ~/.claude/settings.json
5. Write installed_modules.json
6. Log to install.log
```

## Parallel Execution Engine

**Algorithm:**

```
1. Parse task config (---TASK--- delimited format)
2. Build dependency graph
3. Topological sort (detect cycles)
4. Execute in layers:
   Layer 0: Tasks with no dependencies
   Layer 1: Tasks depending only on Layer 0
   Layer 2: Tasks depending on Layers 0-1
   ...
5. Within each layer: unlimited concurrency
6. On failure: skip dependent tasks, continue others
7. Aggregate results
```

**Example:**

```
Tasks:
  A (no deps)
  B (no deps)
  C (depends on A)
  D (depends on A, B)
  E (depends on D)

Execution:
  Layer 0: A, B (parallel)
  Layer 1: C (waits for A), D (waits for A, B)
  Layer 2: E (waits for D)

Timeline:
  t=0:   Start A, B
  t=1:   A completes
  t=2:   B completes, start C, D
  t=3:   C completes
  t=4:   D completes, start E
  t=5:   E completes
```

## Security Considerations

1. **No credential storage** - Uses existing CLI auth
2. **Sandbox execution** - Tasks run in specified workdir
3. **Timeout enforcement** - Prevents runaway processes
4. **Signal handling** - Graceful shutdown on Ctrl+C
5. **Input validation** - Sanitizes task configs

## Performance Characteristics

| Operation | Complexity | Notes |
|-----------|------------|-------|
| Backend selection | O(1) | Map lookup |
| Task parsing | O(n) | Linear scan |
| Topological sort | O(V+E) | Kahn's algorithm |
| Parallel execution | O(depth) | Depth of dependency graph |
| JSON parsing | O(n) | Streaming parser |

## Extensibility

### Adding New Backend

1. Implement `Backend` interface:
```go
type NewBackend struct{}

func (NewBackend) Name() string { return "new" }
func (NewBackend) Command() string { return "new-cli" }
func (NewBackend) BuildArgs(cfg *Config, targetArg string) []string {
    return []string{"--json", targetArg}
}
```

2. Register in `config.go`:
```go
backendRegistry = map[string]Backend{
    "codex":  CodexBackend{},
    "claude": ClaudeBackend{},
    "gemini": GeminiBackend{},
    "new":    NewBackend{},
}
```

3. Add JSON format detection in `parser.go`:
```go
if hasKey(obj, "new_specific_field") {
    // Parse new backend format
}
```

### Adding New Hook

1. Create script in `hooks/`:
```bash
#!/bin/bash
# hooks/my-custom-hook.sh
echo "Hook executed"
```

2. Register in `.claude/settings.json`:
```json
{
  "hooks": {
    "UserPromptSubmit": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "$CLAUDE_PROJECT_DIR/hooks/my-custom-hook.sh"
          }
        ]
      }
    ]
  }
}
```

### Adding New Skill

1. Create `skills/my-skill/SKILL.md`:
```markdown
---
name: my-skill
description: My custom skill
---

# My Custom Skill

Usage instructions...
```

2. Add activation rule in `skills/skill-rules.json`:
```json
{
  "rules": [
    {
      "trigger": {"pattern": "my keyword", "type": "regex"},
      "skill": "my-skill",
      "priority": 1
    }
  ]
}
```

## Further Reading

- [Codeagent-Wrapper Guide](./CODEAGENT-WRAPPER.md)
- [GitHub Workflow Guide](./GITHUB-WORKFLOW.md)
- [Hooks Documentation](./HOOKS.md)
- [README](../README.md)
