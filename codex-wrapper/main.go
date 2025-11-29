package main

import (
	"bufio"
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
	"syscall"
	"time"
)

const (
	version        = "1.0.0"
	defaultWorkdir = "."
	defaultTimeout = 7200 // seconds
	forceKillDelay = 5    // seconds
	maxJSONLine    = 64 * 1024 * 1024
)

// Test hooks for dependency injection
var (
	stdinReader     io.Reader = os.Stdin
	isTerminalFn              = defaultIsTerminal
	codexCommand              = "codex"
	liveLogDefault            = "1"
	popupLogDefault           = "1"
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
		pipedTask := readPipedTask()
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

	message, threadID, exitCode := runCodexProcess(codexArgs, taskText, useStdin, cfg.Timeout)

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

func readPipedTask() string {
	if isTerminal() {
		logInfo("Stdin is tty, skipping pipe read")
		return ""
	}
	logInfo("Reading from stdin pipe...")
	data, err := io.ReadAll(stdinReader)
	if err != nil || len(data) == 0 {
		logInfo("Stdin pipe returned empty data")
		return ""
	}
	logInfo(fmt.Sprintf("Read %d bytes from stdin pipe", len(data)))
	return string(data)
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

func runCodexProcess(codexArgs []string, taskText string, useStdin bool, timeoutSec int) (message, threadID string, exitCode int) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, codexCommand, codexArgs...)
	cmd.Stderr = os.Stderr

	enableLiveLog := getEnv("CODEX_LIVE_LOG", liveLogDefault) != "0"
	enablePopup := getEnv("CODEX_POPUP_LOG", popupLogDefault) != "0"

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

	var mirrorWriters []io.Writer

	var livePipe io.WriteCloser
	var liveDone chan struct{}
	if enableLiveLog {
		liveReader, liveWriter := io.Pipe()
		livePipe = liveWriter
		liveDone = make(chan struct{})
		go func() {
			defer close(liveDone)
			streamLiveLog(liveReader)
		}()
		mirrorWriters = append(mirrorWriters, livePipe)
	}
	if livePipe != nil {
		defer func() {
			livePipe.Close()
			if liveDone != nil {
				<-liveDone
			}
		}()
	}

	var popupPipe io.WriteCloser
	if enableLiveLog && enablePopup {
		var waitFn func()
		popupPipe, waitFn = startPopupLog()
		if popupPipe != nil {
			mirrorWriters = append(mirrorWriters, popupPipe)
			defer func() {
				popupPipe.Close()
				if waitFn != nil {
					waitFn()
				}
			}()
		}
	}

	reader := io.Reader(stdout)
	if len(mirrorWriters) > 0 {
		reader = io.TeeReader(stdout, io.MultiWriter(mirrorWriters...))
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

	// Setup signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		logError(fmt.Sprintf("Received signal: %v", sig))
		if cmd.Process != nil {
			cmd.Process.Signal(syscall.SIGTERM)
			time.AfterFunc(time.Duration(forceKillDelay)*time.Second, func() {
				if cmd.Process != nil {
					cmd.Process.Kill()
				}
			})
		}
	}()

	logInfo("Reading stdout...")

	// Parse JSON stream
	message, threadID = parseJSONStream(reader)

	// Wait for process to complete
	err = cmd.Wait()

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		logError("Codex execution timeout")
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return "", "", 124
	}

	// Check exit code
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code := exitErr.ExitCode()
			logError(fmt.Sprintf("Codex exited with status %d", code))
			return "", "", code
		}
		logError("Codex error: " + err.Error())
		return "", "", 1
	}

	if message == "" {
		logError("Codex completed without agent_message output")
		return "", "", 1
	}

	return message, threadID, 0
}

