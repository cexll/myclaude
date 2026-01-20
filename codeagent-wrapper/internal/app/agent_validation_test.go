package wrapper

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	config "codeagent-wrapper/internal/config"
	executor "codeagent-wrapper/internal/executor"
)

func TestValidateAgentName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "simple", input: "develop", wantErr: false},
		{name: "upper", input: "ABC", wantErr: false},
		{name: "digits", input: "a1", wantErr: false},
		{name: "dash underscore", input: "a-b_c", wantErr: false},
		{name: "empty", input: "", wantErr: true},
		{name: "space", input: "a b", wantErr: true},
		{name: "slash", input: "a/b", wantErr: true},
		{name: "dotdot", input: "../evil", wantErr: true},
		{name: "unicode", input: "中文", wantErr: true},
		{name: "symbol", input: "a$b", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := config.ValidateAgentName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateAgentName(%q) err=%v, wantErr=%v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestParseArgs_InvalidAgentNameRejected(t *testing.T) {
	defer resetTestHooks()
	os.Args = []string{"codeagent-wrapper", "--agent", "../evil", "task"}
	if _, err := parseArgs(); err == nil {
		t.Fatalf("expected parseArgs to reject invalid agent name")
	}
}

func TestParseParallelConfig_InvalidAgentNameRejected(t *testing.T) {
	input := `---TASK---
id: task-1
agent: ../evil
---CONTENT---
do something`
	if _, err := parseParallelConfig([]byte(input)); err == nil {
		t.Fatalf("expected parseParallelConfig to reject invalid agent name")
	}
}

func TestParseParallelConfig_ResolvesAgentPromptFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Cleanup(config.ResetModelsConfigCacheForTest)
	config.ResetModelsConfigCacheForTest()

	configDir := filepath.Join(home, ".codeagent")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "models.json"), []byte(`{
  "default_backend": "codex",
  "default_model": "gpt-test",
  "agents": {
    "custom-agent": {
      "backend": "codex",
      "model": "gpt-test",
      "prompt_file": "~/.claude/prompt.md"
    }
  }
}`), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	input := `---TASK---
id: task-1
agent: custom-agent
---CONTENT---
do something`
	cfg, err := parseParallelConfig([]byte(input))
	if err != nil {
		t.Fatalf("parseParallelConfig() unexpected error: %v", err)
	}
	if len(cfg.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(cfg.Tasks))
	}
	if got := cfg.Tasks[0].PromptFile; got != "~/.claude/prompt.md" {
		t.Fatalf("PromptFile = %q, want %q", got, "~/.claude/prompt.md")
	}
}

func TestDefaultRunCodexTaskFn_AppliesAgentPromptFile(t *testing.T) {
	defer resetTestHooks()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "prompt.md"), []byte("P\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	fake := newFakeCmd(fakeCmdConfig{
		StdoutPlan: []fakeStdoutEvent{
			{Data: `{"type":"item.completed","item":{"type":"agent_message","text":"ok"}}` + "\n"},
		},
		WaitDelay: 2 * time.Millisecond,
	})

	_ = executor.SetNewCommandRunner(func(ctx context.Context, name string, args ...string) executor.CommandRunner { return fake })
	_ = executor.SetSelectBackendFn(func(name string) (Backend, error) {
		return testBackend{
			name:    name,
			command: "fake-cmd",
			argsFn: func(cfg *Config, targetArg string) []string {
				return []string{targetArg}
			},
		}, nil
	})

	res := defaultRunCodexTaskFn(TaskSpec{
		ID:         "t",
		Task:       "do",
		Backend:    "codex",
		PromptFile: "~/.claude/prompt.md",
	}, 5)
	if res.ExitCode != 0 {
		t.Fatalf("unexpected result: %+v", res)
	}

	want := "<agent-prompt>\nP\n</agent-prompt>\n\ndo"
	if got := fake.StdinContents(); got != want {
		t.Fatalf("stdin mismatch:\n got=%q\nwant=%q", got, want)
	}
}
