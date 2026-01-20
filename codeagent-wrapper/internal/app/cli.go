package wrapper

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	config "codeagent-wrapper/internal/config"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type exitError struct {
	code int
}

func (e exitError) Error() string {
	return fmt.Sprintf("exit %d", e.code)
}

type cliOptions struct {
	Backend         string
	Model           string
	ReasoningEffort string
	Agent           string
	PromptFile      string
	SkipPermissions bool

	Parallel   bool
	FullOutput bool

	Cleanup    bool
	Version    bool
	ConfigFile string
}

func Main() {
	Run()
}

// Run is the program entrypoint for cmd/codeagent/main.go.
func Run() {
	exitFn(run())
}

func run() int {
	cmd := newRootCommand()
	cmd.SetArgs(os.Args[1:])
	if err := cmd.Execute(); err != nil {
		var ee exitError
		if errors.As(err, &ee) {
			return ee.code
		}
		return 1
	}
	return 0
}

func newRootCommand() *cobra.Command {
	name := currentWrapperName()
	opts := &cliOptions{}

	cmd := &cobra.Command{
		Use:           fmt.Sprintf("%s [flags] <task>|resume <session_id> <task> [workdir]", name),
		Short:         "Go wrapper for AI CLI backends",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.Version {
				fmt.Printf("%s version %s\n", name, version)
				return nil
			}
			if opts.Cleanup {
				code := runCleanupMode()
				if code == 0 {
					return nil
				}
				return exitError{code: code}
			}

			exitCode := runWithLoggerAndCleanup(func() int {
				v, err := config.NewViper(opts.ConfigFile)
				if err != nil {
					logError(err.Error())
					return 1
				}

				if opts.Parallel {
					return runParallelMode(cmd, args, opts, v, name)
				}

				logInfo("Script started")

				cfg, err := buildSingleConfig(cmd, args, os.Args[1:], opts, v)
				if err != nil {
					logError(err.Error())
					return 1
				}
				logInfo(fmt.Sprintf("Parsed args: mode=%s, task_len=%d, backend=%s", cfg.Mode, len(cfg.Task), cfg.Backend))
				return runSingleMode(cfg, name)
			})

			if exitCode == 0 {
				return nil
			}
			return exitError{code: exitCode}
		},
	}
	cmd.CompletionOptions.DisableDefaultCmd = true

	addRootFlags(cmd.Flags(), opts)
	cmd.AddCommand(newVersionCommand(name), newCleanupCommand())

	return cmd
}

func addRootFlags(fs *pflag.FlagSet, opts *cliOptions) {
	fs.StringVar(&opts.ConfigFile, "config", "", "Config file path (default: $HOME/.codeagent/config.*)")
	fs.BoolVarP(&opts.Version, "version", "v", false, "Print version and exit")
	fs.BoolVar(&opts.Cleanup, "cleanup", false, "Clean up old logs and exit")

	fs.BoolVar(&opts.Parallel, "parallel", false, "Run tasks in parallel (config from stdin)")
	fs.BoolVar(&opts.FullOutput, "full-output", false, "Parallel mode: include full task output (legacy)")

	fs.StringVar(&opts.Backend, "backend", defaultBackendName, "Backend to use (codex, claude, gemini, opencode)")
	fs.StringVar(&opts.Model, "model", "", "Model override")
	fs.StringVar(&opts.ReasoningEffort, "reasoning-effort", "", "Reasoning effort (backend-specific)")
	fs.StringVar(&opts.Agent, "agent", "", "Agent preset name (from ~/.codeagent/models.json)")
	fs.StringVar(&opts.PromptFile, "prompt-file", "", "Prompt file path")

	fs.BoolVar(&opts.SkipPermissions, "skip-permissions", false, "Skip permissions prompts (also via CODEAGENT_SKIP_PERMISSIONS)")
	fs.BoolVar(&opts.SkipPermissions, "dangerously-skip-permissions", false, "Alias for --skip-permissions")
}

func newVersionCommand(name string) *cobra.Command {
	return &cobra.Command{
		Use:           "version",
		Short:         "Print version and exit",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("%s version %s\n", name, version)
			return nil
		},
	}
}

