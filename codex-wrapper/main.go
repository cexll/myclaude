package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const (
	version            = "4.8.2"
	defaultWorkdir     = "."
	defaultTimeout     = 7200 // seconds
	codexLogLineLimit  = 1000
	stdinSpecialChars  = "\n\\\"'`$"
	stderrCaptureLimit = 4 * 1024
)

// Test hooks for dependency injection
var (
	stdinReader  io.Reader = os.Stdin
	isTerminalFn           = defaultIsTerminal
	codexCommand           = "codex"
	cleanupHook  func()
	loggerPtr    atomic.Pointer[Logger]

	buildCodexArgsFn = buildCodexArgs
	commandContext   = exec.CommandContext
	jsonMarshal      = json.Marshal
	cleanupLogsFn    = cleanupOldLogs
	signalNotifyFn   = signal.Notify
	signalStopFn     = signal.Stop
)

var forceKillDelay atomic.Int32

func init() {
	forceKillDelay.Store(5) // seconds - default value
}

// Config holds CLI configuration
type Config struct {
	Mode          string // "new" or "resume"
	Task          string
	SessionID     string
	WorkDir       string
	ExplicitStdin bool
	Timeout       int
}

// ParallelConfig defines the JSON schema for parallel execution
type ParallelConfig struct {
	Tasks []TaskSpec `json:"tasks"`
}

