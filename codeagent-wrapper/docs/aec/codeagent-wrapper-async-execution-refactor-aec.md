# Agent Execution Contract: codeagent-wrapper Async Execution Refactor

**Version**: 1.0
**Date**: 2026-04-09
**Author**: AEC Compiler
**Quality Score**: 94/100
**Status**: Final

## Quick Reference (Agent Context)

> **Goal**: Refactor `codeagent-wrapper` into a thin CLI over an async task execution application layer with persistent temp logs, explicit cleanup, and dependency-aware parallel summaries.
> **Non-Goals**: Detaching into a long-lived daemon, changing user-facing `resume` semantics, changing `--parallel` task config format, deleting logs at task completion
> **Primary Workflow**: User runs `codeagent-wrapper --backend <backend> <task>` or `codeagent-wrapper --parallel`, the wrapper executes backend tasks asynchronously inside the process, writes logs to temp files, and streams or prints per-task completion summaries until all requested work finishes.
> **Success Metric**: Existing CLI compatibility for `resume`, `--parallel`, and `cleanup` is preserved while timeout-driven termination is removed and async task completion is driven by backend events/channel coordination.
> **Architecture Style**: Modular monolith with thin CLI adapter and hexagonal application core
> **Primary Stack**: Go 1.21 + Cobra + Viper + zerolog + gopsutil
> **Naming Rule**: Commands: kebab-case | Files: snake_case existing-compatible | Types: PascalCase | Functions: camelCase | Log files: `codeagent-wrapper-<pid>[-suffix].log`
> **Repo**: `git@github.com:stellarlinkco/myclaude.git`
> **Base Branch**: `master`
> **Merge Target**: `master`
> **Dev Agent**: Codex
> **Review Agent**: Codex
> **Key Constraint**: Preserve `resume`, preserve `--parallel` config format, preserve `cleanup` / `--cleanup` as historical log cleanup commands

---

## Executive Summary

`codeagent-wrapper` currently mixes CLI parsing, backend process execution, timeout handling, logger lifecycle, log cleanup, and parallel aggregation in the same application-facing package. That coupling makes behavior hard to reason about and makes timeout removal risky because cancellation, result delivery, and cleanup are not isolated concerns.

This refactor separates the system into a thin CLI adapter and an application core that owns task execution, result aggregation, log references, and cleanup policies. Backend commands continue to run as child processes, but completion must be driven by parser events and process exit rather than wrapper-managed execution timeouts.

The delivery target is not a daemon. The process still runs in the foreground for the invoked command, but internally uses async coordination and channels so single-task and parallel-task execution follow one model. Logs remain available after completion for debugging and are removed only by explicit cleanup or stale-log retention logic.

---

## Goals

- Remove timeout-driven execution cancellation from `codeagent-wrapper` runtime behavior.
- Refactor the codebase so CLI parsing is separate from task execution orchestration.
- Make single-task execution and parallel execution share the same async orchestration model.
- Preserve temp log files for post-run debugging and expose them in task results.
- Keep historical log cleanup as an explicit command and a safe background maintenance policy.
- Preserve compatibility for `resume` and `--parallel` task config format.

## Non-Goals

- Do not introduce a detached daemon, background service, or external broker in v1.
- Do not change backend CLI contracts for Codex, Claude, Gemini, or OpenCode beyond removing wrapper timeout behavior.
- Do not redesign prompt injection, skill auto-detection, worktree behavior, or review/report extraction beyond what is required to support the refactor.
- Do not delete task logs immediately on task completion.
- Do not replace the current `cleanup` semantics with destructive cleanup of active logs.

---

## Confirmed Facts / Assumptions / Open Questions

### Confirmed Facts
- The project is a Go CLI wrapper with entry point `cmd/codeagent-wrapper/main.go`.
- Current supported backends are `codex`, `claude`, `gemini`, and `opencode`.
- Current CLI supports single task mode, `resume`, `--parallel`, `cleanup`, `--cleanup`, worktree mode, skill injection, and structured output.
- Current implementation stores logs in the system temp directory and already has stale log cleanup logic based on PID liveness and PID reuse checks.
- Current branch and merge target for this work are both `master`.
- Development and review agents for this feature are both Codex.

