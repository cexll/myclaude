package cli

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestBuildParallelInputUsesCLIAndViperValues(t *testing.T) {
	opts := &Options{}
	cmd := &cobra.Command{}
	AddRootFlags(cmd.Flags(), opts)
	opts.Backend = "claude"
	opts.Model = "sonnet"
	opts.Output = "out.json"
	opts.FullOutput = true
	opts.SkipPermissions = true
	cmd.Flags().Lookup("backend").Changed = true
	cmd.Flags().Lookup("model").Changed = true
	cmd.Flags().Lookup("output").Changed = true
	cmd.Flags().Lookup("full-output").Changed = true
	cmd.Flags().Lookup("skip-permissions").Changed = true

	v := viper.New()
	got, err := BuildParallelInput(cmd, opts, v, ParallelInputDeps{
		StdinReader:        bytes.NewBufferString("task stream"),
		DefaultBackendName: "codex",
		ResolveMaxWorkers:  func(*viper.Viper) int { return 7 },
	})
	if err != nil {
		t.Fatalf("BuildParallelInput() error = %v", err)
	}

	if got.BackendName != "claude" || got.Model != "sonnet" || got.OutputPath != "out.json" || got.SummaryOnly || !got.SkipPermissions || got.MaxWorkers != 7 || string(got.StdinData) != "task stream" {
		t.Fatalf("BuildParallelInput() = %#v", got)
	}
}

func TestBuildParallelInputFallsBackToViper(t *testing.T) {
	opts := &Options{}
	cmd := &cobra.Command{}
	AddRootFlags(cmd.Flags(), opts)

	v := viper.New()
	v.Set("backend", "gemini")
	v.Set("model", "pro")
	v.Set("output", "parallel.json")
	v.Set("full-output", true)
	v.Set("skip-permissions", true)

	got, err := BuildParallelInput(cmd, opts, v, ParallelInputDeps{
		StdinReader:        bytes.NewBuffer(nil),
		DefaultBackendName: "codex",
		ResolveMaxWorkers:  func(*viper.Viper) int { return 10 },
	})
	if err != nil {
		t.Fatalf("BuildParallelInput() error = %v", err)
	}

	if got.BackendName != "gemini" || got.Model != "pro" || got.OutputPath != "parallel.json" || got.SummaryOnly || !got.SkipPermissions || got.MaxWorkers != 10 {
		t.Fatalf("BuildParallelInput() = %#v", got)
	}
}
