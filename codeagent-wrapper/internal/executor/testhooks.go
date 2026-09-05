package executor

import (
	"context"
	"io"
	"os/exec"
	"time"

	backend "codeagent-wrapper/internal/backend"
)

type CommandRunner = commandRunner
type ProcessHandle = processHandle

func SetForceKillDelay(seconds int32) (restore func()) {
	prev := forceKillDelay.Load()
	forceKillDelay.Store(seconds)
	return func() { forceKillDelay.Store(prev) }
}

func SetSelectBackendFn(fn func(string) (Backend, error)) (restore func()) {
	prev := selectBackendFn
	if fn != nil {
		selectBackendFn = fn
	} else {
		selectBackendFn = backend.Select
	}
	return func() { selectBackendFn = prev }
}

func SetCommandContextFn(fn func(context.Context, string, ...string) *exec.Cmd) (restore func()) {
	prev := commandContext
	if fn != nil {
		commandContext = fn
	} else {
		commandContext = exec.CommandContext
	}
	return func() { commandContext = prev }
}

func SetNewCommandRunner(fn func(context.Context, string, ...string) CommandRunner) (restore func()) {
	prev := newCommandRunner
	if fn != nil {
		newCommandRunner = fn
	} else {
		newCommandRunner = func(ctx context.Context, name string, args ...string) commandRunner {
			return &realCmd{cmd: commandContext(ctx, name, args...)}
		}
	}
	return func() { newCommandRunner = prev }
}

func SetProgressOutput(w io.Writer) (restore func()) {
	prev := progressOutput
	progressOutput = func() io.Writer { return w }
	return func() { progressOutput = prev }
}

func SetProgressHeartbeatInterval(d time.Duration) (restore func()) {
	prev := progressHeartbeatInterval.Load()
	progressHeartbeatInterval.Store(int64(d))
	return func() { progressHeartbeatInterval.Store(prev) }
}

func WithTaskLogger(ctx context.Context, logger *Logger) context.Context {
	return withTaskLogger(ctx, logger)
}

func TaskLoggerFromContext(ctx context.Context) *Logger {
	return taskLoggerFromContext(ctx)
}