func parseJSONStream(r io.Reader) (message, threadID string) {
	reader := bufio.NewReaderSize(r, 128*1024)

	for {
		line, err := readLineWithLimit(reader)
		if err != nil {
			if errors.Is(err, errLineTooLong) {
				logWarn(fmt.Sprintf("Skipping line exceeding %d bytes", maxJSONLine))
				if line == "" {
					if errors.Is(err, io.EOF) {
						break
					}
					continue
				}
			}
			if !errors.Is(err, io.EOF) && !errors.Is(err, errLineTooLong) {
				logWarn("Read stdout error: " + err.Error())
			}
			if errors.Is(err, io.EOF) && line == "" {
				break
			}
		}

		line = strings.TrimSpace(line)
		if line == "" {
			if errors.Is(err, io.EOF) {
				break
			}
			continue
		}

		var event JSONEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			logWarn(fmt.Sprintf("Failed to parse line: %s", truncate(line, 100)))
			if errors.Is(err, io.EOF) {
				break
			}
			continue
		}

		if event.Type == "thread.started" {
			threadID = event.ThreadID
		}

		if event.Type == "item.completed" && event.Item != nil && event.Item.Type == "agent_message" {
			if text := normalizeText(event.Item.Text); text != "" {
				message = text
			}
		}

		if errors.Is(err, io.EOF) {
			break
		}
	}

	return message, threadID
}

var errLineTooLong = errors.New("line too long")

func readLineWithLimit(reader *bufio.Reader) (string, error) {
	var sb strings.Builder

	for {
		chunk, err := reader.ReadString('\n')
		sb.WriteString(chunk)

		if sb.Len() > maxJSONLine {
			// Discard remainder of this line
			for err == bufio.ErrBufferFull {
				_, err = reader.ReadString('\n')
			}
			return "", errLineTooLong
		}

		if err == nil {
			return sb.String(), nil
		}

		if errors.Is(err, io.EOF) {
			return sb.String(), io.EOF
		}

		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}

		return sb.String(), err
	}
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

func streamLiveLog(r io.Reader) {
	reader := bufio.NewReaderSize(r, 128*1024)
	for {
		line, err := readLineWithLimit(reader)
		line = strings.TrimSpace(line)
		if line != "" {
			if formatted := formatEventLine(line); formatted != "" {
				fmt.Fprintf(os.Stderr, "[codex] %s\n", formatted)
			} else {
				fmt.Fprintf(os.Stderr, "[codex-raw] %s\n", line)
			}
		}
		if err != nil {
			break
		}
	}
}