func newCleanupCommand() *cobra.Command {
	return &cobra.Command{
		Use:           "cleanup",
		Short:         "Clean up old logs and exit",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			code := runCleanupMode()
			if code == 0 {
				return nil
			}
			return exitError{code: code}
		},
	}
}

func runWithLoggerAndCleanup(fn func() int) (exitCode int) {
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
				fmt.Fprintf(os.Stderr, "Log file: %s (deleted)\n", logger.Path())
			}
		}
		_ = logger.RemoveLogFile()
	}()
	defer runCleanupHook()

	// Clean up stale logs from previous runs.
	scheduleStartupCleanup()

	return fn()
}

func parseArgs() (*Config, error) {
	opts := &cliOptions{}
	cmd := &cobra.Command{SilenceErrors: true, SilenceUsage: true, Args: cobra.ArbitraryArgs}
	addRootFlags(cmd.Flags(), opts)

	rawArgv := os.Args[1:]
	if err := cmd.ParseFlags(rawArgv); err != nil {
		return nil, err
	}
	args := cmd.Flags().Args()

	v, err := config.NewViper(opts.ConfigFile)
	if err != nil {
		return nil, err
	}

	return buildSingleConfig(cmd, args, rawArgv, opts, v)
}

func buildSingleConfig(cmd *cobra.Command, args []string, rawArgv []string, opts *cliOptions, v *viper.Viper) (*Config, error) {
	backendName := defaultBackendName
	model := ""
	reasoningEffort := ""
	agentName := ""
	promptFile := ""
	promptFileExplicit := false
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
	if agentName != "" {
		var resolvedYolo bool
		resolvedBackend, resolvedModel, resolvedPromptFile, resolvedReasoning, _, _, resolvedYolo = config.ResolveAgentConfig(agentName)
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

	agentFlagChanged := cmd.Flags().Changed("agent")
	backendFlagChanged := cmd.Flags().Changed("backend")
	if backendFlagChanged {
		backendName = strings.TrimSpace(opts.Backend)
		if backendName == "" {
			return nil, fmt.Errorf("--backend flag requires a value")
		}
	}

	switch {
	case agentFlagChanged && backendFlagChanged && lastFlagIndex(rawArgv, "agent") > lastFlagIndex(rawArgv, "backend"):
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
	case agentFlagChanged && modelFlagChanged && lastFlagIndex(rawArgv, "agent") > lastFlagIndex(rawArgv, "model"):
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

	cfg := &Config{
		WorkDir:            defaultWorkdir,
		Backend:            backendName,
		Agent:              agentName,
		PromptFile:         promptFile,
		PromptFileExplicit: promptFileExplicit,
		SkipPermissions:    skipPermissions,
		Yolo:               yolo,
		Model:              model,
		ReasoningEffort:    reasoningEffort,
		MaxParallelWorkers: config.ResolveMaxParallelWorkers(),
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
		cfg.ExplicitStdin = (args[2] == "-")
		if len(args) > 3 {
			if args[3] == "-" {
				return nil, fmt.Errorf("invalid workdir: '-' is not a valid directory path")
			}
			cfg.WorkDir = args[3]
		}
	} else {
		cfg.Mode = "new"
		cfg.Task = args[0]
		cfg.ExplicitStdin = (args[0] == "-")
		if len(args) > 1 {
			if args[1] == "-" {
				return nil, fmt.Errorf("invalid workdir: '-' is not a valid directory path")
			}
			cfg.WorkDir = args[1]
		}
	}

	return cfg, nil
}

func lastFlagIndex(argv []string, name string) int {
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

func runParallelMode(cmd *cobra.Command, args []string, opts *cliOptions, v *viper.Viper, name string) int {
	if len(args) > 0 {
		fmt.Fprintln(os.Stderr, "ERROR: --parallel reads its task configuration from stdin; no positional arguments are allowed.")
		fmt.Fprintln(os.Stderr, "Usage examples:")
		fmt.Fprintf(os.Stderr, "  %s --parallel < tasks.txt\n", name)
		fmt.Fprintf(os.Stderr, "  echo '...' | %s --parallel\n", name)
		fmt.Fprintf(os.Stderr, "  %s --parallel <<'EOF'\n", name)
		fmt.Fprintf(os.Stderr, "  %s --parallel --full-output <<'EOF'  # include full task output\n", name)
		return 1
	}

	if cmd.Flags().Changed("agent") || cmd.Flags().Changed("prompt-file") || cmd.Flags().Changed("reasoning-effort") {
		fmt.Fprintln(os.Stderr, "ERROR: --parallel reads its task configuration from stdin; only --backend, --model, --full-output and --skip-permissions are allowed.")
		return 1
	}

	backendName := defaultBackendName
	if cmd.Flags().Changed("backend") {
		backendName = strings.TrimSpace(opts.Backend)
		if backendName == "" {
			fmt.Fprintln(os.Stderr, "ERROR: --backend flag requires a value")
			return 1
		}
	} else if val := strings.TrimSpace(v.GetString("backend")); val != "" {
		backendName = val
	}

	model := ""
	if cmd.Flags().Changed("model") {
		model = strings.TrimSpace(opts.Model)
		if model == "" {
			fmt.Fprintln(os.Stderr, "ERROR: --model flag requires a value")
			return 1
		}
	} else {
		model = strings.TrimSpace(v.GetString("model"))
	}

	fullOutput := opts.FullOutput
	if !cmd.Flags().Changed("full-output") && v.IsSet("full-output") {
		fullOutput = v.GetBool("full-output")
	}

	skipChanged := cmd.Flags().Changed("skip-permissions") || cmd.Flags().Changed("dangerously-skip-permissions")
	skipPermissions := false
	if skipChanged {
		skipPermissions = opts.SkipPermissions
	} else {
		skipPermissions = v.GetBool("skip-permissions")
	}

	backend, err := selectBackendFn(backendName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}
	backendName = backend.Name()

	data, err := io.ReadAll(stdinReader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to read stdin: %v\n", err)
		return 1
	}

	cfg, err := parseParallelConfig(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}

	cfg.GlobalBackend = backendName
	model = strings.TrimSpace(model)
	for i := range cfg.Tasks {
		if strings.TrimSpace(cfg.Tasks[i].Backend) == "" {
			cfg.Tasks[i].Backend = backendName
		}
		if strings.TrimSpace(cfg.Tasks[i].Model) == "" && model != "" {
			cfg.Tasks[i].Model = model
		}
		cfg.Tasks[i].SkipPermissions = cfg.Tasks[i].SkipPermissions || skipPermissions
	}

	timeoutSec := resolveTimeout()
	layers, err := topologicalSort(cfg.Tasks)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return 1
	}

	results := executeConcurrent(layers, timeoutSec)

	for i := range results {
		results[i].CoverageTarget = defaultCoverageTarget
		if results[i].Message == "" {
			continue
		}

		lines := strings.Split(results[i].Message, "\n")
		results[i].Coverage = extractCoverageFromLines(lines)
		results[i].CoverageNum = extractCoverageNum(results[i].Coverage)
		results[i].FilesChanged = extractFilesChangedFromLines(lines)
		results[i].TestsPassed, results[i].TestsFailed = extractTestResultsFromLines(lines)
		results[i].KeyOutput = extractKeyOutputFromLines(lines, 150)
	}

	fmt.Println(generateFinalOutputWithMode(results, !fullOutput))

	exitCode := 0
	for _, res := range results {
		if res.ExitCode != 0 {
			exitCode = res.ExitCode
		}
	}
	return exitCode
}

func runSingleMode(cfg *Config, name string) int {
	backend, err := selectBackendFn(cfg.Backend)
	if err != nil {
		logError(err.Error())
		return 1
	}
	cfg.Backend = backend.Name()

	cmdInjected := codexCommand != defaultCodexCommand
	argsInjected := buildCodexArgsFn != nil && reflect.ValueOf(buildCodexArgsFn).Pointer() != reflect.ValueOf(defaultBuildArgsFn).Pointer()

	if backend.Name() != defaultBackendName || !cmdInjected {
		codexCommand = backend.Command()
	}
	if backend.Name() != defaultBackendName || !argsInjected {
		buildCodexArgsFn = backend.BuildArgs
	}
	logInfo(fmt.Sprintf("Selected backend: %s", backend.Name()))

	timeoutSec := resolveTimeout()
	logInfo(fmt.Sprintf("Timeout: %ds", timeoutSec))
	cfg.Timeout = timeoutSec

	var taskText string
	var piped bool

	if cfg.ExplicitStdin {
		logInfo("Explicit stdin mode: reading task from stdin")
		data, err := io.ReadAll(stdinReader)
		if err != nil {
			logError("Failed to read stdin: " + err.Error())
			return 1
		}
		taskText = string(data)
		if taskText == "" {
			logError("Explicit stdin mode requires task input from stdin")
			return 1
		}
		piped = !isTerminal()
	} else {
		pipedTask, err := readPipedTask()
		if err != nil {
			logError("Failed to read piped stdin: " + err.Error())
			return 1
		}
		piped = pipedTask != ""
		if piped {
			taskText = pipedTask
		} else {
			taskText = cfg.Task
		}
	}

	if strings.TrimSpace(cfg.PromptFile) != "" {
		prompt, err := readAgentPromptFile(cfg.PromptFile, cfg.PromptFileExplicit)
		if err != nil {
			logError("Failed to read prompt file: " + err.Error())
			return 1
		}
		taskText = wrapTaskWithAgentPrompt(prompt, taskText)
	}

	useStdin := cfg.ExplicitStdin || shouldUseStdin(taskText, piped)

	targetArg := taskText
	if useStdin {
		targetArg = "-"
	}
	codexArgs := buildCodexArgsFn(cfg, targetArg)

	logger := activeLogger()
	if logger == nil {
		fmt.Fprintln(os.Stderr, "ERROR: logger is not initialized")
		return 1
	}

	fmt.Fprintf(os.Stderr, "[%s]\n", name)
	fmt.Fprintf(os.Stderr, "  Backend: %s\n", cfg.Backend)
	fmt.Fprintf(os.Stderr, "  Command: %s %s\n", codexCommand, strings.Join(codexArgs, " "))
	fmt.Fprintf(os.Stderr, "  PID: %d\n", os.Getpid())
	fmt.Fprintf(os.Stderr, "  Log: %s\n", logger.Path())

	if useStdin {
		var reasons []string
		if piped {
			reasons = append(reasons, "piped input")
		}
		if cfg.ExplicitStdin {
			reasons = append(reasons, "explicit \"-\"")
		}
		if strings.Contains(taskText, "\n") {
			reasons = append(reasons, "newline")
		}
		if strings.Contains(taskText, "\\") {
			reasons = append(reasons, "backslash")
		}
		if strings.Contains(taskText, "\"") {
			reasons = append(reasons, "double-quote")
		}
		if strings.Contains(taskText, "'") {
			reasons = append(reasons, "single-quote")
		}
		if strings.Contains(taskText, "`") {
			reasons = append(reasons, "backtick")
		}
		if strings.Contains(taskText, "$") {
			reasons = append(reasons, "dollar")
		}
		if len(taskText) > 800 {
			reasons = append(reasons, "length>800")
		}
		if len(reasons) > 0 {
			logWarn(fmt.Sprintf("Using stdin mode for task due to: %s", strings.Join(reasons, ", ")))
		}
	}

	logInfo(fmt.Sprintf("%s running...", cfg.Backend))

	taskSpec := TaskSpec{
		Task:            taskText,
		WorkDir:         cfg.WorkDir,
		Mode:            cfg.Mode,
		SessionID:       cfg.SessionID,
		Model:           cfg.Model,
		ReasoningEffort: cfg.ReasoningEffort,
		Agent:           cfg.Agent,
		SkipPermissions: cfg.SkipPermissions,
		UseStdin:        useStdin,
	}

	result := runTaskFn(taskSpec, false, cfg.Timeout)

	if result.ExitCode != 0 {
		return result.ExitCode
	}

	fmt.Println(result.Message)
	if result.SessionID != "" {
		fmt.Printf("\n---\nSESSION_ID: %s\n", result.SessionID)
	}

	return 0
}
