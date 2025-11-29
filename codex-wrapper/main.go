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
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

const (
	version        = "1.0.0"
	defaultWorkdir = "."
	defaultTimeout = 7200 // seconds
	forceKillDelay = 5    // seconds
)

// Test hooks for dependency injection
var (
	stdinReader  io.Reader = os.Stdin
	isTerminalFn           = defaultIsTerminal
	codexCommand           = "codex"
	cleanupHook  func()
	loggerPtr    atomic.Pointer[Logger]
)

// Config holds CLI configuration
type Config struct {
	Mode          string // "new" or "resume"
	Task          string
	SessionID     string
	WorkDir       string
	ExplicitStdin bool
	Timeout       int
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

// run is the main logic, returns exit code for testability
func run() int {
	logger, err := NewLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to initialize logger: %v\n", err)
		return 1
	}
	setLogger(logger)

	defer func() {
		if err := closeLogger(); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: failed to close logger: %v\n", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	defer runCleanupHook()

	// Handle --version and --help first
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Printf("codex-wrapper version %s\n", version)
			return 0
		case "--help", "-h":
			printHelp()
			return 0
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

	// Determine task text and stdin mode
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
		if len(taskText) > 800 {
			reasons = append(reasons, "length>800")
		}
		if len(reasons) > 0 {
			logWarn(fmt.Sprintf("Using stdin mode for task due to: %s", strings.Join(reasons, ", ")))
		}
	}

	targetArg := taskText
	if useStdin {
		targetArg = "-"
	}

	codexArgs := buildCodexArgs(cfg, targetArg)
	logInfo("codex running...")

	message, threadID, exitCode := runCodexProcess(ctx, codexArgs, taskText, useStdin, cfg.Timeout)

	if exitCode != 0 {
		return exitCode
	}

	// Output agent_message
	fmt.Println(message)

	// Output session_id if present
	if threadID != "" {
		fmt.Printf("\n---\nSESSION_ID: %s\n", threadID)
	}

	return 0
}

func parseArgs() (*Config, error) {
	args := os.Args[1:]
	if len(args) == 0 {
		return nil, fmt.Errorf("task required")
	}

	cfg := &Config{
		WorkDir: defaultWorkdir,
	}

	// Check for resume mode
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
	if strings.Contains(taskText, "\n") {
		return true
	}
	if strings.Contains(taskText, "\\") {
		return true
	}
	if len(taskText) > 800 {
		return true
	}
	return false
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

func runCodexProcess(parentCtx context.Context, codexArgs []string, taskText string, useStdin bool, timeoutSec int) (message, threadID string, exitCode int) {
	ctx, cancel := context.WithTimeout(parentCtx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	cmd := exec.Command(codexCommand, codexArgs...)
	cmd.Stderr = os.Stderr

	// Setup stdin if needed
	var stdinPipe io.WriteCloser
	var err error
	if useStdin {
		stdinPipe, err = cmd.StdinPipe()
		if err != nil {
			logError("Failed to create stdin pipe: " + err.Error())
			return "", "", 1
		}
	}

	// Setup stdout
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logError("Failed to create stdout pipe: " + err.Error())
		return "", "", 1
	}

	logInfo(fmt.Sprintf("Starting codex with args: codex %s...", strings.Join(codexArgs[:min(5, len(codexArgs))], " ")))

	// Start process
	if err := cmd.Start(); err != nil {
		if strings.Contains(err.Error(), "executable file not found") {
			logError("codex command not found in PATH")
			return "", "", 127
		}
		logError("Failed to start codex: " + err.Error())
		return "", "", 1
	}
	logInfo(fmt.Sprintf("Process started with PID: %d", cmd.Process.Pid))

	// Write to stdin if needed
	if useStdin && stdinPipe != nil {
		logInfo(fmt.Sprintf("Writing %d chars to stdin...", len(taskText)))
		go func() {
			defer stdinPipe.Close()
			io.WriteString(stdinPipe, taskText)
		}()
		logInfo("Stdin closed")
	}

	logInfo("Reading stdout...")

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()

	parseCh := make(chan parseResult, 1)
	go func() {
		msg, tid := parseJSONStream(stdout)
		parseCh <- parseResult{message: msg, threadID: tid}
	}()

	var waitErr error
	var forceKillTimer *time.Timer

	select {
	case waitErr = <-waitCh:
	case <-ctx.Done():
		logError(cancelReason(ctx))
		forceKillTimer = terminateProcess(cmd)
		waitErr = <-waitCh
	}

	if forceKillTimer != nil {
		forceKillTimer.Stop()
	}

	result := <-parseCh

	if ctxErr := ctx.Err(); ctxErr != nil {
		if errors.Is(ctxErr, context.DeadlineExceeded) {
			return "", "", 124
		}
		return "", "", 130
	}

	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			code := exitErr.ExitCode()
			logError(fmt.Sprintf("Codex exited with status %d", code))
			return "", "", code
		}
		logError("Codex error: " + waitErr.Error())
		return "", "", 1
	}

	message = result.message
	threadID = result.threadID
	if message == "" {
		logError("Codex completed without agent_message output")
		return "", "", 1
	}

	return message, threadID, 0
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

	_ = cmd.Process.Signal(syscall.SIGTERM)

	return time.AfterFunc(time.Duration(forceKillDelay)*time.Second, func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	})
}

func parseJSONStream(r io.Reader) (message, threadID string) {
	reader := bufio.NewReaderSize(r, 64*1024)
	decoder := json.NewDecoder(reader)

	for {
		var event JSONEvent
		if err := decoder.Decode(&event); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			logWarn(fmt.Sprintf("Failed to decode JSON: %v", err))
			var skipErr error
			reader, skipErr = discardInvalidJSON(decoder, reader)
			if skipErr != nil {
				if errors.Is(skipErr, os.ErrClosed) || errors.Is(skipErr, io.ErrClosedPipe) {
					logWarn("Read stdout error: " + skipErr.Error())
					break
				}
				if !errors.Is(skipErr, io.EOF) {
					logWarn("Read stdout error: " + skipErr.Error())
				}
			}
			decoder = json.NewDecoder(reader)
			continue
		}

		switch event.Type {
		case "thread.started":
			threadID = event.ThreadID
		case "item.completed":
			if event.Item != nil && event.Item.Type == "agent_message" {
				if text := normalizeText(event.Item.Text); text != "" {
					message = text
				}
			}
		}
	}

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

	// Environment variable is in milliseconds if > 10000, convert to seconds
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

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
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

func logInfo(msg string) {
	if logger := activeLogger(); logger != nil {
		logger.Info(msg)
		return
	}
	fmt.Fprintf(os.Stderr, "INFO: %s\n", msg)
}

func logWarn(msg string) {
	if logger := activeLogger(); logger != nil {
		logger.Warn(msg)
		return
	}
	fmt.Fprintf(os.Stderr, "WARN: %s\n", msg)
}

func logError(msg string) {
	if logger := activeLogger(); logger != nil {
		logger.Error(msg)
		return
	}
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", msg)
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
    codex-wrapper --version
    codex-wrapper --help

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
