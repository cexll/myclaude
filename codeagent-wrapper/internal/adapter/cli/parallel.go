package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type ParallelDeps[Plan any] struct {
	BuildPlan func(cmd *cobra.Command, opts *Options, v *viper.Viper) (Plan, error)
	RunPlan   func(plan Plan) (string, int, error)
}

func ExecuteParallelMode[Plan any](cmd *cobra.Command, args []string, opts *Options, v *viper.Viper, name string, deps ParallelDeps[Plan]) int {
	if len(args) > 0 {
		WriteParallelModeArgumentError(stderrWriter(), name)
		return 1
	}

	if cmd.Flags().Changed("agent") || cmd.Flags().Changed("prompt-file") || cmd.Flags().Changed("reasoning-effort") || cmd.Flags().Changed("skills") {
		WriteParallelModeFlagError(stderrWriter())
		return 1
	}

	plan, err := deps.BuildPlan(cmd, opts, v)
	if err != nil {
		fmt.Fprintf(stderrWriter(), "ERROR: %v\n", err)
		return 1
	}

	stdout, exitCode, err := deps.RunPlan(plan)
	if err != nil {
		fmt.Fprintf(stderrWriter(), "ERROR: %v\n", err)
		return 1
	}
	return EmitParallelResult(stdoutWriter(), stdout, exitCode)
}

var stdoutWriter = func() io.Writer { return os.Stdout }
var stderrWriter = func() io.Writer { return os.Stderr }
