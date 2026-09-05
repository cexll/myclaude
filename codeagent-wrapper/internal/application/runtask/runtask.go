package runtask

import (
	"context"
	"fmt"
	"strings"

	domaintask "codeagent-wrapper/internal/domain/task"
)

type Plan struct {
	TaskSpec  Spec
	TaskText  string
	TargetArg string
	Piped     bool
	UseStdin  bool
	Explicit  bool
	Command   []string
}

type Command struct {
	Task               string
	WorkDir            string
	Mode               string
	SessionID          string
	Backend            string
	Model              string
	ReasoningEffort    string
	Agent              string
	SkipPermissions    bool
	Worktree           bool
	AllowedTools       []string
	DisallowedTools    []string
	ExplicitStdin      bool
	PromptFile         string
	PromptFileExplicit bool
	Skills             []string
}

type PrepareDeps struct {
	ResolveTaskText      func(Command) (string, bool, error)
	ApplyPromptAndSkills func(Command, string) (string, error)
	ShouldUseStdin       func(string, bool) bool
	BuildCommandArgs     func(Command, string) []string
}

type ExecuteDeps struct {
	ExecuteTaskLayers func(context.Context, [][]Spec, int, func(Spec, int) domaintask.TaskResult) []domaintask.TaskResult
	RunTask           func(Spec, int) domaintask.TaskResult
	EnrichResults     func([]domaintask.TaskResult)
}

func PreparePlan(cmd Command, deps PrepareDeps) (Plan, error) {
	taskText, piped, err := deps.ResolveTaskText(cmd)
	if err != nil {
		return Plan{}, err
	}

	taskText, err = deps.ApplyPromptAndSkills(cmd, taskText)
	if err != nil {
		return Plan{}, err
	}

	useStdin := cmd.ExplicitStdin || deps.ShouldUseStdin(taskText, piped)
	targetArg := taskText
	if useStdin {
		targetArg = "-"
	}

	taskSpec := Spec{
		Task:            taskText,
		WorkDir:         cmd.WorkDir,
		Mode:            cmd.Mode,
		SessionID:       cmd.SessionID,
		Backend:         cmd.Backend,
		Model:           cmd.Model,
		ReasoningEffort: cmd.ReasoningEffort,
		Agent:           cmd.Agent,
		SkipPermissions: cmd.SkipPermissions,
		Worktree:        cmd.Worktree,
		AllowedTools:    cmd.AllowedTools,
		DisallowedTools: cmd.DisallowedTools,
		UseStdin:        useStdin,
	}

	return Plan{
		TaskSpec:  taskSpec,
		TaskText:  taskText,
		TargetArg: targetArg,
		Piped:     piped,
		UseStdin:  useStdin,
		Explicit:  cmd.ExplicitStdin,
		Command:   deps.BuildCommandArgs(cmd, targetArg),
	}, nil
}

func ExecutePlan(parentCtx context.Context, plan Plan, deps ExecuteDeps) (domaintask.TaskResult, error) {
	results := deps.ExecuteTaskLayers(parentCtx, [][]Spec{{plan.TaskSpec}}, 1, func(task Spec, timeout int) domaintask.TaskResult {
		return deps.RunTask(task, timeout)
	})
	if len(results) != 1 {
		return domaintask.TaskResult{}, fmt.Errorf("unexpected single-task result count: %d", len(results))
	}
	deps.EnrichResults(results)
	return results[0], nil
}

func StdinReasons(plan Plan) []string {
	if !plan.UseStdin {
		return nil
	}

	var reasons []string
	if plan.Piped {
		reasons = append(reasons, "piped input")
	}
	if plan.Explicit {
		reasons = append(reasons, "explicit \"-\"")
	}
	if strings.Contains(plan.TaskText, "\n") {
		reasons = append(reasons, "newline")
	}
	if strings.Contains(plan.TaskText, "\\") {
		reasons = append(reasons, "backslash")
	}
	if strings.Contains(plan.TaskText, "\"") {
		reasons = append(reasons, "double-quote")
	}
	if strings.Contains(plan.TaskText, "'") {
		reasons = append(reasons, "single-quote")
	}
	if strings.Contains(plan.TaskText, "`") {
		reasons = append(reasons, "backtick")
	}
	if strings.Contains(plan.TaskText, "$") {
		reasons = append(reasons, "dollar")
	}
	if len(plan.TaskText) > 800 {
		reasons = append(reasons, "length>800")
	}
	return reasons
}
