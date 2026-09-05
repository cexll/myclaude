package runtask

import (
	"context"
	"strings"
	"testing"

	domaintask "codeagent-wrapper/internal/domain/task"
)

func TestPreparePlanBuildsTaskSpecAndCommand(t *testing.T) {
	cmd := Command{Task: "body", Backend: "codex", WorkDir: ".", ExplicitStdin: false}

	plan, err := PreparePlan(cmd, PrepareDeps{
		ResolveTaskText: func(Command) (string, bool, error) { return "body", false, nil },
		ApplyPromptAndSkills: func(_ Command, task string) (string, error) {
			return task + "\nextra", nil
		},
		ShouldUseStdin:   func(task string, piped bool) bool { return strings.Contains(task, "\n") || piped },
		BuildCommandArgs: func(_ Command, target string) []string { return []string{"exec", target} },
	})
	if err != nil {
		t.Fatalf("PreparePlan() error = %v", err)
	}
	if !plan.UseStdin || plan.TargetArg != "-" {
		t.Fatalf("plan = %#v, want stdin target '-'", plan)
	}
	if plan.TaskSpec.Task != "body\nextra" {
		t.Fatalf("task = %q, want wrapped task", plan.TaskSpec.Task)
	}
	if len(plan.Command) != 2 || plan.Command[1] != "-" {
		t.Fatalf("command = %#v, want target '-'", plan.Command)
	}
}

func TestExecutePlanUsesSharedExecutor(t *testing.T) {
	plan := Plan{TaskSpec: Spec{Task: "body"}}
	var gotLayers [][]Spec
	var gotTask Spec

	result, err := ExecutePlan(context.Background(), plan, ExecuteDeps{
		ExecuteTaskLayers: func(_ context.Context, layers [][]Spec, maxWorkers int, runTask func(Spec, int) domaintask.TaskResult) []domaintask.TaskResult {
			gotLayers = layers
			return []domaintask.TaskResult{runTask(layers[0][0], 0)}
		},
		RunTask: func(task Spec, timeout int) domaintask.TaskResult {
			gotTask = task
			return domaintask.TaskResult{Message: "ok"}
		},
		EnrichResults: func([]domaintask.TaskResult) {},
	})
	if err != nil {
		t.Fatalf("ExecutePlan() error = %v", err)
	}
	if result.Message != "ok" || gotTask.Task != "body" || len(gotLayers) != 1 || len(gotLayers[0]) != 1 {
		t.Fatalf("unexpected result=%#v task=%#v layers=%#v", result, gotTask, gotLayers)
	}
}
