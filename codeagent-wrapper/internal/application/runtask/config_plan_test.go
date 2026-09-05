package runtask

import (
	"testing"

	config "codeagent-wrapper/internal/config"
)

func TestPrepareConfigPlanMapsConfigFields(t *testing.T) {
	cfg := &config.Config{
		Task:               "task body",
		WorkDir:            "/tmp/project",
		Mode:               "resume",
		SessionID:          "sid-1",
		Backend:            "claude",
		Model:              "sonnet",
		ReasoningEffort:    "high",
		Agent:              "develop",
		SkipPermissions:    true,
		Worktree:           true,
		AllowedTools:       []string{"a"},
		DisallowedTools:    []string{"b"},
		ExplicitStdin:      true,
		PromptFile:         "/tmp/prompt.md",
		PromptFileExplicit: true,
		Skills:             []string{"go"},
	}

	plan, err := PrepareConfigPlan(cfg, PrepareDeps{
		ResolveTaskText:      func(Command) (string, bool, error) { return "task body", true, nil },
		ApplyPromptAndSkills: func(_ Command, task string) (string, error) { return task, nil },
		ShouldUseStdin:       func(string, bool) bool { return true },
		BuildCommandArgs:     func(_ Command, target string) []string { return []string{"run", target} },
	})
	if err != nil {
		t.Fatalf("PrepareConfigPlan() error = %v", err)
	}

	if plan.TaskSpec.Task != "task body" || plan.TaskSpec.WorkDir != "/tmp/project" || plan.TaskSpec.Mode != "resume" || plan.TaskSpec.SessionID != "sid-1" || plan.TaskSpec.Backend != "claude" || !plan.TaskSpec.SkipPermissions || !plan.TaskSpec.Worktree || !plan.UseStdin || plan.TargetArg != "-" {
		t.Fatalf("PrepareConfigPlan() = %#v", plan)
	}
}
