package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type ExitError struct {
	Code int
}

func (e ExitError) Error() string {
	return fmt.Sprintf("exit %d", e.Code)
}

type Options struct {
	Backend         string
	Model           string
	ReasoningEffort string
	Agent           string
	PromptFile      string
	Output          string
	Skills          string
	SkipPermissions bool
	Worktree        bool

	Parallel   bool
	FullOutput bool

	Cleanup    bool
	Version    bool
	ConfigFile string
}

type RootDeps struct {
	Name    func() string
	Version func() string
	Cleanup func() int
	Execute func(cmd *cobra.Command, args []string, opts *Options, name string) error
}

func Run(args []string, deps RootDeps) int {
	cmd := NewRootCommand(deps)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		var ee ExitError
		if errors.As(err, &ee) {
			return ee.Code
		}
		return 1
	}
	return 0
}

func NewRootCommand(deps RootDeps) *cobra.Command {
	name := deps.Name()
	opts := &Options{}

	cmd := &cobra.Command{
		Use:           fmt.Sprintf("%s [flags] <task>|resume <session_id> <task> [workdir]", name),
		Short:         "Go wrapper for AI CLI backends",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.Version {
				fmt.Printf("%s version %s\n", name, deps.Version())
				return nil
			}
			if opts.Cleanup {
				code := deps.Cleanup()
				if code == 0 {
					return nil
				}
				return ExitError{Code: code}
			}
			return deps.Execute(cmd, args, opts, name)
		},
	}
	cmd.CompletionOptions.DisableDefaultCmd = true

	AddRootFlags(cmd.Flags(), opts)
	cmd.AddCommand(newVersionCommand(name, deps.Version), newCleanupCommand(deps.Cleanup))

	return cmd
}

func AddRootFlags(fs *pflag.FlagSet, opts *Options) {
	fs.StringVar(&opts.ConfigFile, "config", "", "Config file path (default: $HOME/.codeagent/config.*)")
	fs.BoolVarP(&opts.Version, "version", "v", false, "Print version and exit")
	fs.BoolVar(&opts.Cleanup, "cleanup", false, "Clean up old logs and exit")

	fs.BoolVar(&opts.Parallel, "parallel", false, "Run tasks in parallel (config from stdin)")
	fs.BoolVar(&opts.FullOutput, "full-output", false, "Parallel mode: include full task output (legacy)")

	fs.StringVar(&opts.Backend, "backend", "codex", "Backend to use (codex, claude, gemini, opencode, minimax, minimax-anthropic-cn, minimax-openai-global, minimax-openai-cn)")
	fs.StringVar(&opts.Model, "model", "", "Model override")
	fs.StringVar(&opts.ReasoningEffort, "reasoning-effort", "", "Reasoning effort (backend-specific)")
	fs.StringVar(&opts.Agent, "agent", "", "Agent preset name (from ~/.codeagent/models.json)")
	fs.StringVar(&opts.PromptFile, "prompt-file", "", "Prompt file path")
	fs.StringVar(&opts.Output, "output", "", "Write structured JSON output to file")
	fs.StringVar(&opts.Skills, "skills", "", "Comma-separated skill names for spec injection")

	fs.BoolVar(&opts.SkipPermissions, "skip-permissions", false, "Skip permissions prompts (also via CODEAGENT_SKIP_PERMISSIONS)")
	fs.BoolVar(&opts.SkipPermissions, "dangerously-skip-permissions", false, "Alias for --skip-permissions")
	fs.BoolVar(&opts.Worktree, "worktree", false, "Execute in a new git worktree (auto-generates task ID)")
}

func newVersionCommand(name string, version func() string) *cobra.Command {
	return &cobra.Command{
		Use:           "version",
		Short:         "Print version and exit",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("%s version %s\n", name, version())
			return nil
		},
	}
}

func newCleanupCommand(cleanup func() int) *cobra.Command {
	return &cobra.Command{
		Use:           "cleanup",
		Short:         "Clean up old logs and exit",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			code := cleanup()
			if code == 0 {
				return nil
			}
			return ExitError{Code: code}
		},
	}
}