### Working Assumptions
- Existing users depend on command-line compatibility more than package-level Go API compatibility.
- It is acceptable to move internal packages and types as long as CLI behavior and structured output remain compatible.
- The current parser event model is sufficient to drive async task completion without introducing a new persistent queue.
- A default parallel worker limit of 10 is acceptable if it remains configurable.

### Open Questions (Non-Blocking)
- Whether future detached execution should reuse the same task registry introduced by this refactor.
- Whether log retention should later become time-based, size-based, or both.

### Build Blockers (Must Resolve Before Execution)
- None

---

## User Stories & Acceptance Criteria

### Story 1: Run a single backend task without wrapper timeouts

**As a** CLI user **I want to** run a backend task through `codeagent-wrapper` **so that** the wrapper waits for backend completion without enforcing an execution timeout.

**Acceptance Criteria:**
- [ ] Running `codeagent-wrapper --backend codex "task"` does not read or apply `CODEX_TIMEOUT`.
- [ ] Wrapper-managed execution does not return exit code `124` for timeout expiration.
- [ ] Backend completion is determined by backend output parsing and process exit, not by wrapper timeout.
- [ ] The final output still includes the backend message and `SESSION_ID` when available.
- [ ] The task log path is recorded and remains on disk after completion.

### Story 2: Run multiple tasks concurrently with per-task completion summaries

**As a** CLI user **I want to** run `--parallel` tasks **so that** each completed task yields a summary and the final report includes all task outcomes.

**Acceptance Criteria:**
- [ ] Existing `--parallel` task block format remains valid.
- [ ] The default concurrency limit is 10 and can be overridden by configuration or environment.
- [ ] Independent tasks can run concurrently while dependency order remains enforced.
- [ ] When a task finishes, its result is captured without waiting for unrelated running tasks to finish first.
- [ ] The final summary lists every task result, including log path and extracted summary fields.

### Story 3: Preserve logs for debugging and clean them later

**As a** CLI user **I want to** keep task logs after execution **so that** I can debug failures and clean historical logs explicitly when I choose.

**Acceptance Criteria:**
- [ ] Task logs are not deleted immediately after task completion.
- [ ] `codeagent-wrapper cleanup` and `codeagent-wrapper --cleanup` still remove historical stale logs and then exit.
- [ ] Cleanup never removes logs owned by active processes.
- [ ] Startup cleanup, if retained, only targets stale historical logs and does not block task execution.
- [ ] Cleanup results report scanned, deleted, kept, and errored files.

### Story 4: Keep `resume` behavior stable through the refactor

**As a** repeat user **I want to** keep using `resume` **so that** multi-step workflows continue to work after the refactor.

**Acceptance Criteria:**
- [ ] `resume <session_id> <task>` remains a supported CLI entry.
- [ ] Session ID propagation remains backend-specific but behaviorally unchanged from the user perspective.
- [ ] Refactoring does not require users to learn a new session continuation flow.

---

## Functional Requirements

### FR-1: Thin CLI Adapter
- **Description**: The CLI layer must parse commands, validate flags, map user input to application requests, and print application results without directly owning backend process orchestration.
- **Trigger**: Any CLI invocation of `codeagent-wrapper`.
- **Expected Result**: CLI code delegates task execution, cleanup, and summarization to application use cases through explicit interfaces.
- **Traces to**: Story 1 / Story 2 / Story 3 / Goal 2

### FR-2: Async In-Process Task Execution
- **Description**: Single-task and parallel-task flows must execute through a shared async orchestration model using channels or equivalent completion signaling.
- **Trigger**: A single task request or a parallel task set request.
- **Expected Result**: The wrapper can start backend work, observe completion events, collect final results, and return them without polling on a wrapper timeout.
- **Traces to**: Story 1 / Story 2 / Goal 1 / Goal 3

### FR-3: Timeout-Free Runtime Semantics
- **Description**: Wrapper-managed execution must not enforce execution time limits for backend tasks.
- **Trigger**: Any backend task run through the application layer.
- **Expected Result**: No runtime path reads `CODEX_TIMEOUT` or returns timeout-based cancellation due to wrapper elapsed time.
- **Traces to**: Story 1 / Goal 1

### FR-4: Persistent Log References
- **Description**: Every task result must include a stable log reference for debugging.
- **Trigger**: Task start and task completion.
- **Expected Result**: Logs are created in temp storage, written asynchronously, exposed in results, and retained until cleanup removes them.
- **Traces to**: Story 1 / Story 3 / Goal 4

