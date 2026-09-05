package wrapper

import (
	"context"
	"fmt"
	"io"
	"strings"

	runtask "codeagent-wrapper/internal/application/runtask"
	runtaskset "codeagent-wrapper/internal/application/runtaskset"
	config "codeagent-wrapper/internal/config"
	domaintask "codeagent-wrapper/internal/domain/task"
	executor "codeagent-wrapper/internal/executor"

	"github.com/spf13/viper"
)

type taskRunner func(executor.TaskSpec, int) domaintask.TaskResult

func defaultExecuteTaskLayers(parentCtx context.Context, layers [][]executor.TaskSpec, maxWorkers int, runTask taskRunner) []domaintask.TaskResult {
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	if len(layers) == 1 && len(layers[0]) == 1 && strings.TrimSpace(layers[0][0].ID) == "" {
		task := layers[0][0]
		resultsCh := make(chan domaintask.TaskResult, 1)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					resultsCh <- runtaskset.PanicTaskResult(task.ID, r)
				}
			}()
			resultsCh <- runTask(task, noExecutionTimeout)
		}()
		select {
		case res := <-resultsCh:
			return []domaintask.TaskResult{res}
		case <-parentCtx.Done():
			return []domaintask.TaskResult{{TaskID: task.ID, ExitCode: 130, Error: "execution cancelled"}}
		}
	}
	return executor.ExecuteConcurrentWithContext(parentCtx, layers, noExecutionTimeout, maxWorkers, runTask)
}

var executeTaskLayersFn = defaultExecuteTaskLayers

func resolveSingleTaskText(cmd runtask.Command) (taskText string, piped bool, err error) {
	if cmd.ExplicitStdin {
		logInfo("Explicit stdin mode: reading task from stdin")
		data, err := io.ReadAll(stdinReader)
		if err != nil {
			return "", false, fmt.Errorf("failed to read stdin: %w", err)
		}
		taskText = string(data)
		if taskText == "" {
			return "", false, fmt.Errorf("explicit stdin mode requires task input from stdin")
		}
		return taskText, !isTerminal(), nil
	}

	pipedTask, err := readPipedTask()
	if err != nil {
		return "", false, fmt.Errorf("failed to read piped stdin: %w", err)
	}
	if pipedTask != "" {
		return pipedTask, true, nil
	}
	return cmd.Task, false, nil
}

func applySingleTaskPromptAndSkills(cmd runtask.Command, taskText string) (string, error) {
	if strings.TrimSpace(cmd.PromptFile) != "" {
		prompt, err := readAgentPromptFile(cmd.PromptFile, cmd.PromptFileExplicit)
		if err != nil {
			return "", fmt.Errorf("failed to read prompt file: %w", err)
		}
		taskText = wrapTaskWithAgentPrompt(prompt, taskText)
	}

	skills := cmd.Skills
	if len(skills) == 0 {
		skills = detectProjectSkills(cmd.WorkDir)
	}
	if len(skills) > 0 {
		if content := resolveSkillContent(skills, 0); content != "" {
			taskText += "\n\n# Domain Best Practices\n\n" + content
		}
	}
	return taskText, nil
}

func buildSingleTaskCommandArgs(cmd runtask.Command, targetArg string) []string {
	cfg := &config.Config{
		Backend:         cmd.Backend,
		Model:           cmd.Model,
		ReasoningEffort: cmd.ReasoningEffort,
		SkipPermissions: cmd.SkipPermissions,
	}
	return buildCodexArgsFn(cfg, targetArg)
}

func logSingleTaskStdinReasons(plan runtask.Plan) {
	reasons := runtask.StdinReasons(plan)
	if len(reasons) == 0 {
		return
	}
	logWarn(fmt.Sprintf("Using stdin mode for task due to: %s", strings.Join(reasons, ", ")))
}

func enrichTaskResults(results []domaintask.TaskResult) {
	for i := range results {
		results[i].CoverageTarget = defaultCoverageTarget
		if results[i].Message == "" {
			continue
		}

		lines := strings.Split(results[i].Message, "\n")
		results[i].Coverage = extractCoverageFromLines(lines)
		results[i].CoverageNum = extractCoverageNum(results[i].Coverage)
		results[i].FilesChanged = extractFilesChangedFromLines(lines)
		results[i].TestsPassed, results[i].TestsFailed = extractTestResultsFromLines(lines)
		results[i].KeyOutput = extractKeyOutputFromLines(lines, 0)
	}
}

func resolveConfiguredMaxParallelWorkers(v *viper.Viper) int {
	if v == nil || !v.IsSet("max-parallel-workers") {
		return config.ResolveMaxParallelWorkers()
	}
	return config.NormalizeMaxParallelWorkers(v.GetInt("max-parallel-workers"))
}
