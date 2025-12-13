# Changelog

## 5.2.0 - 2025-12-12

### 🚀 Core Features

#### Multi-Backend Support (codeagent-wrapper)
- **Renamed**: `codex-wrapper` → `codeagent-wrapper` with pluggable backend architecture
- **Three Backends**: Codex (default), Claude, Gemini via `--backend` flag
- **Smart Parser**: Auto-detects backend JSON stream formats
- **Session Resume**: All backends support `-r <session_id>` cross-session resume
- **Parallel Execution**: DAG task scheduling with global and per-task backend configuration
- **Concurrency Control**: `CODEAGENT_MAX_PARALLEL_WORKERS` env var limits concurrent tasks (max 100)
- **Test Coverage**: 93.4% (backend.go 100%, config.go 97.8%, executor.go 96.4%)

#### GitHub Workflow Automation
- **`/gh-create-issue`**: Guided dialogue for structured issue creation
- **`/gh-issue-implement`**: Full issue-to-PR lifecycle
  - Issue analysis and requirements clarification
  - Development execution via codeagent-wrapper
  - Automated progress updates and PR creation
- **`/dev`**: 6-step minimal dev workflow with mandatory 90% test coverage

#### Hooks System
- **UserPromptSubmit**: Auto-activate skills based on context
- **PostToolUse**: Auto-validation/formatting after tool execution
- **Stop**: Cleanup and reporting on session end
- **Examples**: Skill auto-activation, pre-commit checks

#### Skills System
- **Auto-Activation**: `skill-rules.json` regex trigger rules
- **codeagent skill**: Multi-backend wrapper integration
- **Modular Design**: Easy to extend with custom skills

#### Installation System Enhancements
- **`merge_json` operation**: Auto-merge `settings.json` configuration
- **Modular Installation**: `python3 install.py --module dev,gh`
- **Verbose Logging**: `--verbose/-v` enables terminal real-time output
- **Streaming Output**: `op_run_command` streams bash script execution

### 📚 Documentation

- `docs/architecture.md` (21KB): Architecture overview with ASCII diagrams
- `docs/CODEAGENT-WRAPPER.md` (9KB): Complete usage guide
- `docs/HOOKS.md` (4KB): Customization guide
- `README.md`: Added documentation index, corrected default backend description

### 🔧 Important Fixes

#### codeagent-wrapper
- Fixed Claude/Gemini backend `-C` (workdir) and `-r` (resume) parameter support (codeagent-wrapper/backend.go:80-120)
- Corrected Claude backend permission flag logic `if cfg.SkipPermissions` (codeagent-wrapper/backend.go:95)
- Fixed parallel mode startup banner duplication (codeagent-wrapper/main.go:184-194 removed)
- Extract and display recent errors on abnormal exit `Logger.ExtractRecentErrors()` (codeagent-wrapper/logger.go:156)
- Added task block index to parallel config error messages (codeagent-wrapper/config.go:245)
- Refactored signal handling logic to avoid duplicate nil checks (codeagent-wrapper/main.go:290-305)
- Removed binary artifacts from tracking (codeagent-wrapper, *.test, coverage.out)

#### Installation Scripts
- Fixed issue #55: `op_run_command` uses Popen + selectors for real-time streaming output
- Fixed issue #56: Display recent errors instead of entire log
- Changed module list header from "Enabled" to "Default" to avoid ambiguity

#### CI/CD
- Removed `.claude/` config file validation step (.github/workflows/ci.yml:45)
- Updated version test case from 5.1.0 → 5.2.0 (codeagent-wrapper/main_test.go:23)

#### Commands & Documentation
- `gh-implement.md` → `gh-issue-implement.md` semantic naming
- Fixed README example command: `/gh-implement` → `/gh-issue-implement`
- Reverted `skills/codex/SKILL.md` to `codex-wrapper` for backward compatibility

#### dev-workflow
- Replaced Codex skill → codeagent skill throughout
- Added UI auto-detection: backend tasks use codex, UI tasks use gemini
- Corrected agent name: `develop-doc-generator` → `dev-plan-generator`

### ⚙️ Configuration & Environment Variables

#### New Environment Variables
- `CODEAGENT_SKIP_PERMISSIONS`: Control permission check behavior
  - Claude backend defaults to `--dangerously-skip-permissions` enabled, set to `true` to disable
  - Codex/Gemini backends default to permission checks enabled, set to `true` to skip
- `CODEAGENT_MAX_PARALLEL_WORKERS`: Parallel task concurrency limit (default: unlimited, recommended: 8, max: 100)

#### Configuration Files
- `config.json`: Added "gh" module definition
- `config.schema.json`: Added `op_merge_json` schema validation

### ⚠️ Breaking Changes

**codex-wrapper → codeagent-wrapper rename**

**Migration**:
```bash
python3 install.py --module dev --force
```

**Backward Compatibility**: `codex-wrapper/main.go` provides compatibility entry point

### 📦 Installation

```bash
# Full install (dev + GitHub workflow)
python3 install.py --module dev,gh

# List all modules
python3 install.py --list-modules

# Verbose logging mode
python3 install.py --module gh --verbose
```

### 🧪 Test Results

✅ **All tests passing**
- Overall coverage: 93.4%
- Security scan: 0 issues (gosec)
- Linting: Pass

### 📄 Related PRs & Issues

- PR #53: Enterprise Workflow with Multi-Backend Support
- Issue #55: Installation script execution not visible
- Issue #56: Unfriendly error logging on abnormal exit

### 👥 Contributors

- Claude Sonnet 4.5
- Claude Opus 4.5
- SWE-Agent-Bot