func formatEventLine(line string) string {
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(line), &raw); err != nil {
		return ""
	}

	eventType, _ := raw["type"].(string)
	prefix := map[string]string{
		"item.started":   "开始任务",
		"item.completed": "完成任务",
		"item.updated":   "更新任务",
	}[eventType]
	if prefix == "" {
		prefix = eventType
	}
	switch eventType {
	case "thread.started":
		if tid, ok := raw["thread_id"].(string); ok && tid != "" {
			return fmt.Sprintf("会话启动 | 线程ID: %s", tid)
		}
		return "会话启动"
	case "turn.started":
		return "开始请求 Codex"
	case "turn.failed":
		if errObj, ok := raw["error"].(map[string]interface{}); ok {
			if msg, ok := errObj["message"].(string); ok && msg != "" {
				return fmt.Sprintf("请求失败 | %s", translateErrorMessage(msg))
			}
		}
		return "请求失败"
	case "error":
		if msg, ok := raw["message"].(string); ok && msg != "" {
			return fmt.Sprintf("Codex错误 | %s", translateErrorMessage(msg))
		}
		return "Codex错误"
	case "item.started", "item.completed", "item.updated":
		item, _ := raw["item"].(map[string]interface{})
		if item == nil {
			return prefix
		}
		itemType, _ := item["type"].(string)
		status, _ := item["status"].(string)
		statusLabel := map[string]string{
			"in_progress": "执行中",
			"completed":   "已完成",
			"failed":      "失败",
			"pending":     "待开始",
		}[status]
		if statusLabel == "" && status != "" {
			statusLabel = status
		}
		switch itemType {
		case "agent_message":
			if msg := normalizeText(item["text"]); msg != "" {
				return fmt.Sprintf("%s | 助手输出 (%s)", prefix, msg)
			}
		case "reasoning":
			if msg := normalizeText(item["text"]); msg != "" {
				return fmt.Sprintf("%s | 思路 (%s)", prefix, msg)
			}
		case "command_execution":
			cmd, _ := item["command"].(string)
			var exitStr string
			if exit, ok := item["exit_code"].(float64); ok {
				exitStr = fmt.Sprintf(" [exit=%d]", int(exit))
			}
			if cmd != "" {
				return fmt.Sprintf("%s | 命令 %s (%s)%s", prefix, cmd, statusLabel, exitStr)
			}
		case "file_change":
			if changes, ok := item["changes"].([]interface{}); ok && len(changes) > 0 {
				if c0, ok := changes[0].(map[string]interface{}); ok {
					path, _ := c0["path"].(string)
					kind, _ := c0["kind"].(string)
					if path != "" {
						if kind == "" {
							return fmt.Sprintf("%s | 文件: %s", prefix, path)
						}
						return fmt.Sprintf("%s | 文件 %s %s", prefix, kind, path)
					}
				}
			}
		case "mcp_tool_call":
			server, _ := item["server"].(string)
			tool, _ := item["tool"].(string)
			args := compactValue(item["arguments"], 600)
			out := compactValue(item["aggregated_output"], 600)
			if server != "" || tool != "" {
				var parts []string
				if args != "" {
					parts = append(parts, fmt.Sprintf("args=%s", args))
				}
				if out != "" {
					parts = append(parts, fmt.Sprintf("out=%s", out))
				}
				if len(parts) > 0 {
					return fmt.Sprintf("%s | 工具 %s/%s (%s) %s", prefix, server, tool, statusLabel, strings.Join(parts, " "))
				}
				return fmt.Sprintf("%s | 工具 %s/%s (%s)", prefix, server, tool, statusLabel)
			}
			return fmt.Sprintf("%s | 工具调用", prefix)
		case "todo_list":
			return fmt.Sprintf("%s | 待办更新", prefix)
		}
		if itemType != "" {
			return fmt.Sprintf("%s | %s", prefix, itemType)
		}
		return prefix
	default:
		return fmt.Sprintf("事件: %s", truncate(eventType, 100))
	}
}

func translateErrorMessage(msg string) string {
	if strings.HasPrefix(msg, "Reconnecting... ") {
		return strings.Replace(msg, "Reconnecting...", "重连中", 1)
	}
	if strings.Contains(msg, "Token data is not available") {
		return "令牌数据不可用或未登录"
	}
	return msg
}

func compactValue(v interface{}, limit int) string {
	if v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return truncateLimit(s, limit)
	}

	data, err := json.Marshal(v)
	if err != nil {
		return truncateLimit(fmt.Sprint(v), limit)
	}
	return truncateLimit(string(data), limit)
}

func truncateLimit(s string, limit int) string {
	if limit <= 0 || len(s) <= limit {
		return s
	}
	return s[:limit] + "..."
}

func startPopupLog() (io.WriteCloser, func()) {
	cmd := exec.Command(
		"zenity",
		"--text-info",
		"--title=Codex 实时日志",
		"--width=760",
		"--height=520",
		"--font=monospace",
		"--auto-scroll",
	)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		logWarn(fmt.Sprintf("无法创建弹窗日志管道: %v", err))
		return nil, nil
	}

	if err := cmd.Start(); err != nil {
		logWarn(fmt.Sprintf("弹窗日志启动失败(zenity不可用？): %v", err))
		stdin.Close()
		return nil, nil
	}

	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(done)
	}()

	return stdin, func() {
		<-done
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

func logInfo(msg string) {
	fmt.Fprintf(os.Stderr, "INFO: %s\n", msg)
}

func logWarn(msg string) {
	fmt.Fprintf(os.Stderr, "WARN: %s\n", msg)
}

func logError(msg string) {
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", msg)
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
	CODEX_TIMEOUT   Timeout in milliseconds (default: 7200000)

Exit Codes:
    0    Success
    1    General error (missing args, no output)
    124  Timeout
    127  codex command not found
    130  Interrupted (Ctrl+C)
    *    Passthrough from codex process`
	fmt.Println(help)
}