### FR-5: Safe Historical Cleanup
- **Description**: Cleanup must remove stale historical logs while preserving logs of active processes.
- **Trigger**: `cleanup`, `--cleanup`, and optional startup janitor execution.
- **Expected Result**: Only stale logs are deleted, with safety checks for symlinks, PID liveness, and PID reuse.
- **Traces to**: Story 3 / Goal 5

### FR-6: Parallel Compatibility
- **Description**: The refactor must keep the current `--parallel` config shape and dependency semantics intact.
- **Trigger**: `--parallel` mode invocation.
- **Expected Result**: Existing task block headers continue to parse, dependencies continue to skip correctly on failed parents, and per-task summaries remain available.
- **Traces to**: Story 2 / Goal 6

### FR-7: Resume Compatibility
- **Description**: The refactor must preserve current resume entry points and session ID result handling.
- **Trigger**: `resume` mode invocation.
- **Expected Result**: Session continuation behavior remains user-compatible across supported backends.
- **Traces to**: Story 4 / Goal 6

---

## Architecture Overview

### Architecture Principles
1. **CLI is an adapter, not an orchestrator**: Command parsing must not contain backend lifecycle logic.
2. **One execution model**: Single and parallel modes must share the same async task runtime.
3. **Logs are artifacts, not side effects**: Log creation and retention are first-class outputs of task execution.
4. **Cleanup is policy-driven**: Cleanup behavior must be isolated from task completion semantics.
5. **Compatibility at the command boundary**: Preserve user-facing commands while simplifying internals.

### High-Level Architecture

```text
CLI (cobra)
  -> Application Use Cases
      -> Task Runtime Port
      -> Log Store Port
      -> Cleanup Port
      -> Result Summarizer Port
  -> Infrastructure Adapters
      -> Backend Process Runner
      -> Stream Parser
      -> Temp Log Store
      -> PID-based Janitor
```

### Component Responsibilities

| Component | Responsibility | Depends On | Depended By |
|-----------|---------------|------------|-------------|
| `cli` adapter | Parse flags, map commands, print results | application ports | user entry point |
| `application/taskrun` | Execute one task request and return result | runtime port, log store port | cli, parallel use case |
| `application/taskset` | Execute dependency-ordered task sets, aggregate results | task runtime, summarizer | cli |
| `application/cleanup` | Execute historical log cleanup policy | cleanup port | cli |
| `domain/task` | Task request/result/state contracts | none | application |
| `infra/backend` | Spawn backend commands and surface stream events | OS process APIs | runtime port |
| `infra/logstore` | Create and persist temp logs | filesystem | application/runtime |
| `infra/janitor` | Remove stale logs safely | filesystem, process inspection | cleanup use case |
| `infra/parser` | Convert backend stream JSON into completion events | backend stdout/stderr | runtime port |

---

## Implementation Patterns

> Primary reference for agents during execution. Every pattern is mechanical.

### Naming Conventions

| Layer | Convention | Example |
|-------|-----------|---------|
| CLI commands | kebab-case | `codeagent-wrapper cleanup` |
| Source files | existing snake_case | `task_runtime.go` |
| Types/Interfaces | PascalCase | `TaskRunner`, `CleanupPolicy` |
| Functions/Methods | camelCase | `RunTask`, `CollectResult` |
| Channels | `<name>Ch` suffix | `resultCh`, `eventCh` |
| Domain IDs | descriptive string aliases | `TaskID`, `SessionID` |
| Temp log files | `codeagent-wrapper-<pid>[-suffix].log` | `codeagent-wrapper-12345-task-a.log` |

### Code Organization

**Dependency direction** (strict — never reverse):
```text
cli -> application -> domain
infra -> application -> domain
```

### API Patterns

The CLI remains command-based, not HTTP-based. Application-layer request/response structs must be explicit:

```go
type RunTaskRequest struct {
    Task TaskSpec
}

type RunTaskResponse struct {
    Result TaskResult
}
```

### Error Handling
- **Boundary rule**: Parse and validate at CLI and config boundaries; application and domain consume typed requests.
- **Error hierarchy**: Distinguish request validation errors, backend launch errors, backend execution errors, and cleanup errors.
- **Logging**: Structured log lines with task ID, backend, pid, event type, and log path when available.
- **Cancellation**: Preserve explicit user cancellation via context or signal; remove wrapper-generated timeout cancellation.

