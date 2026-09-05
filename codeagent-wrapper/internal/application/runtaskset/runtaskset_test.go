package runtaskset

import (
	"context"
	"testing"

	appruntask "codeagent-wrapper/internal/application/runtask"
	domaintask "codeagent-wrapper/internal/domain/task"
)

func TestBuildPlanAppliesGlobalDefaults(t *testing.T) {
	plan, err := BuildPlan(BuildInput{
		BackendName:     "codex",
		Model:           "gpt-5",
		OutputPath:      "out.json",
		SummaryOnly:     true,
		SkipPermissions: true,
		StdinData:       []byte("ignored"),
		MaxWorkers:      10,
	}, BuildDeps{
		ResolveBackendName: func(name string) (string, error) { return name, nil },
		ParseConfig: func([]byte) (*ParallelConfig, error) {
			return &ParallelConfig{Tasks: []appruntask.Spec{{ID: "a", Task: "body"}}}, nil
		},
		TopologicalSort: func(tasks []domaintask.TaskSpec) ([][]domaintask.TaskSpec, error) {
			return [][]domaintask.TaskSpec{tasks}, nil
		},
	})
	if err != nil {
		t.Fatalf("BuildPlan() error = %v", err)
	}
	if plan.MaxWorkers != 10 || len(plan.Layers) != 1 || plan.Layers[0][0].Backend != "codex" || plan.Layers[0][0].Model != "gpt-5" || !plan.Layers[0][0].SkipPermissions {
		t.Fatalf("plan = %#v", plan)
	}
}

func TestRunPlanWritesAndRendersResults(t *testing.T) {
	out, exitCode, err := RunPlan(context.Background(), Plan{
		SummaryOnly: true,
		Layers:      [][]appruntask.Spec{{{ID: "a"}}},
		MaxWorkers:  2,
	}, RunDeps{
		ExecuteConcurrent: func(context.Context, [][]appruntask.Spec, int) []domaintask.TaskResult {
			return []domaintask.TaskResult{{TaskID: "a", ExitCode: 0, Message: "ok"}}
		},
		EnrichResults: func([]domaintask.TaskResult) {},
		RenderFinalOutputWithMode: func(results []domaintask.TaskResult, summaryOnly bool) string {
			if !summaryOnly || len(results) != 1 || results[0].TaskID != "a" {
				t.Fatalf("unexpected render args: %#v summaryOnly=%v", results, summaryOnly)
			}
			return "summary"
		},
	})
	if err != nil {
		t.Fatalf("RunPlan() error = %v", err)
	}
	if out != "summary" || exitCode != 0 {
		t.Fatalf("RunPlan() = (%q, %d), want (%q, 0)", out, exitCode, "summary")
	}
}
