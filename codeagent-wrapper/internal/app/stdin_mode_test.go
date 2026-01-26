package wrapper

import (
	"strings"
	"testing"
)

func TestRunSingleMode_UseStdin_TargetArgAndTaskText(t *testing.T) {
	defer resetTestHooks()

	t.Setenv("TMPDIR", t.TempDir())
	logger, err := NewLogger()
	if err != nil {
		t.Fatalf("NewLogger(): %v", err)
	}
	setLogger(logger)
	t.Cleanup(func() { _ = closeLogger() })

	type testCase struct {
		name       string
		cfgTask    string
		explicit   bool
		stdinData  string
		isTerminal bool

		wantUseStdin bool
		wantTarget   string
		wantTaskText string
	}

	longTask := strings.Repeat("a", 801)

	tests := []testCase{
		{
			name:         "piped input forces stdin mode",
			cfgTask:      "cli-task",
			stdinData:    "piped task text",
			isTerminal:   false,
			wantUseStdin: true,
			wantTarget:   "-",
			wantTaskText: "piped task text",
		},
		{
			name:         "explicit dash forces stdin mode",
			cfgTask:      "-",
			explicit:     true,
			stdinData:    "explicit task text",
			isTerminal:   true,
			wantUseStdin: true,
			wantTarget:   "-",
			wantTaskText: "explicit task text",
		},
		{
			name:         "special char backslash forces stdin mode",
			cfgTask:      `C:\repo\file.go`,
			isTerminal:   true,
			wantUseStdin: true,
			wantTarget:   "-",
			wantTaskText: `C:\repo\file.go`,
		},
		{
			name:         "length>800 forces stdin mode",
			cfgTask:      longTask,
			isTerminal:   true,
			wantUseStdin: true,
			wantTarget:   "-",
			wantTaskText: longTask,
		},
		{
			name:         "simple task uses argv target",
			cfgTask:      "analyze code",
			isTerminal:   true,
			wantUseStdin: false,
			wantTarget:   "analyze code",
			wantTaskText: "analyze code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotTarget string
			buildCodexArgsFn = func(cfg *Config, targetArg string) []string {
				gotTarget = targetArg
				return []string{targetArg}
			}

			var gotTask TaskSpec
			runTaskFn = func(task TaskSpec, silent bool, timeout int) TaskResult {
				gotTask = task
				return TaskResult{ExitCode: 0, Message: "ok"}
			}

			stdinReader = strings.NewReader(tt.stdinData)
			isTerminalFn = func() bool { return tt.isTerminal }

			cfg := &Config{
				Mode:          "new",
				Task:          tt.cfgTask,
				WorkDir:       defaultWorkdir,
				Backend:       defaultBackendName,
				ExplicitStdin: tt.explicit,
			}

			if code := runSingleMode(cfg, "codeagent-wrapper"); code != 0 {
				t.Fatalf("runSingleMode() = %d, want 0", code)
			}

			if gotTarget != tt.wantTarget {
				t.Fatalf("targetArg = %q, want %q", gotTarget, tt.wantTarget)
			}
			if gotTask.UseStdin != tt.wantUseStdin {
				t.Fatalf("taskSpec.UseStdin = %v, want %v", gotTask.UseStdin, tt.wantUseStdin)
			}
			if gotTask.Task != tt.wantTaskText {
				t.Fatalf("taskSpec.Task = %q, want %q", gotTask.Task, tt.wantTaskText)
			}
		})
	}
}
