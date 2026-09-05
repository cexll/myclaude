package cli

import (
	"fmt"
	"strings"

	config "codeagent-wrapper/internal/config"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const defaultWorkdir = "."

func ParseSingleConfig(rawArgv []string) (*config.Config, error) {
	opts := &Options{}
	cmd := &cobra.Command{SilenceErrors: true, SilenceUsage: true, Args: cobra.ArbitraryArgs}
	AddRootFlags(cmd.Flags(), opts)

	if err := cmd.ParseFlags(rawArgv); err != nil {
		return nil, err
	}
	args := cmd.Flags().Args()

	v, err := config.NewViper(opts.ConfigFile)
	if err != nil {
		return nil, err
	}

	return BuildSingleConfig(cmd, args, rawArgv, opts, v)
}

func BuildSingleConfig(cmd *cobra.Command, args []string, rawArgv []string, opts *Options, v *viper.Viper) (*config.Config, error) {
	backendName := "codex"
	model := ""
	reasoningEffort := ""
	agentName := ""
	promptFile := ""
	promptFileExplicit := false
	outputPath := ""
	yolo := false

	if cmd.Flags().Changed("agent") {
		agentName = strings.TrimSpace(opts.Agent)
		if agentName == "" {
			return nil, fmt.Errorf("--agent flag requires a value")
		}
		if err := config.ValidateAgentName(agentName); err != nil {
			return nil, fmt.Errorf("--agent flag invalid value: %w", err)
		}
	} else {
		agentName = strings.TrimSpace(v.GetString("agent"))
		if agentName != "" {
			if err := config.ValidateAgentName(agentName); err != nil {
				return nil, fmt.Errorf("--agent flag invalid value: %w", err)
			}
		}
	}

	var resolvedBackend, resolvedModel, resolvedPromptFile, resolvedReasoning string
	var resolvedAllowedTools, resolvedDisallowedTools []string
	if agentName != "" {
		var resolvedYolo bool
		var err error
		resolvedBackend, resolvedModel, resolvedPromptFile, resolvedReasoning, _, _, resolvedYolo, resolvedAllowedTools, resolvedDisallowedTools, err = config.ResolveAgentConfig(agentName)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve agent %q: %w", agentName, err)
		}
		yolo = resolvedYolo
	}

	if cmd.Flags().Changed("prompt-file") {
		promptFile = strings.TrimSpace(opts.PromptFile)
		if promptFile == "" {
			return nil, fmt.Errorf("--prompt-file flag requires a value")
		}
		promptFileExplicit = true
	} else if val := strings.TrimSpace(v.GetString("prompt-file")); val != "" {
		promptFile = val
		promptFileExplicit = true
	} else {
		promptFile = resolvedPromptFile
	}

	if cmd.Flags().Changed("output") {
		outputPath = strings.TrimSpace(opts.Output)
		if outputPath == "" {
			return nil, fmt.Errorf("--output flag requires a value")
		}
	} else if val := strings.TrimSpace(v.GetString("output")); val != "" {
		outputPath = val
	}

	agentFlagChanged := cmd.Flags().Changed("agent")
	backendFlagChanged := cmd.Flags().Changed("backend")
	if backendFlagChanged {
		backendName = strings.TrimSpace(opts.Backend)
		if backendName == "" {
			return nil, fmt.Errorf("--backend flag requires a value")
		}
	}

	switch {
	case agentFlagChanged && backendFlagChanged && LastFlagIndex(rawArgv, "agent") > LastFlagIndex(rawArgv, "backend"):
		backendName = resolvedBackend
	case !backendFlagChanged && agentName != "":
		backendName = resolvedBackend
	case !backendFlagChanged:
		if val := strings.TrimSpace(v.GetString("backend")); val != "" {
			backendName = val
		}
	}

	modelFlagChanged := cmd.Flags().Changed("model")
	if modelFlagChanged {
		model = strings.TrimSpace(opts.Model)
		if model == "" {
			return nil, fmt.Errorf("--model flag requires a value")
		}
	}

	switch {
	case agentFlagChanged && modelFlagChanged && LastFlagIndex(rawArgv, "agent") > LastFlagIndex(rawArgv, "model"):
		model = strings.TrimSpace(resolvedModel)
	case !modelFlagChanged && agentName != "":
		model = strings.TrimSpace(resolvedModel)
	case !modelFlagChanged:
		model = strings.TrimSpace(v.GetString("model"))
	}

	if cmd.Flags().Changed("reasoning-effort") {
		reasoningEffort = strings.TrimSpace(opts.ReasoningEffort)
		if reasoningEffort == "" {
			return nil, fmt.Errorf("--reasoning-effort flag requires a value")
		}
	} else if val := strings.TrimSpace(v.GetString("reasoning-effort")); val != "" {
		reasoningEffort = val
	} else if agentName != "" {
		reasoningEffort = strings.TrimSpace(resolvedReasoning)
	}

	skipChanged := cmd.Flags().Changed("skip-permissions") || cmd.Flags().Changed("dangerously-skip-permissions")
	skipPermissions := false
	if skipChanged {
		skipPermissions = opts.SkipPermissions
	} else {
		skipPermissions = v.GetBool("skip-permissions")
	}

	if len(args) == 0 {
		return nil, fmt.Errorf("task required")
	}

	var skills []string
	if cmd.Flags().Changed("skills") {
		for _, s := range strings.Split(opts.Skills, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				skills = append(skills, s)
			}
		}
	}

	cfg := &config.Config{
		WorkDir:            defaultWorkdir,
		Backend:            backendName,
		Agent:              agentName,
		PromptFile:         promptFile,
		PromptFileExplicit: promptFileExplicit,
		OutputPath:         outputPath,
		SkipPermissions:    skipPermissions,
		Yolo:               yolo,
		Model:              model,
		ReasoningEffort:    reasoningEffort,
		MaxParallelWorkers: config.ResolveMaxParallelWorkers(),
		AllowedTools:       resolvedAllowedTools,
		DisallowedTools:    resolvedDisallowedTools,
		Skills:             skills,
		Worktree:           opts.Worktree,
	}

	if args[0] == "resume" {
		if len(args) < 3 {
			return nil, fmt.Errorf("resume mode requires: resume <session_id> <task>")
		}
		cfg.Mode = "resume"
		cfg.SessionID = strings.TrimSpace(args[1])
		if cfg.SessionID == "" {
			return nil, fmt.Errorf("resume mode requires non-empty session_id")
		}
		cfg.Task = args[2]
		cfg.ExplicitStdin = args[2] == "-"
		if len(args) > 3 {
			if args[3] == "-" {
				return nil, fmt.Errorf("invalid workdir: '-' is not a valid directory path")
			}
			cfg.WorkDir = args[3]
		}
		return cfg, nil
	}

	cfg.Mode = "new"
	cfg.Task = args[0]
	cfg.ExplicitStdin = args[0] == "-"
	if len(args) > 1 {
		if args[1] == "-" {
			return nil, fmt.Errorf("invalid workdir: '-' is not a valid directory path")
		}
		cfg.WorkDir = args[1]
	}
	return cfg, nil
}

func LastFlagIndex(argv []string, name string) int {
	if len(argv) == 0 {
		return -1
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return -1
	}

	needle := "--" + name
	prefix := needle + "="
	last := -1
	for i, arg := range argv {
		if arg == needle || strings.HasPrefix(arg, prefix) {
			last = i
		}
	}
	return last
}