// TaskSpec describes an individual task entry in the parallel config
type TaskSpec struct {
	ID           string   `json:"id"`
	Task         string   `json:"task"`
	WorkDir      string   `json:"workdir,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
	SessionID    string   `json:"session_id,omitempty"`
	Mode         string   `json:"-"`
	UseStdin     bool     `json:"-"`
}

// TaskResult captures the execution outcome of a task
type TaskResult struct {
	TaskID    string `json:"task_id"`
	ExitCode  int    `json:"exit_code"`
	Message   string `json:"message"`
	SessionID string `json:"session_id"`
	Error     string `json:"error"`
}

func parseParallelConfig(data []byte) (*ParallelConfig, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("parallel config is empty")
	}

	tasks := strings.Split(string(trimmed), "---TASK---")
	var cfg ParallelConfig
	seen := make(map[string]struct{})

	for _, taskBlock := range tasks {
		taskBlock = strings.TrimSpace(taskBlock)
		if taskBlock == "" {
			continue
		}

		parts := strings.SplitN(taskBlock, "---CONTENT---", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("task block missing ---CONTENT--- separator")
		}

		meta := strings.TrimSpace(parts[0])
		content := strings.TrimSpace(parts[1])

		task := TaskSpec{WorkDir: defaultWorkdir}
		for _, line := range strings.Split(meta, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			kv := strings.SplitN(line, ":", 2)
			if len(kv) != 2 {
				continue
			}
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])

			switch key {
			case "id":
				task.ID = value
			case "workdir":
				task.WorkDir = value
			case "session_id":
				task.SessionID = value
				task.Mode = "resume"
			case "dependencies":
				for _, dep := range strings.Split(value, ",") {
					dep = strings.TrimSpace(dep)
					if dep != "" {
						task.Dependencies = append(task.Dependencies, dep)
					}
				}
			}
		}

		if task.ID == "" {
			return nil, fmt.Errorf("task missing id field")
		}
		if content == "" {
			return nil, fmt.Errorf("task %q missing content", task.ID)
		}
		if _, exists := seen[task.ID]; exists {
			return nil, fmt.Errorf("duplicate task id: %s", task.ID)
		}

		task.Task = content
		cfg.Tasks = append(cfg.Tasks, task)
		seen[task.ID] = struct{}{}
	}

	if len(cfg.Tasks) == 0 {
		return nil, fmt.Errorf("no tasks found")
	}

	return &cfg, nil
}

func topologicalSort(tasks []TaskSpec) ([][]TaskSpec, error) {
	idToTask := make(map[string]TaskSpec, len(tasks))
	indegree := make(map[string]int, len(tasks))
	adj := make(map[string][]string, len(tasks))

	for _, task := range tasks {
		idToTask[task.ID] = task
		indegree[task.ID] = 0
	}

	for _, task := range tasks {
		for _, dep := range task.Dependencies {
			if _, ok := idToTask[dep]; !ok {
				return nil, fmt.Errorf("dependency %q not found for task %q", dep, task.ID)
			}
			indegree[task.ID]++
			adj[dep] = append(adj[dep], task.ID)
		}
	}

	queue := make([]string, 0, len(tasks))
	for _, task := range tasks {
		if indegree[task.ID] == 0 {
			queue = append(queue, task.ID)
		}
	}

	layers := make([][]TaskSpec, 0)
	processed := 0

	for len(queue) > 0 {
		current := queue
		queue = nil
		layer := make([]TaskSpec, len(current))
		for i, id := range current {
			layer[i] = idToTask[id]
			processed++
		}
		layers = append(layers, layer)

		next := make([]string, 0)
		for _, id := range current {
			for _, neighbor := range adj[id] {
				indegree[neighbor]--
				if indegree[neighbor] == 0 {
					next = append(next, neighbor)
				}
			}
		}
		queue = append(queue, next...)
	}

	if processed != len(tasks) {
		cycleIDs := make([]string, 0)
		for id, deg := range indegree {
			if deg > 0 {
				cycleIDs = append(cycleIDs, id)
			}
		}
		sort.Strings(cycleIDs)
		return nil, fmt.Errorf("cycle detected involving tasks: %s", strings.Join(cycleIDs, ","))
	}

	return layers, nil
}

var runCodexTaskFn = func(task TaskSpec, timeout int) TaskResult {
	if task.WorkDir == "" {
		task.WorkDir = defaultWorkdir
	}
	if task.Mode == "" {
		task.Mode = "new"
	}
	if task.UseStdin || shouldUseStdin(task.Task, false) {
		task.UseStdin = true
	}

	return runCodexTask(task, true, timeout)
}

func executeConcurrent(layers [][]TaskSpec, timeout int) []TaskResult {
	totalTasks := 0
	for _, layer := range layers {
		totalTasks += len(layer)
	}

	results := make([]TaskResult, 0, totalTasks)
	failed := make(map[string]TaskResult, totalTasks)
	resultsCh := make(chan TaskResult, totalTasks)

	for _, layer := range layers {
		var wg sync.WaitGroup
		executed := 0

		for _, task := range layer {
			if skip, reason := shouldSkipTask(task, failed); skip {
				res := TaskResult{TaskID: task.ID, ExitCode: 1, Error: reason}
				results = append(results, res)
				failed[task.ID] = res
				continue
			}

			executed++
			wg.Add(1)
			go func(ts TaskSpec) {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						resultsCh <- TaskResult{TaskID: ts.ID, ExitCode: 1, Error: fmt.Sprintf("panic: %v", r)}
					}
				}()
				resultsCh <- runCodexTaskFn(ts, timeout)
			}(task)
		}

		wg.Wait()

		for i := 0; i < executed; i++ {
			res := <-resultsCh
			results = append(results, res)
			if res.ExitCode != 0 || res.Error != "" {
				failed[res.TaskID] = res
			}
		}
	}

	return results
}

func shouldSkipTask(task TaskSpec, failed map[string]TaskResult) (bool, string) {
	if len(task.Dependencies) == 0 {
		return false, ""
	}

	var blocked []string
	for _, dep := range task.Dependencies {
		if _, ok := failed[dep]; ok {
			blocked = append(blocked, dep)
		}
	}

	if len(blocked) == 0 {
		return false, ""
	}

	return true, fmt.Sprintf("skipped due to failed dependencies: %s", strings.Join(blocked, ","))
}

func generateFinalOutput(results []TaskResult) string {
	var sb strings.Builder

	success := 0
	failed := 0
	for _, res := range results {
		if res.ExitCode == 0 && res.Error == "" {
			success++
		} else {
			failed++
		}
	}

	sb.WriteString(fmt.Sprintf("=== Parallel Execution Summary ===\n"))
	sb.WriteString(fmt.Sprintf("Total: %d | Success: %d | Failed: %d\n\n", len(results), success, failed))

	for _, res := range results {
		sb.WriteString(fmt.Sprintf("--- Task: %s ---\n", res.TaskID))
		if res.Error != "" {
			sb.WriteString(fmt.Sprintf("Status: FAILED (exit code %d)\nError: %s\n", res.ExitCode, res.Error))
		} else if res.ExitCode != 0 {
			sb.WriteString(fmt.Sprintf("Status: FAILED (exit code %d)\n", res.ExitCode))
		} else {
			sb.WriteString("Status: SUCCESS\n")
		}
		if res.SessionID != "" {
			sb.WriteString(fmt.Sprintf("Session: %s\n", res.SessionID))
		}
		if res.Message != "" {
			sb.WriteString(fmt.Sprintf("\n%s\n", res.Message))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// JSONEvent represents a Codex JSON output event
type JSONEvent struct {
	Type     string     `json:"type"`
	ThreadID string     `json:"thread_id,omitempty"`
	Item     *EventItem `json:"item,omitempty"`
}

// EventItem represents the item field in a JSON event
type EventItem struct {
	Type string      `json:"type"`
	Text interface{} `json:"text"`
}

func main() {
	exitCode := run()
	os.Exit(exitCode)
}

func runStartupCleanup() {
	if cleanupLogsFn == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			logWarn(fmt.Sprintf("cleanupOldLogs panic: %v", r))
		}
	}()
	if _, err := cleanupLogsFn(); err != nil {
		logWarn(fmt.Sprintf("cleanupOldLogs error: %v", err))
	}
}

// run is the main logic, returns exit code for testability
func run() (exitCode int) {
	// Handle --version and --help first (no logger needed)
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--cleanup":
			return runCleanupMode()
		case "--version", "-v":
			fmt.Printf("codex-wrapper version %s\n", version)
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

	// Run cleanup asynchronously to avoid blocking startup
	go runStartupCleanup()

	// Handle remaining commands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--parallel":
			if len(os.Args) > 2 {
				fmt.Fprintln(os.Stderr, "ERROR: --parallel reads its task configuration from stdin and does not accept additional arguments.")
				fmt.Fprintln(os.Stderr, "Usage examples:")
				fmt.Fprintln(os.Stderr, "  codex-wrapper --parallel < tasks.txt")
				fmt.Fprintln(os.Stderr, "  echo '...' | codex-wrapper --parallel")
				fmt.Fprintln(os.Stderr, "  codex-wrapper --parallel <<'EOF'")
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
	logInfo(fmt.Sprintf("Parsed args: mode=%s, task_len=%d", cfg.Mode, len(cfg.Task)))

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
	fmt.Fprintf(os.Stderr, "[codex-wrapper]\n")
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

	logInfo("codex running...")

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

func runCleanupMode() int {
	if cleanupLogsFn == nil {
		fmt.Fprintln(os.Stderr, "Cleanup failed: log cleanup function not configured")
		return 1
	}

	stats, err := cleanupLogsFn()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cleanup failed: %v\n", err)
		return 1
	}

	fmt.Println("Cleanup completed")
	fmt.Printf("Files scanned: %d\n", stats.Scanned)
	fmt.Printf("Files deleted: %d\n", stats.Deleted)
	if len(stats.DeletedFiles) > 0 {
		for _, f := range stats.DeletedFiles {
			fmt.Printf("  - %s\n", f)
		}
	}
	fmt.Printf("Files kept: %d\n", stats.Kept)
	if len(stats.KeptFiles) > 0 {
		for _, f := range stats.KeptFiles {
			fmt.Printf("  - %s\n", f)
		}
	}
	if stats.Errors > 0 {
		fmt.Printf("Deletion errors: %d\n", stats.Errors)
	}
	return 0
}

func parseArgs() (*Config, error) {
	args := os.Args[1:]
	if len(args) == 0 {
		return nil, fmt.Errorf("task required")
	}

	cfg := &Config{WorkDir: defaultWorkdir}

	if args[0] == "resume" {
		if len(args) < 3 {
			return nil, fmt.Errorf("resume mode requires: resume <session_id> <task>")
		}
		cfg.Mode = "resume"
		cfg.SessionID = args[1]
		cfg.Task = args[2]
		cfg.ExplicitStdin = (args[2] == "-")
		if len(args) > 3 {
			cfg.WorkDir = args[3]
		}
	} else {
		cfg.Mode = "new"
		cfg.Task = args[0]
		cfg.ExplicitStdin = (args[0] == "-")
		if len(args) > 1 {
			cfg.WorkDir = args[1]
		}
	}

	return cfg, nil
}

func readPipedTask() (string, error) {
	if isTerminal() {
		logInfo("Stdin is tty, skipping pipe read")
		return "", nil
	}
	logInfo("Reading from stdin pipe...")
	data, err := io.ReadAll(stdinReader)
	if err != nil {
		return "", fmt.Errorf("read stdin: %w", err)
	}
	if len(data) == 0 {
		logInfo("Stdin pipe returned empty data")
		return "", nil
	}
	logInfo(fmt.Sprintf("Read %d bytes from stdin pipe", len(data)))
	return string(data), nil
}

func shouldUseStdin(taskText string, piped bool) bool {
	if piped {
		return true
	}
	if len(taskText) > 800 {
		return true
	}
	return strings.IndexAny(taskText, stdinSpecialChars) >= 0
}

func buildCodexArgs(cfg *Config, targetArg string) []string {
	if cfg.Mode == "resume" {
		return []string{
			"e",
			"--skip-git-repo-check",
			"--json",
			"resume",
			cfg.SessionID,
			targetArg,
		}
	}
	return []string{
		"e",
		"--skip-git-repo-check",
		"-C", cfg.WorkDir,
		"--json",
		targetArg,
	}
}

type parseResult struct {
	message  string
	threadID string
}

func runCodexTask(taskSpec TaskSpec, silent bool, timeoutSec int) TaskResult {
	return runCodexTaskWithContext(context.Background(), taskSpec, nil, false, silent, timeoutSec)
}

func runCodexProcess(parentCtx context.Context, codexArgs []string, taskText string, useStdin bool, timeoutSec int) (message, threadID string, exitCode int) {
	res := runCodexTaskWithContext(parentCtx, TaskSpec{Task: taskText, WorkDir: defaultWorkdir, Mode: "new", UseStdin: useStdin}, codexArgs, true, false, timeoutSec)
	return res.Message, res.SessionID, res.ExitCode
}

func runCodexTaskWithContext(parentCtx context.Context, taskSpec TaskSpec, customArgs []string, useCustomArgs bool, silent bool, timeoutSec int) TaskResult {
	result := TaskResult{TaskID: taskSpec.ID}

	cfg := &Config{
		Mode:      taskSpec.Mode,
		Task:      taskSpec.Task,
		SessionID: taskSpec.SessionID,
		WorkDir:   taskSpec.WorkDir,
	}
	if cfg.Mode == "" {
		cfg.Mode = "new"
	}
	if cfg.WorkDir == "" {
		cfg.WorkDir = defaultWorkdir
	}

	useStdin := taskSpec.UseStdin
	targetArg := taskSpec.Task
	if useStdin {
		targetArg = "-"
	}

	var codexArgs []string
	if useCustomArgs {
		codexArgs = customArgs
	} else {
		codexArgs = buildCodexArgsFn(cfg, targetArg)
	}

	prefixMsg := func(msg string) string {
		if taskSpec.ID == "" {
			return msg
		}
		return fmt.Sprintf("[Task: %s] %s", taskSpec.ID, msg)
	}

	var logInfoFn func(string)
	var logWarnFn func(string)
	var logErrorFn func(string)

	if silent {
		// Silent mode: only persist to file when available; avoid stderr noise.
		logInfoFn = func(msg string) {
			if logger := activeLogger(); logger != nil {
				logger.Info(prefixMsg(msg))
			}
		}
		logWarnFn = func(msg string) {
			if logger := activeLogger(); logger != nil {
				logger.Warn(prefixMsg(msg))
			}
		}
		logErrorFn = func(msg string) {
			if logger := activeLogger(); logger != nil {
				logger.Error(prefixMsg(msg))
			}
		}
	} else {
		logInfoFn = func(msg string) { logInfo(prefixMsg(msg)) }
		logWarnFn = func(msg string) { logWarn(prefixMsg(msg)) }
		logErrorFn = func(msg string) { logError(prefixMsg(msg)) }
	}

	stderrBuf := &tailBuffer{limit: stderrCaptureLimit}

	var stdoutLogger *logWriter
	var stderrLogger *logWriter

	var tempLogger *Logger
	if silent && activeLogger() == nil {
		if l, err := NewLogger(); err == nil {
			setLogger(l)
			tempLogger = l
		}
	}
	defer func() {
		if tempLogger != nil {
			closeLogger()
		}
	}()

	if !silent {
		stdoutLogger = newLogWriter("CODEX_STDOUT: ", codexLogLineLimit)
		stderrLogger = newLogWriter("CODEX_STDERR: ", codexLogLineLimit)
	}

	ctx := parentCtx
	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	attachStderr := func(msg string) string {
		return fmt.Sprintf("%s; stderr: %s", msg, stderrBuf.String())
	}

	cmd := commandContext(ctx, codexCommand, codexArgs...)

	stderrWriters := []io.Writer{stderrBuf}
	if stderrLogger != nil {
		stderrWriters = append(stderrWriters, stderrLogger)
	}
	if !silent {
		stderrWriters = append([]io.Writer{os.Stderr}, stderrWriters...)
	}
	if len(stderrWriters) == 1 {
		cmd.Stderr = stderrWriters[0]
	} else {
		cmd.Stderr = io.MultiWriter(stderrWriters...)
	}

	var stdinPipe io.WriteCloser
	var err error
	if useStdin {
		stdinPipe, err = cmd.StdinPipe()
		if err != nil {
			logErrorFn("Failed to create stdin pipe: " + err.Error())
			result.ExitCode = 1
			result.Error = attachStderr("failed to create stdin pipe: " + err.Error())
			return result
		}
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logErrorFn("Failed to create stdout pipe: " + err.Error())
		result.ExitCode = 1
		result.Error = attachStderr("failed to create stdout pipe: " + err.Error())
		return result
	}

	stdoutReader := io.Reader(stdout)
	if stdoutLogger != nil {
		stdoutReader = io.TeeReader(stdout, stdoutLogger)
	}

	logInfoFn(fmt.Sprintf("Starting codex with args: codex %s...", strings.Join(codexArgs[:min(5, len(codexArgs))], " ")))

	if err := cmd.Start(); err != nil {
		if strings.Contains(err.Error(), "executable file not found") {
			logErrorFn("codex command not found in PATH")
			result.ExitCode = 127
			result.Error = attachStderr("codex command not found in PATH")
			return result
		}
		logErrorFn("Failed to start codex: " + err.Error())
		result.ExitCode = 1
		result.Error = attachStderr("failed to start codex: " + err.Error())
		return result
	}

	logInfoFn(fmt.Sprintf("Starting codex with PID: %d", cmd.Process.Pid))
	if logger := activeLogger(); logger != nil {
		logInfoFn(fmt.Sprintf("Log capturing to: %s", logger.Path()))
	}

	if useStdin && stdinPipe != nil {
		logInfoFn(fmt.Sprintf("Writing %d chars to stdin...", len(taskSpec.Task)))
		go func(data string) {
			defer stdinPipe.Close()
			_, _ = io.WriteString(stdinPipe, data)
		}(taskSpec.Task)
		logInfoFn("Stdin closed")
	}

	waitCh := make(chan error, 1)
	go func() { waitCh <- cmd.Wait() }()

	parseCh := make(chan parseResult, 1)
	go func() {
		msg, tid := parseJSONStreamWithLog(stdoutReader, logWarnFn, logInfoFn)
		parseCh <- parseResult{message: msg, threadID: tid}
	}()

	var waitErr error
	var forceKillTimer *time.Timer

	select {
	case waitErr = <-waitCh:
	case <-ctx.Done():
		logErrorFn(cancelReason(ctx))
		forceKillTimer = terminateProcess(cmd)
		waitErr = <-waitCh
	}

	if forceKillTimer != nil {
		forceKillTimer.Stop()
	}

	parsed := <-parseCh

	if ctxErr := ctx.Err(); ctxErr != nil {
		if errors.Is(ctxErr, context.DeadlineExceeded) {
			result.ExitCode = 124
			result.Error = attachStderr("codex execution timeout")
			return result
		}
		result.ExitCode = 130
		result.Error = attachStderr("execution cancelled")
		return result
	}

	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			code := exitErr.ExitCode()
			logErrorFn(fmt.Sprintf("Codex exited with status %d", code))
			result.ExitCode = code
			result.Error = attachStderr(fmt.Sprintf("codex exited with status %d", code))
			return result
		}
		logErrorFn("Codex error: " + waitErr.Error())
		result.ExitCode = 1
		result.Error = attachStderr("codex error: " + waitErr.Error())
		return result
	}

	message := parsed.message
	threadID := parsed.threadID
	if message == "" {
		logErrorFn("Codex completed without agent_message output")
		result.ExitCode = 1
		result.Error = attachStderr("codex completed without agent_message output")
		return result
	}

	if stdoutLogger != nil {
		stdoutLogger.Flush()
	}
	if stderrLogger != nil {
		stderrLogger.Flush()
	}

	result.ExitCode = 0
	result.Message = message
	result.SessionID = threadID

	return result
}

type tailBuffer struct {
	limit int
	data  []byte
}

func (b *tailBuffer) Write(p []byte) (int, error) {
	if b.limit <= 0 {
		return len(p), nil
	}

	if len(p) >= b.limit {
		b.data = append(b.data[:0], p[len(p)-b.limit:]...)
		return len(p), nil
	}

	total := len(b.data) + len(p)
	if total <= b.limit {
		b.data = append(b.data, p...)
		return len(p), nil
	}

	overflow := total - b.limit
	b.data = append(b.data[overflow:], p...)
	return len(p), nil
}

func (b *tailBuffer) String() string {
	return string(b.data)
}

func forwardSignals(ctx context.Context, cmd *exec.Cmd, logErrorFn func(string)) {
	sigCh := make(chan os.Signal, 1)
	signals := []os.Signal{syscall.SIGINT}
	if runtime.GOOS != "windows" {
		signals = append(signals, syscall.SIGTERM)
	}
	signalNotifyFn(sigCh, signals...)

	go func() {
		defer signalStopFn(sigCh)
		select {
		case sig := <-sigCh:
			logErrorFn(fmt.Sprintf("Received signal: %v", sig))
			if cmd.Process == nil {
				return
			}
			if runtime.GOOS == "windows" {
				_ = cmd.Process.Kill()
				return
			}
			_ = cmd.Process.Signal(syscall.SIGTERM)
			time.AfterFunc(time.Duration(forceKillDelay.Load())*time.Second, func() {
				if cmd.Process != nil {
					_ = cmd.Process.Kill()
				}
			})
		case <-ctx.Done():
		}
	}()
}

func cancelReason(ctx context.Context) string {
	if ctx == nil {
		return "Context cancelled"
	}

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return "Codex execution timeout"
	}

	return "Execution cancelled, terminating codex process"
}

func terminateProcess(cmd *exec.Cmd) *time.Timer {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	if runtime.GOOS == "windows" {
		_ = cmd.Process.Kill()
		return nil
	}

	_ = cmd.Process.Signal(syscall.SIGTERM)

	return time.AfterFunc(time.Duration(forceKillDelay.Load())*time.Second, func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	})
}

func parseJSONStream(r io.Reader) (message, threadID string) {
	return parseJSONStreamWithLog(r, logWarn, logInfo)
}

func parseJSONStreamWithWarn(r io.Reader, warnFn func(string)) (message, threadID string) {
	return parseJSONStreamWithLog(r, warnFn, logInfo)
}

func parseJSONStreamWithLog(r io.Reader, warnFn func(string), infoFn func(string)) (message, threadID string) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)

	if warnFn == nil {
		warnFn = func(string) {}
	}
	if infoFn == nil {
		infoFn = func(string) {}
	}

	totalEvents := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		totalEvents++

		var event JSONEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			warnFn(fmt.Sprintf("Failed to parse line: %s", truncate(line, 100)))
			continue
		}

		var details []string
		if event.ThreadID != "" {
			details = append(details, fmt.Sprintf("thread_id=%s", event.ThreadID))
		}
		if event.Item != nil && event.Item.Type != "" {
			details = append(details, fmt.Sprintf("item_type=%s", event.Item.Type))
		}
		if len(details) > 0 {
			infoFn(fmt.Sprintf("Parsed event #%d type=%s (%s)", totalEvents, event.Type, strings.Join(details, ", ")))
		} else {
			infoFn(fmt.Sprintf("Parsed event #%d type=%s", totalEvents, event.Type))
		}

		switch event.Type {
		case "thread.started":
			threadID = event.ThreadID
			infoFn(fmt.Sprintf("thread.started event thread_id=%s", threadID))
		case "item.completed":
			var itemType string
			var normalized string
			if event.Item != nil {
				itemType = event.Item.Type
				normalized = normalizeText(event.Item.Text)
			}
			infoFn(fmt.Sprintf("item.completed event item_type=%s message_len=%d", itemType, len(normalized)))
			if event.Item != nil && event.Item.Type == "agent_message" && normalized != "" {
				message = normalized
			}
		}
	}

	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		warnFn("Read stdout error: " + err.Error())
	}

	infoFn(fmt.Sprintf("parseJSONStream completed: events=%d, message_len=%d, thread_id_found=%t", totalEvents, len(message), threadID != ""))
	return message, threadID
}

func discardInvalidJSON(decoder *json.Decoder, reader *bufio.Reader) (*bufio.Reader, error) {
	var buffered bytes.Buffer

	if decoder != nil {
		if buf := decoder.Buffered(); buf != nil {
			_, _ = buffered.ReadFrom(buf)
		}
	}

	line, err := reader.ReadBytes('\n')
	buffered.Write(line)

	data := buffered.Bytes()
	newline := bytes.IndexByte(data, '\n')
	if newline == -1 {
		return reader, err
	}

	remaining := data[newline+1:]
	if len(remaining) == 0 {
		return reader, err
	}

	return bufio.NewReader(io.MultiReader(bytes.NewReader(remaining), reader)), err
}

func normalizeText(text interface{}) string {
	switch v := text.(type) {
	case string:
		return v
	case []interface{}:
		var sb strings.Builder
		for _, item := range v {
			if s, ok := item.(string); ok {
				sb.WriteString(s)
			}
		}
		return sb.String()
	default:
		return ""
	}
}

func resolveTimeout() int {
	raw := os.Getenv("CODEX_TIMEOUT")
	if raw == "" {
		return defaultTimeout
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		logWarn(fmt.Sprintf("Invalid CODEX_TIMEOUT '%s', falling back to %ds", raw, defaultTimeout))
		return defaultTimeout
	}

	if parsed > 10000 {
		return parsed / 1000
	}
	return parsed
}

func defaultIsTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return true
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func isTerminal() bool {
	return isTerminalFn()
}

func getEnv(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

type logWriter struct {
	prefix string
	maxLen int
	buf    bytes.Buffer
}

func newLogWriter(prefix string, maxLen int) *logWriter {
	if maxLen <= 0 {
		maxLen = codexLogLineLimit
	}
	return &logWriter{prefix: prefix, maxLen: maxLen}
}

func (lw *logWriter) Write(p []byte) (int, error) {
	if lw == nil {
		return len(p), nil
	}
	total := len(p)
	for len(p) > 0 {
		if idx := bytes.IndexByte(p, '\n'); idx >= 0 {
			lw.buf.Write(p[:idx])
			lw.logLine(true)
			p = p[idx+1:]
			continue
		}
		lw.buf.Write(p)
		break
	}
	return total, nil
}

func (lw *logWriter) Flush() {
	if lw == nil || lw.buf.Len() == 0 {
		return
	}
	lw.logLine(false)
}

func (lw *logWriter) logLine(force bool) {
	if lw == nil {
		return
	}
	line := lw.buf.String()
	lw.buf.Reset()
	if line == "" && !force {
		return
	}
	if lw.maxLen > 0 && len(line) > lw.maxLen {
		cutoff := lw.maxLen
		if cutoff > 3 {
			line = line[:cutoff-3] + "..."
		} else {
			line = line[:cutoff]
		}
	}
	logInfo(lw.prefix + line)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen < 0 {
		return ""
	}
	return s[:maxLen] + "..."
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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

func hello() string {
	return "hello world"
}

func greet(name string) string {
	return "hello " + name
}

func farewell(name string) string {
	return "goodbye " + name
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
	help := `codex-wrapper - Go wrapper for Codex CLI

Usage:
    codex-wrapper "task" [workdir]
    codex-wrapper - [workdir]              Read task from stdin
    codex-wrapper resume <session_id> "task" [workdir]
    codex-wrapper resume <session_id> - [workdir]
    codex-wrapper --parallel               Run tasks in parallel (config from stdin)
    codex-wrapper --version
    codex-wrapper --help

Parallel mode examples:
    codex-wrapper --parallel < tasks.txt
    echo '...' | codex-wrapper --parallel
    codex-wrapper --parallel <<'EOF'

Environment Variables:
    CODEX_TIMEOUT  Timeout in milliseconds (default: 7200000)

Exit Codes:
    0    Success
    1    General error (missing args, no output)
    124  Timeout
    127  codex command not found
    130  Interrupted (Ctrl+C)
    *    Passthrough from codex process`
	fmt.Println(help)
}
