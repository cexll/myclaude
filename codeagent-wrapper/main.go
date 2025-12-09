package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
)

const (
	version            = "5.0.0"
	defaultWorkdir     = "."
	defaultTimeout     = 7200 // seconds
	codexLogLineLimit  = 1000
	stdinSpecialChars  = "\n\\\"'`$"
	stderrCaptureLimit = 4 * 1024
	defaultBackendName = "codex"
	wrapperName        = "codeagent-wrapper"
)

// Test hooks for dependency injection
var (
	stdinReader  io.Reader = os.Stdin
	isTerminalFn           = defaultIsTerminal
	codexCommand           = "codex"
	cleanupHook  func()
	loggerPtr    atomic.Pointer[Logger]

	buildCodexArgsFn = buildCodexArgs
	selectBackendFn  = selectBackend
	commandContext   = exec.CommandContext
	jsonMarshal      = json.Marshal
	forceKillDelay   = 5 // seconds - made variable for testability
)

func main() {
	exitCode := run()
	os.Exit(exitCode)
}

// run is the main logic, returns exit code for testability
func run() (exitCode int) {
	// Handle --version and --help first (no logger needed)
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Printf("%s version %s\n", wrapperName, version)
			return 0
		case "--help", "-h":
			printHelp()
			return 0
		}
	}

	// Initialize logger for all other commands
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
		// Always remove log file after completion
		if logger != nil {
			if err := logger.RemoveLogFile(); err != nil && !os.IsNotExist(err) {
				// Silently ignore removal errors
			}
		}
	}()
	defer runCleanupHook()

	// Handle remaining commands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--parallel":
			if len(os.Args) > 2 {
				fmt.Fprintln(os.Stderr, "ERROR: --parallel reads its task configuration from stdin and does not accept additional arguments.")
				fmt.Fprintln(os.Stderr, "Usage examples:")
				fmt.Fprintf(os.Stderr, "  %s --parallel < tasks.txt\n", wrapperName)
				fmt.Fprintf(os.Stderr, "  echo '...' | %s --parallel\n", wrapperName)
				fmt.Fprintf(os.Stderr, "  %s --parallel <<'EOF'\n", wrapperName)
				return 1
			}
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

			timeoutSec := resolveTimeout()
			layers, err := topologicalSort(cfg.Tasks)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
				return 1
			}

			results := executeConcurrent(layers, timeoutSec)
			fmt.Println(generateFinalOutput(results))

			exitCode = 0
			for _, res := range results {
				if res.ExitCode != 0 {
					exitCode = res.ExitCode
				}
			}

			return exitCode
		}
	}

	logInfo("Script started")

	cfg, err := parseArgs()
	if err != nil {
		logError(err.Error())
		return 1
	}
	logInfo(fmt.Sprintf("Parsed args: mode=%s, task_len=%d, backend=%s", cfg.Mode, len(cfg.Task), cfg.Backend))

	backend, err := selectBackendFn(cfg.Backend)
	if err != nil {
		logError(err.Error())
		return 1
	}
	// Wire selected backend into runtime hooks for the rest of the execution.
	codexCommand = backend.Command()
	buildCodexArgsFn = backend.BuildArgs
	cfg.Backend = backend.Name()
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

	useStdin := cfg.ExplicitStdin || shouldUseStdin(taskText, piped)

	targetArg := taskText
	if useStdin {
		targetArg = "-"
	}
	codexArgs := buildCodexArgsFn(cfg, targetArg)

	// Print startup information to stderr
	fmt.Fprintf(os.Stderr, "[%s]\n", wrapperName)
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
		Task:      taskText,
		WorkDir:   cfg.WorkDir,
		Mode:      cfg.Mode,
		SessionID: cfg.SessionID,
		UseStdin:  useStdin,
	}

	result := runCodexTask(taskSpec, false, cfg.Timeout)

	if result.ExitCode != 0 {
		return result.ExitCode
	}

	fmt.Println(result.Message)
	if result.SessionID != "" {
		fmt.Printf("\n---\nSESSION_ID: %s\n", result.SessionID)
	}

	return 0
}

func setLogger(l *Logger) {
	loggerPtr.Store(l)
}

func closeLogger() error {
	logger := loggerPtr.Swap(nil)
	if logger == nil {
		return nil
	}
	return logger.Close()
}

func activeLogger() *Logger {
	return loggerPtr.Load()
}

func logInfo(msg string) {
	if logger := activeLogger(); logger != nil {
		logger.Info(msg)
	}
}

func logWarn(msg string) {
	if logger := activeLogger(); logger != nil {
		logger.Warn(msg)
	}
}

func logError(msg string) {
	if logger := activeLogger(); logger != nil {
		logger.Error(msg)
	}
}

func runCleanupHook() {
	if logger := activeLogger(); logger != nil {
		logger.Flush()
	}
	if cleanupHook != nil {
		cleanupHook()
	}
}

func printHelp() {
	help := `codeagent-wrapper - Go wrapper for AI CLI backends

Usage:
    codeagent-wrapper "task" [workdir]
    codeagent-wrapper --backend claude "task" [workdir]
    codeagent-wrapper - [workdir]              Read task from stdin
    codeagent-wrapper resume <session_id> "task" [workdir]
    codeagent-wrapper resume <session_id> - [workdir]
    codeagent-wrapper --parallel               Run tasks in parallel (config from stdin)
    codeagent-wrapper --version
    codeagent-wrapper --help

Parallel mode examples:
    codeagent-wrapper --parallel < tasks.txt
    echo '...' | codeagent-wrapper --parallel
    codeagent-wrapper --parallel <<'EOF'

Environment Variables:
    CODEX_TIMEOUT  Timeout in milliseconds (default: 7200000)

Exit Codes:
    0    Success
    1    General error (missing args, no output)
    124  Timeout
    127  backend command not found
    130  Interrupted (Ctrl+C)
    *    Passthrough from backend process`
	fmt.Println(help)
}