### Testing Patterns
- **Acceptance tests**: Exercise CLI-facing or application-boundary behavior for single-task execution, parallel execution, cleanup, and resume compatibility.
- **Unit tests**: Cover parser event mapping, cleanup safety logic, result aggregation, and dependency skipping.
- **Architecture tests**: Enforce that CLI does not import backend-execution details directly.
- **Coverage target**: Preserve current project target of 90% where practical for touched packages.

---

## Project Structure

```text
internal/
├── domain/
│   └── task/                 # task contracts, states, cleanup policy value types
├── application/
│   ├── runtask/              # single-task use case
│   ├── runtaskset/           # parallel/dependency-aware use case
│   ├── cleanup/              # cleanup use case
│   └── ports/                # task runtime, log store, janitor, summarizer
├── infrastructure/
│   ├── backend/              # codex/claude/gemini/opencode command runners
│   ├── parser/               # backend stream event parsing
│   ├── logstore/             # temp-file logger and log references
│   └── janitor/              # stale log cleanup implementation
└── cli/
    └── cobra/                # root command and command adapters
```

**Feature-to-directory mapping:**

| PRD Feature/FR | Directory | Layer |
|---------------|-----------|-------|
| FR-1 Thin CLI Adapter | `internal/cli` | adapter |
| FR-2 Async In-Process Task Execution | `internal/application/runtask` | application |
| FR-3 Timeout-Free Runtime Semantics | `internal/application/runtask`, `internal/infrastructure/backend` | application / infrastructure |
| FR-4 Persistent Log References | `internal/infrastructure/logstore` | infrastructure |
| FR-5 Safe Historical Cleanup | `internal/application/cleanup`, `internal/infrastructure/janitor` | application / infrastructure |
| FR-6 Parallel Compatibility | `internal/application/runtaskset` | application |
| FR-7 Resume Compatibility | `internal/application/runtask`, `internal/infrastructure/backend` | application / infrastructure |

---

## Data Models

### Core Entities

| Entity | Key Fields | Relationships |
|--------|-----------|---------------|
| `TaskRequest` | `TaskID`, `Backend`, `Mode`, `WorkDir`, `Prompt`, `SessionID` | input to task runtime |
| `TaskResult` | `TaskID`, `ExitCode`, `Message`, `SessionID`, `Error`, `LogRef` | output of task runtime |
| `TaskSetRequest` | `Tasks`, `Dependencies`, `WorkerLimit` | input to parallel runtime |
| `LogRef` | `Path`, `OwnerPID`, `TaskID` | linked from `TaskResult` |
| `CleanupStats` | `Scanned`, `Deleted`, `Kept`, `Errors` | output of cleanup use case |

### Migration Strategy
- No database migration work is required in v1.
- Internal type aliases may be used temporarily during migration, then removed after call sites are updated.

---

## API Design

This feature does not add external HTTP APIs. The command contract to preserve is:

| Method | Endpoint | Purpose | Auth |
|--------|----------|---------|------|
| CLI | `codeagent-wrapper <task>` | run single task | local |
| CLI | `codeagent-wrapper resume <session_id> <task>` | resume task | local |
| CLI | `codeagent-wrapper --parallel` | run dependency-aware task set from stdin | local |
| CLI | `codeagent-wrapper cleanup` | clean historical logs and exit | local |
| CLI | `codeagent-wrapper --cleanup` | clean historical logs and exit | local |

---

## Security & Constraints

### Authentication & Authorization
- No new authentication surface is introduced.
- Existing backend credential injection rules remain unchanged.

### Performance
- Default parallel worker limit is 10.
- Worker limit must remain configurable.
- Cleanup must not block task execution longer than necessary; startup cleanup should remain asynchronous if retained.

### Compliance
- No external compliance changes are required.

### External Dependencies
- `gopsutil` remains the source of process-liveness checks for safe log cleanup.
- Backend CLIs remain external runtime dependencies and must stay optional per selected backend.

---

## Execution Contract

### Repository
- **URL**: `git@github.com:stellarlinkco/myclaude.git`
- **Type**: monorepo module (`codeagent-wrapper` subdirectory)

### Branch Strategy
- **Base branch**: `master`
- **Merge target**: `master`
- **Branch naming**: `feature/codeagent-wrapper-async-refactor` or task-scoped derivative

