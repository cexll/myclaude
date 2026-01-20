package wrapper

import (
	"context"

	backend "codeagent-wrapper/internal/backend"
	config "codeagent-wrapper/internal/config"
	executor "codeagent-wrapper/internal/executor"
)

// defaultRunCodexTaskFn is the default implementation of runCodexTaskFn (exposed for test reset).
func defaultRunCodexTaskFn(task TaskSpec, timeout int) TaskResult {
	return executor.DefaultRunCodexTaskFn(task, timeout)
}

var runCodexTaskFn = defaultRunCodexTaskFn

func topologicalSort(tasks []TaskSpec) ([][]TaskSpec, error) {
	return executor.TopologicalSort(tasks)
}

func executeConcurrent(layers [][]TaskSpec, timeout int) []TaskResult {
	maxWorkers := config.ResolveMaxParallelWorkers()
	return executeConcurrentWithContext(context.Background(), layers, timeout, maxWorkers)
}

func executeConcurrentWithContext(parentCtx context.Context, layers [][]TaskSpec, timeout int, maxWorkers int) []TaskResult {
	return executor.ExecuteConcurrentWithContext(parentCtx, layers, timeout, maxWorkers, runCodexTaskFn)
}

func generateFinalOutput(results []TaskResult) string {
	return executor.GenerateFinalOutput(results)
}

func generateFinalOutputWithMode(results []TaskResult, summaryOnly bool) string {
	return executor.GenerateFinalOutputWithMode(results, summaryOnly)
}

func buildCodexArgs(cfg *Config, targetArg string) []string {
	return backend.BuildCodexArgs(cfg, targetArg)
}

func runCodexTask(taskSpec TaskSpec, silent bool, timeoutSec int) TaskResult {
	return runCodexTaskWithContext(context.Background(), taskSpec, nil, nil, false, silent, timeoutSec)
}

func runCodexProcess(parentCtx context.Context, codexArgs []string, taskText string, useStdin bool, timeoutSec int) (message, threadID string, exitCode int) {
	res := runCodexTaskWithContext(parentCtx, TaskSpec{Task: taskText, WorkDir: defaultWorkdir, Mode: "new", UseStdin: useStdin}, nil, codexArgs, true, false, timeoutSec)
	return res.Message, res.SessionID, res.ExitCode
}

func runCodexTaskWithContext(parentCtx context.Context, taskSpec TaskSpec, backend Backend, customArgs []string, useCustomArgs bool, silent bool, timeoutSec int) TaskResult {
	return executor.RunCodexTaskWithContext(parentCtx, taskSpec, backend, codexCommand, buildCodexArgsFn, customArgs, useCustomArgs, silent, timeoutSec)
}
