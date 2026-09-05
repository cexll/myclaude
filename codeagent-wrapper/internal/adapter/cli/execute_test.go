package cli

import (
	"errors"
	"testing"

	config "codeagent-wrapper/internal/config"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestExecuteMode_ParallelBranch(t *testing.T) {
	opts := &Options{ConfigFile: "cfg", Parallel: true}
	cmd := &cobra.Command{}
	got := ExecuteMode(cmd, []string{"ignored"}, []string{"--parallel"}, opts, "wrapper", ExecuteDeps{
		NewViper: func(path string) (*viper.Viper, error) {
			if path != "cfg" {
				t.Fatalf("path = %q, want cfg", path)
			}
			return viper.New(), nil
		},
		LogError: func(string) {},
		LogInfo:  func(string) {},
		BuildSingleConfig: func(*cobra.Command, []string, []string, *Options, *viper.Viper) (*config.Config, error) {
			t.Fatal("BuildSingleConfig should not be called")
			return nil, nil
		},
		RunParallel: func(gotCmd *cobra.Command, gotArgs []string, gotOpts *Options, gotV *viper.Viper, gotName string) int {
			if gotCmd != cmd || gotOpts != opts || gotV == nil || gotName != "wrapper" || len(gotArgs) != 1 {
				t.Fatalf("unexpected parallel args")
			}
			return 7
		},
		RunSingle: func(*config.Config, string) int {
			t.Fatal("RunSingle should not be called")
			return 0
		},
	})
	if got != 7 {
		t.Fatalf("ExecuteMode() = %d, want 7", got)
	}
}

func TestExecuteMode_SingleBranch(t *testing.T) {
	opts := &Options{}
	cmd := &cobra.Command{}
	var infos []string
	got := ExecuteMode(cmd, []string{"task"}, []string{"task"}, opts, "wrapper", ExecuteDeps{
		NewViper: func(string) (*viper.Viper, error) { return viper.New(), nil },
		LogError: func(string) {},
		LogInfo:  func(msg string) { infos = append(infos, msg) },
		BuildSingleConfig: func(*cobra.Command, []string, []string, *Options, *viper.Viper) (*config.Config, error) {
			return &config.Config{Mode: "new", Task: "task", Backend: "codex"}, nil
		},
		RunParallel: func(*cobra.Command, []string, *Options, *viper.Viper, string) int {
			t.Fatal("RunParallel should not be called")
			return 0
		},
		RunSingle: func(cfg *config.Config, name string) int {
			if cfg.Task != "task" || name != "wrapper" {
				t.Fatalf("cfg=%+v name=%q", cfg, name)
			}
			return 3
		},
	})
	if got != 3 {
		t.Fatalf("ExecuteMode() = %d, want 3", got)
	}
	if len(infos) != 2 {
		t.Fatalf("LogInfo count = %d, want 2", len(infos))
	}
}

func TestExecuteMode_ConfigError(t *testing.T) {
	opts := &Options{ConfigFile: "bad"}
	var gotErr string
	got := ExecuteMode(&cobra.Command{}, nil, nil, opts, "wrapper", ExecuteDeps{
		NewViper: func(string) (*viper.Viper, error) { return nil, errors.New("boom") },
		LogError: func(msg string) { gotErr = msg },
		LogInfo:  func(string) {},
		BuildSingleConfig: func(*cobra.Command, []string, []string, *Options, *viper.Viper) (*config.Config, error) {
			return nil, nil
		},
		RunParallel: func(*cobra.Command, []string, *Options, *viper.Viper, string) int { return 0 },
		RunSingle:   func(*config.Config, string) int { return 0 },
	})
	if got != 1 || gotErr != "boom" {
		t.Fatalf("ExecuteMode() = %d, gotErr=%q", got, gotErr)
	}
}