### Agent Configuration
- **Development agent**: Codex
- **Review agent**: Codex
- **Max review iterations**: 3
- **Concurrency limit**: 10 by default; configurable

### CI/CD Requirements
- **Required checks**: `go test ./...`, build success, touched-package tests passing
- **Auto-merge condition**: CI green plus review approval by designated review agent or human override

### Pre-Check Criteria
- **Before execution**: repository available locally, `master` up to date enough for feature branch creation, Go toolchain available, backend commands installed for target tests, temp directory writable

### Escalation Rules
- **Tier 1 (Agent self-resolve)**: package moves, interface extraction, test rewrites, removal of timeout code, cleanup refactors
- **Tier 2 (Coordinator arbitrate)**: command compatibility ambiguity, output format differences, package-boundary disputes
- **Tier 3 (Human decide)**: dropping backward compatibility for `resume`, changing `--parallel` input format, introducing detached daemon mode

---

## MVP Scope & Delivery

### Must Have (MVP)
- Remove wrapper timeout behavior from runtime and docs.
- Introduce thin CLI to application boundary.
- Unify single-task and parallel-task execution under one async orchestration model.
- Preserve `resume`, `--parallel`, and `cleanup` compatibility.
- Keep logs after task completion and expose log paths in results.
- Keep explicit historical log cleanup command behavior.

### Nice to Have (Later)
- Add file-backed task registry for detached/background execution.
- Add `status` / `watch` commands.
- Add retention-policy cleanup by age or size.

---

## Architecture Decision Records

### ADR-001: Do not introduce detached daemon mode in this refactor
- **Context**: The user wants async execution, channel-driven completion, persistent logs, and easier debugging.
- **Options**: foreground supervisor only; detached daemon with task registry; external queue/service.
- **Decision**: Keep execution inside the invoking process and make orchestration async internally.
- **Rationale**: This satisfies async coordination without adding daemon lifecycle, IPC, or persistent registry complexity.
- **Consequences**: CLI still blocks until requested work completes; future detached mode remains possible as a follow-up.

### ADR-002: Preserve logs until explicit or stale cleanup
- **Context**: Immediate log deletion prevents failure debugging.
- **Options**: delete on completion; retain until explicit cleanup; retain with age-based janitor only.
- **Decision**: Retain logs after task completion and preserve `cleanup` / `--cleanup` for historical log removal.
- **Rationale**: This matches the stated debugging requirement and is already close to current behavior.
- **Consequences**: Temp storage usage can grow; retention policy may be added later.

### ADR-003: Default parallel worker limit is 10
- **Context**: Unlimited concurrency risks noisy backend contention and poor local-machine behavior.
- **Options**: unlimited; fixed 10; configurable with default 10.
- **Decision**: configurable with default 10.
- **Rationale**: Provides safe default behavior while preserving operator control.
- **Consequences**: Existing tests and docs that assume unlimited default must be updated.

---

## Risks & Dependencies

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Timeout removal exposes hidden wait deadlocks | M | H | Add acceptance tests around completion without timeout and inspect parser/process wait handoff |
| CLI compatibility drifts during package moves | M | H | Lock behavior with acceptance tests before moving code |
| Parallel summary timing changes break expected output | M | M | Define completion-order vs final-order rules in tests before refactor |
| Logger close/flush can block indefinitely after timeout removal | M | H | Refactor logger lifecycle separately and add tests around close/flush completion |
| Cleanup deletes wrong files | L | H | Preserve symlink, PID liveness, and PID reuse checks with regression tests |

---

## Handoff Notes

- **For task-decomposition**: Break work into slices: acceptance coverage, timeout removal, application extraction, parallel aggregation, log cleanup isolation, documentation cleanup.
- **For execution agents**: Do not start with package moves. Start with tests that lock current CLI compatibility and target async timeout-free behavior.
- **For review agents**: Focus on behavior regressions around `resume`, `--parallel`, log retention, and cancellation semantics.
- **Open decisions**: Detached daemon mode, task registry, and retention-policy cleanup are deliberately deferred.
- **Verification path**: Fastest proof is targeted `go test` for execution, cleanup, and CLI command behavior, followed by `go test ./...`.

---

*This AEC was created through interactive requirements gathering and is optimized for autonomous multi-agent consumption, execution, and verification.*
