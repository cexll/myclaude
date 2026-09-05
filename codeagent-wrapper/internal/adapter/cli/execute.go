package cli

import (
	"fmt"

	config "codeagent-wrapper/internal/config"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type ExecuteDeps struct {
	NewViper          func(string) (*viper.Viper, error)
	LogError          func(string)
	LogInfo           func(string)
	BuildSingleConfig func(cmd *cobra.Command, args []string, rawArgv []string, opts *Options, v *viper.Viper) (*config.Config, error)
	RunParallel       func(cmd *cobra.Command, args []string, opts *Options, v *viper.Viper, name string) int
	RunSingle         func(cfg *config.Config, name string) int
}

func ExecuteMode(cmd *cobra.Command, args []string, rawArgv []string, opts *Options, name string, deps ExecuteDeps) int {
	v, err := deps.NewViper(opts.ConfigFile)
	if err != nil {
		deps.LogError(err.Error())
		return 1
	}

	if opts.Parallel {
		return deps.RunParallel(cmd, args, opts, v, name)
	}

	deps.LogInfo("Script started")

	cfg, err := deps.BuildSingleConfig(cmd, args, rawArgv, opts, v)
	if err != nil {
		deps.LogError(err.Error())
		return 1
	}
	deps.LogInfo(fmt.Sprintf("Parsed args: mode=%s, task_len=%d, backend=%s", cfg.Mode, len(cfg.Task), cfg.Backend))
	return deps.RunSingle(cfg, name)
}
