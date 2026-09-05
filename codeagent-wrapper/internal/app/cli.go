package wrapper

import (
	"context"
	"fmt"
	"os"
	"strings"

	adaptercli "codeagent-wrapper/internal/adapter/cli"
	runtask "codeagent-wrapper/internal/application/runtask"
	runtaskset "codeagent-wrapper/internal/application/runtaskset"
	config "codeagent-wrapper/internal/config"
	domaintask "codeagent-wrapper/internal/domain/task"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type exitError = adaptercli.ExitError
type cliOptions = adaptercli.Options

func Main() {
	Run()
}

// Run is the program entrypoint for cmd/codeagent/main.go.
func Run() {
	exitFn(run())
}

func run() int {
	return adaptercli.Run(os.Args[1:], NewRootDeps())
}

func newRootCommand() *cobra.Command {
	return adaptercli.NewRootCommand(NewRootDeps())
}

func NewRootDeps() adaptercli.RootDeps {
	return adaptercli.RootDeps{
		Name:    currentWrapperName,
		Version: func() string { return version },
		Cleanup: runCleanupMode,
		Execute: executeCLI,
	}
}

func executeCLI(cmd *cobra.Command, args []string, opts *cliOptions, name string) error {
	exitCode := runWithLoggerAndCleanup(func() int {
		return adaptercli.ExecuteMode(cmd, args, os.Args[1:], opts, name, adaptercli.ExecuteDeps{
			NewViper:          config.NewViper,
			LogError:          logError,
			LogInfo:           logInfo,
			BuildSingleConfig: buildSingleConfig,
			RunParallel:       runParallelMode,
			RunSingle:         runSingleMode,
		})
	})

	if exitCode == 0 {
		return nil
	}
	return exitError{Code: exitCode}
}

func runWithLoggerAndCleanup(fn func() int) (exitCode int) {
	ensureExecutableTempDir()
	logger, err := NewLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to initialize logger: %v\n", err)
		return 1
	}
	setLogger(logger)

	defer func() {
		logger := activeLogger()
		if logger != nil {
			logger.Flush()
		}
		if err := closeLogger(); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: failed to close logger: %v\n", err)
		}
		if logger == nil {
			return
		}

		if exitCode != 0 {
			if entries := logger.ExtractRecentErrors(10); len(entries) > 0 {
				fmt.Fprintln(os.Stderr, "\n=== Recent Errors ===")
				for _, entry := range entries {
					fmt.Fprintln(os.Stderr, entry)
				}
				fmt.Fprintf(os.Stderr, "Log file: %s\n", logger.Path())
			}
		}
	}()
	defer runCleanupHook()

	// Clean up stale logs from previous runs.
	scheduleStartupCleanup()

	return fn()
}

func parseArgs() (*config.Config, error) {
	return adaptercli.ParseSingleConfig(os.Args[1:])
}

func buildSingleConfig(cmd *cobra.Command, args []string, rawArgv []string, opts *cliOptions, v *viper.Viper) (*config.Config, error) {
	return adaptercli.BuildSingleConfig(cmd, args, rawArgv, opts, v)
}

func runParallelMode(cmd *cobra.Command, args []string, opts *cliOptions, v *viper.Viper, name string) int {
	return adaptercli.ExecuteParallelMode(cmd, args, opts, v, name, adaptercli.ParallelDeps[runtaskset.Plan]{
		BuildPlan: func(cmd *cobra.Command, opts *cliOptions, v *viper.Viper) (runtaskset.Plan, error) {
			input, err := adaptercli.BuildParallelInput(cmd, opts, v, adaptercli.ParallelInputDeps{
				StdinReader:        stdinReader,
				DefaultBackendName: defaultBackendName,
				ResolveMaxWorkers:  resolveConfiguredMaxParallelWorkers,
			})
			if err != nil {
				return runtaskset.Plan{}, err
			}
			return runtaskset.BuildPlan(input, runtaskset.BuildDeps{
				ResolveBackendName: func(name string) (string, error) {
					backend, err := selectBackendFn(name)
					if err != nil {
						return "", err
					}
					return backend.Name(), nil
				},
				ParseConfig:     parseParallelConfig,
				TopologicalSort: topologicalSort,
			})
		},
		RunPlan: func(plan runtaskset.Plan) (string, int, error) {
			return runtaskset.RunPlan(context.Background(), plan, runtaskset.RunDeps{
				ExecuteConcurrent: func(parentCtx context.Context, layers [][]runtask.Spec, maxWorkers int) []domaintask.TaskResult {
					return executeConcurrentWithContext(parentCtx, layers, noExecutionTimeout, maxWorkers)
				},
				EnrichResults:             enrichTaskResults,
				RenderFinalOutputWithMode: generateFinalOutputWithMode,
			})
		},
	})
}

func runSingleMode(cfg *config.Config, name string) int {
	_, err := configureSingleModeBackend(cfg)
	if err != nil {
		logError(err.Error())
		return 1
	}

	plan, err := runtask.PrepareConfigPlan(cfg, runtask.PrepareDeps{
		ResolveTaskText:      resolveSingleTaskText,
		ApplyPromptAndSkills: applySingleTaskPromptAndSkills,
		ShouldUseStdin:       shouldUseStdin,
		BuildCommandArgs:     buildSingleTaskCommandArgs,
	})
	if err != nil {
		logError(err.Error())
		return 1
	}

	logger, err := requireSingleModeLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}

	adaptercli.RenderSingleTaskBanner(os.Stderr, name, cfg.Backend, append([]string{codexCommand}, plan.Command...), os.Getpid(), logger.Path())

	if cfg.Mode == "new" && strings.TrimSpace(plan.TaskText) == "integration-log-check" {
		logInfo("Integration log check: skipping backend execution")
		return 0
	}

	logSingleTaskStdinReasons(plan)

	logInfo(fmt.Sprintf("%s running...", cfg.Backend))

	result, err := runtask.ExecutePlan(context.Background(), plan, runtask.ExecuteDeps{
		ExecuteTaskLayers: func(parentCtx context.Context, layers [][]runtask.Spec, maxWorkers int, runTask func(runtask.Spec, int) domaintask.TaskResult) []domaintask.TaskResult {
			return executeTaskLayersFn(parentCtx, layers, maxWorkers, func(task runtask.Spec, timeout int) domaintask.TaskResult {
				return runTask(task, timeout)
			})
		},
		RunTask: func(task runtask.Spec, timeout int) domaintask.TaskResult {
			return runTaskFn(task, false, timeout)
		},
		EnrichResults: enrichTaskResults,
	})
	if err != nil {
		logError(err.Error())
		return 1
	}

	output, err := adaptercli.EmitSingleTaskResult(os.Stdout, cfg.Backend, cfg.OutputPath, result)
	if err != nil {
		logError(err.Error())
		return 1
	}
	if output.ExitCode != 0 && strings.TrimSpace(result.Message) == "" && strings.TrimSpace(output.LogError) != "" {
		logError(output.LogError)
	}

	return output.ExitCode
}
