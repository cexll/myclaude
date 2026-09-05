package wrapper

import (
	"context"

	config "codeagent-wrapper/internal/config"
	domaintask "codeagent-wrapper/internal/domain/task"
	executor "codeagent-wrapper/internal/executor"
	backend "codeagent-wrapper/internal/infrastructure/backend"
)

// defaultRunCodexTaskFn is the default implementation of runCodexTaskFn (exposed for test reset).
func defaultRunCodexTaskFn(task executor.TaskSpec, timeout int) domaintask.TaskResult {
	return executor.DefaultRunCodexTaskFn(task, timeout)
}

var runCodexTaskFn = defaultRunCodexTaskFn

func topologicalSort(tasks []domaintask.TaskSpec) ([][]domaintask.TaskSpec, error) {
	return domaintask.TopologicalSort(tasks)
}

func executeConcurrent(layers [][]executor.TaskSpec, timeout int) []domaintask.TaskResult {
	maxWorkers := config.ResolveMaxParallelWorkers()
	return executeConcurrentWithContext(context.Background(), layers, timeout, maxWorkers)
}

func executeConcurrentWithContext(parentCtx context.Context, layers [][]executor.TaskSpec, timeout int, maxWorkers int) []domaintask.TaskResult {
	_ = timeout
	return executeTaskLayersFn(parentCtx, layers, maxWorkers, runCodexTaskFn)
}

func generateFinalOutput(results []domaintask.TaskResult) string {
	return executor.GenerateFinalOutput(results)
}

func generateFinalOutputWithMode(results []domaintask.TaskResult, summaryOnly bool) string {
	return executor.GenerateFinalOutputWithMode(results, summaryOnly)
}

func buildCodexArgs(cfg *config.Config, targetArg string) []string {
	return backend.BuildCodexArgs(cfg, targetArg)
}

func runCodexTask(taskSpec executor.TaskSpec, silent bool, timeoutSec int) domaintask.TaskResult {
	return runCodexTaskWithContext(context.Background(), taskSpec, nil, nil, false, silent, timeoutSec)
}

func runCodexProcess(parentCtx context.Context, codexArgs []string, taskText string, useStdin bool, timeoutSec int) (message, threadID string, exitCode int) {
	res := runCodexTaskWithContext(parentCtx, executor.TaskSpec{Task: taskText, WorkDir: defaultWorkdir, Mode: "new", UseStdin: useStdin}, nil, codexArgs, true, false, timeoutSec)
	return res.Message, res.SessionID, res.ExitCode
}

func runCodexTaskWithContext(parentCtx context.Context, taskSpec executor.TaskSpec, backend backend.Backend, customArgs []string, useCustomArgs bool, silent bool, timeoutSec int) domaintask.TaskResult {
	return executor.RunCodexTaskWithContext(parentCtx, taskSpec, backend, codexCommand, buildCodexArgsFn, customArgs, useCustomArgs, silent, timeoutSec)
}

func detectProjectSkills(workDir string) []string {
	return executor.DetectProjectSkills(workDir)
}

func resolveSkillContent(skills []string, maxBudget int) string {
	return executor.ResolveSkillContent(skills, maxBudget)
}
