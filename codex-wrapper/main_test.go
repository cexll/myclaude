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
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

// Helper to reset test hooks
func resetTestHooks() {
	stdinReader = os.Stdin
	isTerminalFn = defaultIsTerminal
	codexCommand = "codex"
	cleanupHook = nil
	cleanupLogsFn = cleanupOldLogs
	signalNotifyFn = signal.Notify
	signalStopFn = signal.Stop
	buildCodexArgsFn = buildCodexArgs
	commandContext = exec.CommandContext
	jsonMarshal = json.Marshal
	forceKillDelay.Store(5)
	closeLogger()
}

type capturedStdout struct {
	buf    bytes.Buffer
	old    *os.File
	reader *os.File
	writer *os.File
}

type errReader struct {
	err error
}

func (e errReader) Read([]byte) (int, error) {
	return 0, e.err
}

func captureStdoutPipe() *capturedStdout {
	r, w, _ := os.Pipe()
	state := &capturedStdout{old: os.Stdout, reader: r, writer: w}
	os.Stdout = w
	return state
}

func restoreStdoutPipe(c *capturedStdout) {
	if c == nil {
		return
	}
	c.writer.Close()
	os.Stdout = c.old
	io.Copy(&c.buf, c.reader)
}

func (c *capturedStdout) String() string {
	if c == nil {
		return ""
	}
	return c.buf.String()
}

func captureOutput(t *testing.T, fn func()) string {
	t.Helper()
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, _ := os.Pipe()
	old := os.Stderr
	os.Stderr = w
	fn()
	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func createFakeCodexScript(t *testing.T, threadID, message string) string {
	t.Helper()
	scriptPath := filepath.Join(t.TempDir(), "codex.sh")
	script := fmt.Sprintf(`#!/bin/sh
printf '%%s\n' '{"type":"thread.started","thread_id":"%s"}'
printf '%%s\n' '{"type":"item.completed","item":{"type":"agent_message","text":"%s"}}'
`, threadID, message)
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to create fake codex script: %v", err)
	}
	return scriptPath
}

func TestRunParseArgs_NewMode(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    *Config
		wantErr bool
	}{
		{
			name: "simple task",
			args: []string{"codex-wrapper", "analyze code"},
			want: &Config{Mode: "new", Task: "analyze code", WorkDir: ".", ExplicitStdin: false},
		},
		{
			name: "task with workdir",
			args: []string{"codex-wrapper", "analyze code", "/path/to/dir"},
			want: &Config{Mode: "new", Task: "analyze code", WorkDir: "/path/to/dir", ExplicitStdin: false},
		},
		{
			name: "explicit stdin mode",
			args: []string{"codex-wrapper", "-"},
			want: &Config{Mode: "new", Task: "-", WorkDir: ".", ExplicitStdin: true},
		},
		{
			name: "stdin with workdir",
			args: []string{"codex-wrapper", "-", "/some/dir"},
			want: &Config{Mode: "new", Task: "-", WorkDir: "/some/dir", ExplicitStdin: true},
		},
		{name: "no args", args: []string{"codex-wrapper"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args
			cfg, err := parseArgs()
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseArgs() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseArgs() unexpected error: %v", err)
			}
			if cfg.Mode != tt.want.Mode {
				t.Errorf("Mode = %v, want %v", cfg.Mode, tt.want.Mode)
			}
			if cfg.Task != tt.want.Task {
				t.Errorf("Task = %v, want %v", cfg.Task, tt.want.Task)
			}
			if cfg.WorkDir != tt.want.WorkDir {
				t.Errorf("WorkDir = %v, want %v", cfg.WorkDir, tt.want.WorkDir)
			}
			if cfg.ExplicitStdin != tt.want.ExplicitStdin {
				t.Errorf("ExplicitStdin = %v, want %v", cfg.ExplicitStdin, tt.want.ExplicitStdin)
			}
		})
	}
}

func TestRunParseArgs_ResumeMode(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    *Config
		wantErr bool
	}{
		{
			name: "resume with task",
			args: []string{"codex-wrapper", "resume", "session-123", "continue task"},
			want: &Config{Mode: "resume", SessionID: "session-123", Task: "continue task", WorkDir: ".", ExplicitStdin: false},
		},
		{
			name: "resume with workdir",
			args: []string{"codex-wrapper", "resume", "session-456", "task", "/work"},
			want: &Config{Mode: "resume", SessionID: "session-456", Task: "task", WorkDir: "/work", ExplicitStdin: false},
		},
		{
			name: "resume with stdin",
			args: []string{"codex-wrapper", "resume", "session-789", "-"},
			want: &Config{Mode: "resume", SessionID: "session-789", Task: "-", WorkDir: ".", ExplicitStdin: true},
		},
		{name: "resume missing session_id", args: []string{"codex-wrapper", "resume"}, wantErr: true},
		{name: "resume missing task", args: []string{"codex-wrapper", "resume", "session-123"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args
			cfg, err := parseArgs()
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseArgs() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseArgs() unexpected error: %v", err)
			}
			if cfg.Mode != tt.want.Mode || cfg.SessionID != tt.want.SessionID || cfg.Task != tt.want.Task || cfg.WorkDir != tt.want.WorkDir || cfg.ExplicitStdin != tt.want.ExplicitStdin {
				t.Errorf("parseArgs() mismatch: %+v vs %+v", cfg, tt.want)
			}
		})
	}
}

func TestRunParseParallelConfig_Success(t *testing.T) {
	input := `---TASK---
id: task-1
dependencies: task-0
---CONTENT---
do something`

	cfg, err := parseParallelConfig([]byte(input))
	if err != nil {
		t.Fatalf("parseParallelConfig() unexpected error: %v", err)
	}
	if len(cfg.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(cfg.Tasks))
	}
	task := cfg.Tasks[0]
	if task.ID != "task-1" || task.Task != "do something" || task.WorkDir != defaultWorkdir || len(task.Dependencies) != 1 || task.Dependencies[0] != "task-0" {
		t.Fatalf("task mismatch: %+v", task)
	}
}

func TestRunParseParallelConfig_InvalidFormat(t *testing.T) {
	if _, err := parseParallelConfig([]byte("invalid format")); err == nil {
		t.Fatalf("expected error for invalid format, got nil")
	}
}

func TestRunParseParallelConfig_EmptyTasks(t *testing.T) {
	input := `---TASK---
id: empty
---CONTENT---
`
	if _, err := parseParallelConfig([]byte(input)); err == nil {
		t.Fatalf("expected error for empty tasks array, got nil")
	}
}

func TestRunParseParallelConfig_MissingID(t *testing.T) {
	input := `---TASK---
---CONTENT---
do something`
	if _, err := parseParallelConfig([]byte(input)); err == nil {
		t.Fatalf("expected error for missing id, got nil")
	}
}

func TestRunParseParallelConfig_MissingTask(t *testing.T) {
	input := `---TASK---
id: task-1
---CONTENT---
`
	if _, err := parseParallelConfig([]byte(input)); err == nil {
		t.Fatalf("expected error for missing task, got nil")
	}
}

func TestRunParseParallelConfig_DuplicateID(t *testing.T) {
	input := `---TASK---
id: dup
---CONTENT---
one
---TASK---
id: dup
---CONTENT---
two`
	if _, err := parseParallelConfig([]byte(input)); err == nil {
		t.Fatalf("expected error for duplicate id, got nil")
	}
}

func TestRunParseParallelConfig_DelimiterFormat(t *testing.T) {
	input := `---TASK---
id: T1
workdir: /tmp
---CONTENT---
echo 'test'
---TASK---
id: T2
dependencies: T1
---CONTENT---
code with special chars: $var "quotes"`

	cfg, err := parseParallelConfig([]byte(input))
	if err != nil {
		t.Fatalf("parseParallelConfig() error = %v", err)
	}
	if len(cfg.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(cfg.Tasks))
	}
}

func TestRunShouldUseStdin(t *testing.T) {
	tests := []struct {
		name  string
		task  string
		piped bool
		want  bool
	}{
		{"simple task", "analyze code", false, false},
		{"piped input", "analyze code", true, true},
		{"contains newline", "line1\nline2", false, true},
		{"contains backslash", "path\\to\\file", false, true},
		{"contains double quote", `say "hi"`, false, true},
		{"contains single quote", "it's tricky", false, true},
		{"contains backtick", "use `code`", false, true},
		{"contains dollar", "price is $5", false, true},
		{"long task", strings.Repeat("a", 801), false, true},
		{"exactly 800 chars", strings.Repeat("a", 800), false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldUseStdin(tt.task, tt.piped)
			if got != tt.want {
				t.Errorf("shouldUseStdin(%q, %v) = %v, want %v", truncate(tt.task, 20), tt.piped, got, tt.want)
			}
		})
	}
}

func TestRunBuildCodexArgs_NewMode(t *testing.T) {
	cfg := &Config{Mode: "new", WorkDir: "/test/dir"}
	args := buildCodexArgs(cfg, "my task")
	expected := []string{"e", "--skip-git-repo-check", "-C", "/test/dir", "--json", "my task"}
	if len(args) != len(expected) {
		t.Fatalf("len mismatch")
	}
	for i := range args {
		if args[i] != expected[i] {
			t.Fatalf("args[%d]=%s, want %s", i, args[i], expected[i])
		}
	}
}

func TestRunBuildCodexArgs_ResumeMode(t *testing.T) {
	cfg := &Config{Mode: "resume", SessionID: "session-abc"}
	args := buildCodexArgs(cfg, "-")
	expected := []string{"e", "--skip-git-repo-check", "--json", "resume", "session-abc", "-"}
	if len(args) != len(expected) {
		t.Fatalf("len mismatch")
	}
	for i := range args {
		if args[i] != expected[i] {
			t.Fatalf("args[%d]=%s, want %s", i, args[i], expected[i])
		}
	}
}

func TestRunResolveTimeout(t *testing.T) {
	tests := []struct {
		name   string
		envVal string
		want   int
	}{
		{"empty env", "", 7200},
		{"milliseconds", "7200000", 7200},
		{"seconds", "3600", 3600},
		{"invalid", "invalid", 7200},
		{"negative", "-100", 7200},
		{"zero", "0", 7200},
		{"small milliseconds", "5000", 5000},
		{"boundary", "10000", 10000},
		{"above boundary", "10001", 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("CODEX_TIMEOUT", tt.envVal)
			defer os.Unsetenv("CODEX_TIMEOUT")
			got := resolveTimeout()
			if got != tt.want {
				t.Errorf("resolveTimeout() with env=%q = %v, want %v", tt.envVal, got, tt.want)
			}
		})
	}
}

func TestRunNormalizeText(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  string
	}{
		{"string", "hello world", "hello world"},
		{"string array", []interface{}{"hello", " ", "world"}, "hello world"},
		{"empty array", []interface{}{}, ""},
		{"mixed array", []interface{}{"text", 123, "more"}, "textmore"},
		{"nil", nil, ""},
		{"number", 123, ""},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeText(tt.input)
			if got != tt.want {
				t.Errorf("normalizeText(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRunParseJSONStream(t *testing.T) {
	type testCase struct {
		name         string
		input        string
		wantMessage  string
		wantThreadID string
	}

	longText := strings.Repeat("a", 2*1024*1024)

	tests := []testCase{
		{"thread started and agent message", `{"type":"thread.started","thread_id":"abc-123"}
{"type":"item.completed","item":{"type":"agent_message","text":"Hello world"}}`, "Hello world", "abc-123"},
		{"multiple agent messages", `{"type":"item.completed","item":{"type":"agent_message","text":"First"}}
{"type":"item.completed","item":{"type":"agent_message","text":"Second"}}`, "Second", ""},
		{"text as array", `{"type":"item.completed","item":{"type":"agent_message","text":["Hello"," ","World"]}}`, "Hello World", ""},
		{"ignore other event types", `{"type":"other.event","data":"ignored"}
{"type":"item.completed","item":{"type":"other_type","text":"ignored"}}
{"type":"item.completed","item":{"type":"agent_message","text":"Valid"}}`, "Valid", ""},
		{"super long single line", `{"type":"item.completed","item":{"type":"agent_message","text":"` + longText + `"}}`, longText, ""},
		{"empty input", "", "", ""},
		{"item completed with nil item", strings.Join([]string{`{"type":"thread.started","thread_id":"nil-item-thread"}`, `{"type":"item.completed","item":null}`}, "\n"), "", "nil-item-thread"},
		{"agent message with non-string text", `{"type":"item.completed","item":{"type":"agent_message","text":12345}}`, "", ""},
		{"corrupted json does not break stream", strings.Join([]string{`{"type":"item.completed","item":{"type":"agent_message","text":"before"}}`, `{"type":"item.completed","item":{"type":"agent_message","text":"broken"}`, `{"type":"thread.started","thread_id":"after-thread"}`, `{"type":"item.completed","item":{"type":"agent_message","text":"after"}}`}, "\n"), "after", "after-thread"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMessage, gotThreadID := parseJSONStream(strings.NewReader(tt.input))
			if gotMessage != tt.wantMessage {
				t.Errorf("message = %q, want %q", gotMessage, tt.wantMessage)
			}
			if gotThreadID != tt.wantThreadID {
				t.Errorf("threadID = %q, want %q", gotThreadID, tt.wantThreadID)
			}
		})
	}
}

func TestRunParseJSONStreamWithWarn_InvalidLine(t *testing.T) {
	var warnings []string
	warnFn := func(msg string) { warnings = append(warnings, msg) }
	message, threadID := parseJSONStreamWithWarn(strings.NewReader("not-json"), warnFn)
	if message != "" || threadID != "" {
		t.Fatalf("expected empty output, got message=%q thread=%q", message, threadID)
	}
	if len(warnings) == 0 {
		t.Fatalf("expected warning to be emitted")
	}
}

func TestRunGetEnv(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		defaultVal string
		envVal     string
		setEnv     bool
		want       string
	}{
		{"env set", "TEST_KEY", "default", "custom", true, "custom"},
		{"env not set", "TEST_KEY_MISSING", "default", "", false, "default"},
		{"env empty", "TEST_KEY_EMPTY", "default", "", true, "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv(tt.key)
			if tt.setEnv {
				os.Setenv(tt.key, tt.envVal)
				defer os.Unsetenv(tt.key)
			}

			got := getEnv(tt.key, tt.defaultVal)
			if got != tt.want {
				t.Errorf("getEnv(%q, %q) = %q, want %q", tt.key, tt.defaultVal, got, tt.want)
			}
		})
	}
}

func TestRunTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"truncate", "hello world", 5, "hello..."},
		{"empty", "", 5, ""},
		{"zero maxLen", "hello", 0, "..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}

	if got := truncate("data", -1); got != "" {
		t.Fatalf("truncate should return empty string for negative maxLen, got %q", got)
	}
}

func TestRunMin(t *testing.T) {
	tests := []struct {
		a, b, want int
	}{{1, 2, 1}, {2, 1, 1}, {5, 5, 5}, {-1, 0, -1}, {0, -1, -1}}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := min(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestRunLogFunctions(t *testing.T) {
	defer resetTestHooks()
	tempDir := t.TempDir()
	t.Setenv("TMPDIR", tempDir)

	logger, err := NewLogger()
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	setLogger(logger)
	defer closeLogger()

	logInfo("info message")
	logWarn("warn message")
	logError("error message")
	logger.Flush()

	data, err := os.ReadFile(logger.Path())
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	output := string(data)
	if !strings.Contains(output, "INFO: info message") {
		t.Errorf("logInfo output missing, got: %s", output)
	}
	if !strings.Contains(output, "WARN: warn message") {
		t.Errorf("logWarn output missing, got: %s", output)
	}
	if !strings.Contains(output, "ERROR: error message") {
		t.Errorf("logError output missing, got: %s", output)
	}
}

func TestRunPrintHelp(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	printHelp()
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	expected := []string{"codex-wrapper", "Usage:", "resume", "CODEX_TIMEOUT", "Exit Codes:"}
	for _, phrase := range expected {
		if !strings.Contains(output, phrase) {
			t.Errorf("printHelp() missing phrase %q", phrase)
		}
	}
}

func TestRunIsTerminal(t *testing.T) {
	defer resetTestHooks()
	tests := []struct {
		name   string
		mockFn func() bool
		want   bool
	}{{"is terminal", func() bool { return true }, true}, {"is not terminal", func() bool { return false }, false}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isTerminalFn = tt.mockFn
			got := isTerminal()
			if got != tt.want {
				t.Errorf("isTerminal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRunReadPipedTask(t *testing.T) {
	defer resetTestHooks()
	tests := []struct {
		name       string
		isTerminal bool
		stdin      io.Reader
		want       string
		wantErr    bool
	}{
		{"terminal mode", true, strings.NewReader("ignored"), "", false},
		{"piped with data", false, strings.NewReader("task from pipe"), "task from pipe", false},
		{"piped empty", false, strings.NewReader(""), "", false},
		{"piped read error", false, errReader{errors.New("boom")}, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isTerminalFn = func() bool { return tt.isTerminal }
			stdinReader = tt.stdin
			got, err := readPipedTask()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("readPipedTask() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRunCodexTask_CommandNotFound(t *testing.T) {
	defer resetTestHooks()
	codexCommand = "nonexistent-command-xyz"
	buildCodexArgsFn = func(cfg *Config, targetArg string) []string { return []string{targetArg} }
	res := runCodexTask(TaskSpec{Task: "task"}, false, 10)
	if res.ExitCode != 127 {
		t.Errorf("exitCode = %d, want 127", res.ExitCode)
	}
	if res.Error == "" {
		t.Errorf("expected error message")
	}
}

func TestRunCodexTask_StartError(t *testing.T) {
	defer resetTestHooks()
	tmpFile, err := os.CreateTemp("", "start-error")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	codexCommand = tmpFile.Name()
	buildCodexArgsFn = func(cfg *Config, targetArg string) []string { return []string{} }

	res := runCodexTask(TaskSpec{Task: "task"}, false, 1)
	if res.ExitCode != 1 || !strings.Contains(res.Error, "failed to start codex") {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestRunCodexTask_WithEcho(t *testing.T) {
	defer resetTestHooks()
	codexCommand = "echo"
	buildCodexArgsFn = func(cfg *Config, targetArg string) []string { return []string{targetArg} }

	jsonOutput := `{"type":"thread.started","thread_id":"test-session"}
{"type":"item.completed","item":{"type":"agent_message","text":"Test output"}}`

	res := runCodexTask(TaskSpec{Task: jsonOutput}, false, 10)
	if res.ExitCode != 0 || res.Message != "Test output" || res.SessionID != "test-session" {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestRunCodexTask_NoMessage(t *testing.T) {
	defer resetTestHooks()
	codexCommand = "echo"
	buildCodexArgsFn = func(cfg *Config, targetArg string) []string { return []string{targetArg} }
	jsonOutput := `{"type":"thread.started","thread_id":"test-session"}`
	res := runCodexTask(TaskSpec{Task: jsonOutput}, false, 10)
	if res.ExitCode != 1 || res.Error == "" {
		t.Fatalf("expected error for missing agent_message, got %+v", res)
	}
}

func TestRunCodexTask_WithStdin(t *testing.T) {
	defer resetTestHooks()
	codexCommand = "cat"
	buildCodexArgsFn = func(cfg *Config, targetArg string) []string { return []string{} }
	jsonInput := `{"type":"item.completed","item":{"type":"agent_message","text":"from stdin"}}`
	res := runCodexTask(TaskSpec{Task: jsonInput, UseStdin: true}, false, 10)
	if res.ExitCode != 0 || res.Message != "from stdin" {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestRunCodexTask_ExitError(t *testing.T) {
	defer resetTestHooks()
	codexCommand = "false"
	buildCodexArgsFn = func(cfg *Config, targetArg string) []string { return []string{} }
	res := runCodexTask(TaskSpec{Task: "noop"}, false, 10)
	if res.ExitCode == 0 || res.Error == "" {
		t.Fatalf("expected failure, got %+v", res)
	}
}

func TestRunCodexTask_StdinPipeError(t *testing.T) {
	defer resetTestHooks()
	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		cmd := exec.CommandContext(ctx, "cat")
		cmd.Stdin = os.Stdin
		return cmd
	}
	buildCodexArgsFn = func(cfg *Config, targetArg string) []string { return []string{} }
	res := runCodexTask(TaskSpec{Task: "data", UseStdin: true}, false, 1)
	if res.ExitCode != 1 || !strings.Contains(res.Error, "stdin pipe") {
		t.Fatalf("expected stdin pipe error, got %+v", res)
	}
}

func TestRunCodexTask_StdoutPipeError(t *testing.T) {
	defer resetTestHooks()
	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		cmd := exec.CommandContext(ctx, "echo", "noop")
		cmd.Stdout = os.Stdout
		return cmd
	}
	buildCodexArgsFn = func(cfg *Config, targetArg string) []string { return []string{} }
	res := runCodexTask(TaskSpec{Task: "noop"}, false, 1)
	if res.ExitCode != 1 || !strings.Contains(res.Error, "stdout pipe") {
		t.Fatalf("expected stdout pipe error, got %+v", res)
	}
}

func TestRunCodexTask_Timeout(t *testing.T) {
	defer resetTestHooks()
	codexCommand = "sleep"
	buildCodexArgsFn = func(cfg *Config, targetArg string) []string { return []string{"2"} }
	res := runCodexTask(TaskSpec{Task: "ignored"}, false, 1)
	if res.ExitCode != 124 || !strings.Contains(res.Error, "timeout") {
		t.Fatalf("expected timeout, got %+v", res)
	}
}

func TestRunCodexTask_SignalHandling(t *testing.T) {
	defer resetTestHooks()
	codexCommand = "sleep"
	buildCodexArgsFn = func(cfg *Config, targetArg string) []string { return []string{"5"} }

	resultCh := make(chan TaskResult, 1)
	go func() { resultCh <- runCodexTask(TaskSpec{Task: "ignored"}, false, 5) }()

	time.Sleep(200 * time.Millisecond)
	syscall.Kill(os.Getpid(), syscall.SIGTERM)

	res := <-resultCh
	signal.Reset(syscall.SIGINT, syscall.SIGTERM)

	if res.ExitCode == 0 || res.Error == "" {
		t.Fatalf("expected non-zero exit after signal, got %+v", res)
	}
}

func TestRunCodexProcess(t *testing.T) {
	defer resetTestHooks()
	script := createFakeCodexScript(t, "proc-thread", "proc-msg")
	codexCommand = script

	msg, threadID, exitCode := runCodexProcess(context.Background(), nil, "ignored", false, 5)
	if exitCode != 0 {
		t.Fatalf("exit = %d, want 0", exitCode)
	}
	if msg != "proc-msg" {
		t.Fatalf("message = %q, want proc-msg", msg)
	}
	if threadID != "proc-thread" {
		t.Fatalf("threadID = %q, want proc-thread", threadID)
	}
}

func TestRunSilentMode(t *testing.T) {
	defer resetTestHooks()
	jsonOutput := `{"type":"thread.started","thread_id":"silent-session"}
{"type":"item.completed","item":{"type":"agent_message","text":"quiet"}}`
	codexCommand = "echo"
	buildCodexArgsFn = func(cfg *Config, targetArg string) []string { return []string{targetArg} }

	capture := func(silent bool) string {
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w
		res := runCodexTask(TaskSpec{Task: jsonOutput}, silent, 10)
		if res.ExitCode != 0 {
			t.Fatalf("unexpected exitCode %d", res.ExitCode)
		}
		w.Close()
		os.Stderr = oldStderr
		var buf bytes.Buffer
		io.Copy(&buf, r)
		return buf.String()
	}

	verbose := capture(false)
	quiet := capture(true)

	// After refactoring, logs are only written to file, not stderr
	// Both silent and non-silent modes should produce no stderr output
	if quiet != "" {
		t.Fatalf("silent mode should suppress stderr, got: %q", quiet)
	}
	if verbose != "" {
		t.Fatalf("non-silent mode should also suppress stderr (logs go to file), got: %q", verbose)
	}
}

func TestRunGenerateFinalOutput(t *testing.T) {
	results := []TaskResult{{TaskID: "a", ExitCode: 0, Message: "ok"}, {TaskID: "b", ExitCode: 1, Error: "boom"}, {TaskID: "c", ExitCode: 0}}
	out := generateFinalOutput(results)
	if out == "" {
		t.Fatalf("generateFinalOutput() returned empty string")
	}
	if !strings.Contains(out, "Total: 3") || !strings.Contains(out, "Success: 2") || !strings.Contains(out, "Failed: 1") {
		t.Fatalf("summary missing, got %q", out)
	}
	if !strings.Contains(out, "Task: a") || !strings.Contains(out, "Task: b") {
		t.Fatalf("task entries missing")
	}
}

func TestRunTopologicalSort_LinearChain(t *testing.T) {
	tasks := []TaskSpec{{ID: "a"}, {ID: "b", Dependencies: []string{"a"}}, {ID: "c", Dependencies: []string{"b"}}}
	layers, err := topologicalSort(tasks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(layers) != 3 {
		t.Fatalf("expected 3 layers, got %d", len(layers))
	}
}

func TestRunTopologicalSort_Branching(t *testing.T) {
	tasks := []TaskSpec{{ID: "root"}, {ID: "left", Dependencies: []string{"root"}}, {ID: "right", Dependencies: []string{"root"}}, {ID: "leaf", Dependencies: []string{"left", "right"}}}
	layers, err := topologicalSort(tasks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(layers) != 3 || len(layers[1]) != 2 {
		t.Fatalf("unexpected layers: %+v", layers)
	}
}

func TestRunTopologicalSort_ParallelTasks(t *testing.T) {
	tasks := []TaskSpec{{ID: "a"}, {ID: "b"}, {ID: "c"}}
	layers, err := topologicalSort(tasks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(layers) != 1 || len(layers[0]) != 3 {
		t.Fatalf("unexpected result: %+v", layers)
	}
}

func TestRunShouldSkipTask(t *testing.T) {
	failed := map[string]TaskResult{"a": {TaskID: "a", ExitCode: 1}, "b": {TaskID: "b", ExitCode: 2}}
	tests := []struct {
		name           string
		task           TaskSpec
		skip           bool
		reasonContains []string
	}{
		{"no deps", TaskSpec{ID: "c"}, false, nil},
		{"missing deps not failed", TaskSpec{ID: "d", Dependencies: []string{"x"}}, false, nil},
		{"single failed dep", TaskSpec{ID: "e", Dependencies: []string{"a"}}, true, []string{"a"}},
		{"multiple failed deps", TaskSpec{ID: "f", Dependencies: []string{"a", "b"}}, true, []string{"a", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skip, reason := shouldSkipTask(tt.task, failed)
			if skip != tt.skip {
				t.Fatalf("skip=%v, want %v", skip, tt.skip)
			}
			for _, expect := range tt.reasonContains {
				if !strings.Contains(reason, expect) {
					t.Fatalf("reason %q missing %q", reason, expect)
				}
			}
		})
	}
}

func TestRunTopologicalSort_CycleDetection(t *testing.T) {
	tasks := []TaskSpec{{ID: "a", Dependencies: []string{"b"}}, {ID: "b", Dependencies: []string{"a"}}}
	if _, err := topologicalSort(tasks); err == nil || !strings.Contains(err.Error(), "cycle detected") {
		t.Fatalf("expected cycle error, got %v", err)
	}
}

func TestRunTopologicalSort_IndirectCycle(t *testing.T) {
	tasks := []TaskSpec{{ID: "a", Dependencies: []string{"c"}}, {ID: "b", Dependencies: []string{"a"}}, {ID: "c", Dependencies: []string{"b"}}}
	if _, err := topologicalSort(tasks); err == nil || !strings.Contains(err.Error(), "cycle detected") {
		t.Fatalf("expected cycle error, got %v", err)
	}
}

func TestRunTopologicalSort_MissingDependency(t *testing.T) {
	tasks := []TaskSpec{{ID: "a", Dependencies: []string{"missing"}}}
	if _, err := topologicalSort(tasks); err == nil || !strings.Contains(err.Error(), "dependency \"missing\" not found") {
		t.Fatalf("expected missing dependency error, got %v", err)
	}
}

func TestRunTopologicalSort_LargeGraph(t *testing.T) {
	const count = 200
	tasks := make([]TaskSpec, count)
	for i := 0; i < count; i++ {
		id := fmt.Sprintf("task-%d", i)
		if i == 0 {
			tasks[i] = TaskSpec{ID: id}
			continue
		}
		prev := fmt.Sprintf("task-%d", i-1)
		tasks[i] = TaskSpec{ID: id, Dependencies: []string{prev}}
	}

	layers, err := topologicalSort(tasks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(layers) != count {
		t.Fatalf("expected %d layers, got %d", count, len(layers))
	}
}

func TestRunExecuteConcurrent_ParallelExecution(t *testing.T) {
	orig := runCodexTaskFn
	defer func() { runCodexTaskFn = orig }()

	var maxParallel int64
	var current int64

	runCodexTaskFn = func(task TaskSpec, timeout int) TaskResult {
		cur := atomic.AddInt64(&current, 1)
		for {
			prev := atomic.LoadInt64(&maxParallel)
			if cur <= prev || atomic.CompareAndSwapInt64(&maxParallel, prev, cur) {
				break
			}
		}
		time.Sleep(150 * time.Millisecond)
		atomic.AddInt64(&current, -1)
		return TaskResult{TaskID: task.ID}
	}

	start := time.Now()
	layers := [][]TaskSpec{{{ID: "a"}, {ID: "b"}, {ID: "c"}}}
	results := executeConcurrent(layers, 10)
	elapsed := time.Since(start)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if elapsed >= 400*time.Millisecond {
		t.Fatalf("expected concurrent execution, took %v", elapsed)
	}
	if maxParallel < 2 {
		t.Fatalf("expected parallelism >=2, got %d", maxParallel)
	}
}

func TestRunExecuteConcurrent_LayerOrdering(t *testing.T) {
	orig := runCodexTaskFn
	defer func() { runCodexTaskFn = orig }()

	var mu sync.Mutex
	var order []string

	runCodexTaskFn = func(task TaskSpec, timeout int) TaskResult {
		mu.Lock()
		order = append(order, task.ID)
		mu.Unlock()
		return TaskResult{TaskID: task.ID}
	}

	layers := [][]TaskSpec{{{ID: "first-1"}, {ID: "first-2"}}, {{ID: "second"}}}
	executeConcurrent(layers, 10)

	if len(order) != 3 || order[2] != "second" {
		t.Fatalf("unexpected order: %+v", order)
	}
}

func TestRunExecuteConcurrent_ErrorIsolation(t *testing.T) {
	orig := runCodexTaskFn
	defer func() { runCodexTaskFn = orig }()

	runCodexTaskFn = func(task TaskSpec, timeout int) TaskResult {
		if task.ID == "fail" {
			return TaskResult{TaskID: task.ID, ExitCode: 2, Error: "boom"}
		}
		return TaskResult{TaskID: task.ID, ExitCode: 0}
	}

	layers := [][]TaskSpec{{{ID: "ok"}, {ID: "fail"}}, {{ID: "after"}}}
	results := executeConcurrent(layers, 10)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	var failed, succeeded bool
	for _, res := range results {
		if res.TaskID == "fail" && res.ExitCode == 2 {
			failed = true
		}
		if res.TaskID == "after" && res.ExitCode == 0 {
			succeeded = true
		}
	}

	if !failed || !succeeded {
		t.Fatalf("expected failure isolation, got %+v", results)
	}
}

func TestRunExecuteConcurrent_PanicRecovered(t *testing.T) {
	orig := runCodexTaskFn
	defer func() { runCodexTaskFn = orig }()

	runCodexTaskFn = func(task TaskSpec, timeout int) TaskResult {
		panic("boom")
	}

	results := executeConcurrent([][]TaskSpec{{{ID: "panic"}}}, 10)
	if len(results) != 1 || results[0].Error == "" || results[0].ExitCode == 0 {
		t.Fatalf("panic should be captured, got %+v", results[0])
	}
}

func TestRunExecuteConcurrent_LargeFanout(t *testing.T) {
	orig := runCodexTaskFn
	defer func() { runCodexTaskFn = orig }()

	runCodexTaskFn = func(task TaskSpec, timeout int) TaskResult { return TaskResult{TaskID: task.ID} }
	layer := make([]TaskSpec, 0, 1200)
	for i := 0; i < 1200; i++ {
		layer = append(layer, TaskSpec{ID: fmt.Sprintf("id-%d", i)})
	}
	results := executeConcurrent([][]TaskSpec{layer}, 10)
	if len(results) != 1200 {
		t.Fatalf("expected 1200 results, got %d", len(results))
	}
}

func TestRun_ParallelFlag(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"codex-wrapper", "--parallel"}
	jsonInput := `---TASK---
id: T1
---CONTENT---
test`
	stdinReader = strings.NewReader(jsonInput)
	defer func() { stdinReader = os.Stdin }()

	runCodexTaskFn = func(task TaskSpec, timeout int) TaskResult {
		return TaskResult{TaskID: task.ID, ExitCode: 0, Message: "test output"}
	}
	defer func() {
		runCodexTaskFn = func(task TaskSpec, timeout int) TaskResult { return runCodexTask(task, true, timeout) }
	}()

	exitCode := run()
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
}

func TestRun_ParallelTriggersCleanup(t *testing.T) {
	defer resetTestHooks()
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"codex-wrapper", "--parallel"}
	stdinReader = strings.NewReader(`---TASK---
id: only
---CONTENT---
noop`)

	cleanupCalls := 0
	cleanupLogsFn = func() (CleanupStats, error) {
		cleanupCalls++
		return CleanupStats{}, nil
	}

	orig := runCodexTaskFn
	runCodexTaskFn = func(task TaskSpec, timeout int) TaskResult {
		return TaskResult{TaskID: task.ID, ExitCode: 0, Message: "ok"}
	}
	defer func() { runCodexTaskFn = orig }()

	if exitCode := run(); exitCode != 0 {
		t.Fatalf("exit = %d, want 0", exitCode)
	}
	if cleanupCalls != 1 {
		t.Fatalf("cleanup called %d times, want 1", cleanupCalls)
	}
}

func TestRun_Version(t *testing.T) {
	defer resetTestHooks()
	os.Args = []string{"codex-wrapper", "--version"}
	if code := run(); code != 0 {
		t.Errorf("exit = %d, want 0", code)
	}
}

func TestRun_VersionShort(t *testing.T) {
	defer resetTestHooks()
	os.Args = []string{"codex-wrapper", "-v"}
	if code := run(); code != 0 {
		t.Errorf("exit = %d, want 0", code)
	}
}

func TestRun_Help(t *testing.T) {
	defer resetTestHooks()
	os.Args = []string{"codex-wrapper", "--help"}
	if code := run(); code != 0 {
		t.Errorf("exit = %d, want 0", code)
	}
}

func TestRun_HelpShort(t *testing.T) {
	defer resetTestHooks()
	os.Args = []string{"codex-wrapper", "-h"}
	if code := run(); code != 0 {
		t.Errorf("exit = %d, want 0", code)
	}
}

func TestRun_HelpDoesNotTriggerCleanup(t *testing.T) {
	defer resetTestHooks()
	os.Args = []string{"codex-wrapper", "--help"}
	cleanupLogsFn = func() (CleanupStats, error) {
		t.Fatalf("cleanup should not run for --help")
		return CleanupStats{}, nil
	}

	if code := run(); code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
}

func TestRun_VersionDoesNotTriggerCleanup(t *testing.T) {
	defer resetTestHooks()
	os.Args = []string{"codex-wrapper", "--version"}
	cleanupLogsFn = func() (CleanupStats, error) {
		t.Fatalf("cleanup should not run for --version")
		return CleanupStats{}, nil
	}

	if code := run(); code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
}

func TestRunCleanupMode_Success(t *testing.T) {
	defer resetTestHooks()
	cleanupLogsFn = func() (CleanupStats, error) {
		return CleanupStats{
			Scanned:      5,
			Deleted:      3,
			Kept:         2,
			DeletedFiles: []string{"codex-wrapper-111.log", "codex-wrapper-222.log", "codex-wrapper-333.log"},
			KeptFiles:    []string{"codex-wrapper-444.log", "codex-wrapper-555.log"},
		}, nil
	}

	var exitCode int
	output := captureOutput(t, func() {
		exitCode = runCleanupMode()
	})
	if exitCode != 0 {
		t.Fatalf("exit = %d, want 0", exitCode)
	}
	want := "Cleanup completed\nFiles scanned: 5\nFiles deleted: 3\n  - codex-wrapper-111.log\n  - codex-wrapper-222.log\n  - codex-wrapper-333.log\nFiles kept: 2\n  - codex-wrapper-444.log\n  - codex-wrapper-555.log\n"
	if output != want {
		t.Fatalf("output = %q, want %q", output, want)
	}
}

func TestRunCleanupMode_SuccessWithErrorsLine(t *testing.T) {
	defer resetTestHooks()
	cleanupLogsFn = func() (CleanupStats, error) {
		return CleanupStats{
			Scanned:      2,
			Deleted:      1,
			Kept:         0,
			Errors:       1,
			DeletedFiles: []string{"codex-wrapper-123.log"},
		}, nil
	}

	var exitCode int
	output := captureOutput(t, func() {
		exitCode = runCleanupMode()
	})
	if exitCode != 0 {
		t.Fatalf("exit = %d, want 0", exitCode)
	}
	want := "Cleanup completed\nFiles scanned: 2\nFiles deleted: 1\n  - codex-wrapper-123.log\nFiles kept: 0\nDeletion errors: 1\n"
	if output != want {
		t.Fatalf("output = %q, want %q", output, want)
	}
}

func TestRunCleanupMode_ZeroStatsOutput(t *testing.T) {
	defer resetTestHooks()
	calls := 0
	cleanupLogsFn = func() (CleanupStats, error) {
		calls++
		return CleanupStats{}, nil
	}

	var exitCode int
	output := captureOutput(t, func() {
		exitCode = runCleanupMode()
	})
	if exitCode != 0 {
		t.Fatalf("exit = %d, want 0", exitCode)
	}
	want := "Cleanup completed\nFiles scanned: 0\nFiles deleted: 0\nFiles kept: 0\n"
	if output != want {
		t.Fatalf("output = %q, want %q", output, want)
	}
	if calls != 1 {
		t.Fatalf("cleanup called %d times, want 1", calls)
	}
}

func TestRunCleanupMode_Error(t *testing.T) {
	defer resetTestHooks()
	cleanupLogsFn = func() (CleanupStats, error) {
		return CleanupStats{}, fmt.Errorf("boom")
	}

	var exitCode int
	errOutput := captureStderr(t, func() {
		exitCode = runCleanupMode()
	})
	if exitCode != 1 {
		t.Fatalf("exit = %d, want 1", exitCode)
	}
	if !strings.Contains(errOutput, "Cleanup failed") || !strings.Contains(errOutput, "boom") {
		t.Fatalf("stderr = %q, want error message", errOutput)
	}
}

func TestRunCleanupMode_MissingFn(t *testing.T) {
	defer resetTestHooks()
	cleanupLogsFn = nil

	var exitCode int
	errOutput := captureStderr(t, func() {
		exitCode = runCleanupMode()
	})
	if exitCode != 1 {
		t.Fatalf("exit = %d, want 1", exitCode)
	}
	if !strings.Contains(errOutput, "log cleanup function not configured") {
		t.Fatalf("stderr = %q, want missing-fn message", errOutput)
	}
}

func TestRun_CleanupFlag(t *testing.T) {
	defer resetTestHooks()
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"codex-wrapper", "--cleanup"}

	calls := 0
	cleanupLogsFn = func() (CleanupStats, error) {
		calls++
		return CleanupStats{Scanned: 1, Deleted: 1}, nil
	}

	var exitCode int
	output := captureOutput(t, func() {
		exitCode = run()
	})
	if exitCode != 0 {
		t.Fatalf("exit = %d, want 0", exitCode)
	}
	if calls != 1 {
		t.Fatalf("cleanup called %d times, want 1", calls)
	}
	want := "Cleanup completed\nFiles scanned: 1\nFiles deleted: 1\nFiles kept: 0\n"
	if output != want {
		t.Fatalf("output = %q, want %q", output, want)
	}
	if logger := activeLogger(); logger != nil {
		t.Fatalf("logger should not initialize for --cleanup mode")
	}
}

func TestRun_NoArgs(t *testing.T) {
	defer resetTestHooks()
	os.Args = []string{"codex-wrapper"}
	if code := run(); code != 1 {
		t.Errorf("exit = %d, want 1", code)
	}
}

func TestRun_ExplicitStdinEmpty(t *testing.T) {
	defer resetTestHooks()
	os.Args = []string{"codex-wrapper", "-"}
	stdinReader = strings.NewReader("")
	isTerminalFn = func() bool { return false }
	if code := run(); code != 1 {
		t.Errorf("exit = %d, want 1", code)
	}
}

func TestRun_ExplicitStdinReadError(t *testing.T) {
	defer resetTestHooks()
	tempDir := t.TempDir()
	t.Setenv("TMPDIR", tempDir)
	logPath := filepath.Join(tempDir, fmt.Sprintf("codex-wrapper-%d.log", os.Getpid()))

	var logOutput string
	cleanupHook = func() {
		data, err := os.ReadFile(logPath)
		if err == nil {
			logOutput = string(data)
		}
	}

	os.Args = []string{"codex-wrapper", "-"}
	stdinReader = errReader{errors.New("broken stdin")}
	isTerminalFn = func() bool { return false }

	exitCode := run()

	if exitCode != 1 {
		t.Fatalf("exit code %d, want 1", exitCode)
	}
	if !strings.Contains(logOutput, "Failed to read stdin: broken stdin") {
		t.Fatalf("log missing read error entry, got %q", logOutput)
	}
	// Log file is always removed after completion (new behavior)
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatalf("log file should be removed after completion")
	}
}

func TestRun_CommandFails(t *testing.T) {
	defer resetTestHooks()
	os.Args = []string{"codex-wrapper", "task"}
	stdinReader = strings.NewReader("")
	isTerminalFn = func() bool { return true }
	codexCommand = "false"
	if code := run(); code == 0 {
		t.Errorf("expected non-zero")
	}
}

func TestRun_SuccessfulExecution(t *testing.T) {
	defer resetTestHooks()
	stdout := captureStdoutPipe()

	codexCommand = createFakeCodexScript(t, "tid-123", "ok")
	stdinReader = strings.NewReader("")
	isTerminalFn = func() bool { return true }
	os.Args = []string{"codex-wrapper", "task"}

	exitCode := run()
	if exitCode != 0 {
		t.Fatalf("exit=%d, want 0", exitCode)
	}

	restoreStdoutPipe(stdout)
	output := stdout.String()
	if !strings.Contains(output, "ok") || !strings.Contains(output, "SESSION_ID: tid-123") {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestRun_ExplicitStdinSuccess(t *testing.T) {
	defer resetTestHooks()
	stdout := captureStdoutPipe()

	codexCommand = createFakeCodexScript(t, "tid-stdin", "from-stdin")
	stdinReader = strings.NewReader("line1\nline2")
	isTerminalFn = func() bool { return false }
	os.Args = []string{"codex-wrapper", "-"}

	exitCode := run()
	restoreStdoutPipe(stdout)
	if exitCode != 0 {
		t.Fatalf("exit=%d, want 0", exitCode)
	}
	output := stdout.String()
	if !strings.Contains(output, "from-stdin") || !strings.Contains(output, "SESSION_ID: tid-stdin") {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestRun_PipedTaskReadError(t *testing.T) {
	defer resetTestHooks()
	tempDir := t.TempDir()
	t.Setenv("TMPDIR", tempDir)
	logPath := filepath.Join(tempDir, fmt.Sprintf("codex-wrapper-%d.log", os.Getpid()))

	var logOutput string
	cleanupHook = func() {
		data, err := os.ReadFile(logPath)
		if err == nil {
			logOutput = string(data)
		}
	}

	codexCommand = createFakeCodexScript(t, "tid-pipe", "piped-task")
	isTerminalFn = func() bool { return false }
	stdinReader = errReader{errors.New("pipe failure")}
	os.Args = []string{"codex-wrapper", "cli-task"}

	exitCode := run()
	if exitCode != 1 {
		t.Fatalf("exit=%d, want 1", exitCode)
	}
	if !strings.Contains(logOutput, "ERROR: Failed to read piped stdin: read stdin: pipe failure") {
		t.Fatalf("log missing piped read error, got %q", logOutput)
	}
	// Log file is always removed after completion (new behavior)
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatalf("log file should be removed after completion")
	}
}

func TestRun_PipedTaskSuccess(t *testing.T) {
	defer resetTestHooks()
	stdout := captureStdoutPipe()

	codexCommand = createFakeCodexScript(t, "tid-pipe", "piped-task")
	isTerminalFn = func() bool { return false }
	stdinReader = strings.NewReader("piped task text")
	os.Args = []string{"codex-wrapper", "cli-task"}

	exitCode := run()
	restoreStdoutPipe(stdout)
	if exitCode != 0 {
		t.Fatalf("exit=%d, want 0", exitCode)
	}
	output := stdout.String()
	if !strings.Contains(output, "piped-task") || !strings.Contains(output, "SESSION_ID: tid-pipe") {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestRun_LoggerLifecycle(t *testing.T) {
	defer resetTestHooks()
	tempDir := t.TempDir()
	t.Setenv("TMPDIR", tempDir)
	logPath := filepath.Join(tempDir, fmt.Sprintf("codex-wrapper-%d.log", os.Getpid()))

	stdout := captureStdoutPipe()

	codexCommand = createFakeCodexScript(t, "tid-logger", "ok")
	isTerminalFn = func() bool { return true }
	stdinReader = strings.NewReader("")
	os.Args = []string{"codex-wrapper", "task"}

	var fileExisted bool
	cleanupHook = func() {
		if _, err := os.Stat(logPath); err == nil {
			fileExisted = true
		}
	}

	exitCode := run()
	restoreStdoutPipe(stdout)

	if exitCode != 0 {
		t.Fatalf("exit=%d, want 0", exitCode)
	}
	if !fileExisted {
		t.Fatalf("log file was not present during run")
	}
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatalf("log file should be removed on success, but it exists")
	}
}

func TestRun_LoggerRemovedOnSignal(t *testing.T) {
	// Skip in CI due to unreliable signal delivery in containerized environments
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
		t.Skip("Skipping signal test in CI environment")
	}

	defer resetTestHooks()
	defer signal.Reset(syscall.SIGINT, syscall.SIGTERM)

	// Set shorter delays for faster test
	forceKillDelay.Store(1)

	tempDir := t.TempDir()
	t.Setenv("TMPDIR", tempDir)
	logPath := filepath.Join(tempDir, fmt.Sprintf("codex-wrapper-%d.log", os.Getpid()))

	scriptPath := filepath.Join(tempDir, "sleepy-codex.sh")
	script := `#!/bin/sh
printf '%s\n' '{"type":"thread.started","thread_id":"sig-thread"}'
sleep 2
printf '%s\n' '{"type":"item.completed","item":{"type":"agent_message","text":"late"}}'`
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	codexCommand = scriptPath
	isTerminalFn = func() bool { return true }
	stdinReader = strings.NewReader("")
	os.Args = []string{"codex-wrapper", "task"}

	exitCh := make(chan int, 1)
	go func() { exitCh <- run() }()

	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(logPath); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	_ = syscall.Kill(os.Getpid(), syscall.SIGINT)

	var exitCode int
	select {
	case exitCode = <-exitCh:
	case <-time.After(5 * time.Second):
		t.Fatalf("run() did not return after signal")
	}

	if exitCode != 130 {
		t.Fatalf("exit code = %d, want 130", exitCode)
	}
	// Log file is always removed after completion (new behavior)
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatalf("log file should be removed after completion")
	}
}

func TestRun_CleanupHookAlwaysCalled(t *testing.T) {
	defer resetTestHooks()
	called := false
	cleanupHook = func() { called = true }
	// Use a command that goes through normal flow, not --version which returns early
	codexCommand = "echo"
	buildCodexArgsFn = func(cfg *Config, targetArg string) []string {
		return []string{`{"type":"thread.started","thread_id":"x"}
{"type":"item.completed","item":{"type":"agent_message","text":"ok"}}`}
	}
	os.Args = []string{"codex-wrapper", "task"}
	if exitCode := run(); exitCode != 0 {
		t.Fatalf("exit = %d, want 0", exitCode)
	}
	if !called {
		t.Fatalf("cleanup hook not invoked")
	}
}

func TestRunStartupCleanupNil(t *testing.T) {
	defer resetTestHooks()
	cleanupLogsFn = nil
	runStartupCleanup()
}

func TestRun_CleanupFailureDoesNotBlock(t *testing.T) {
	defer resetTestHooks()
	stdout := captureStdoutPipe()
	defer restoreStdoutPipe(stdout)

	cleanupCalled := 0
	cleanupLogsFn = func() (CleanupStats, error) {
		cleanupCalled++
		panic("boom")
	}

	codexCommand = createFakeCodexScript(t, "tid-cleanup", "ok")
	stdinReader = strings.NewReader("")
	isTerminalFn = func() bool { return true }
	os.Args = []string{"codex-wrapper", "task"}

	if exit := run(); exit != 0 {
		t.Fatalf("exit = %d, want 0", exit)
	}
	if cleanupCalled != 1 {
		t.Fatalf("cleanup called %d times, want 1", cleanupCalled)
	}
}

// Coverage helper reused by logger_test to keep focused runs exercising core paths.
func TestRunParseJSONStream_CoverageSuite(t *testing.T) {
	suite := []struct {
		name string
		fn   func(*testing.T)
	}{
		{"TestRunParseJSONStream", TestRunParseJSONStream},
		{"TestRunNormalizeText", TestRunNormalizeText},
		{"TestRunTruncate", TestRunTruncate},
		{"TestRunMin", TestRunMin},
		{"TestRunGetEnv", TestRunGetEnv},
	}

	for _, tc := range suite {
		t.Run(tc.name, tc.fn)
	}
}

func TestRunHello(t *testing.T) {
	if got := hello(); got != "hello world" {
		t.Fatalf("hello() = %q, want %q", got, "hello world")
	}
}

func TestRunGreet(t *testing.T) {
	if got := greet("Linus"); got != "hello Linus" {
		t.Fatalf("greet() = %q, want %q", got, "hello Linus")
	}
}

func TestRunFarewell(t *testing.T) {
	if got := farewell("Linus"); got != "goodbye Linus" {
		t.Fatalf("farewell() = %q, want %q", got, "goodbye Linus")
	}
}

func TestRunFarewellEmpty(t *testing.T) {
	if got := farewell(""); got != "goodbye " {
		t.Fatalf("farewell(\"\") = %q, want %q", got, "goodbye ")
	}
}

func TestRunTailBuffer(t *testing.T) {
	tb := &tailBuffer{limit: 5}
	if n, err := tb.Write([]byte("abcd")); err != nil || n != 4 {
		t.Fatalf("Write returned (%d, %v), want (4, nil)", n, err)
	}
	if n, err := tb.Write([]byte("efg")); err != nil || n != 3 {
		t.Fatalf("Write returned (%d, %v), want (3, nil)", n, err)
	}
	if got := tb.String(); got != "cdefg" {
		t.Fatalf("tail buffer = %q, want %q", got, "cdefg")
	}
	if n, err := tb.Write([]byte("0123456")); err != nil || n != 7 {
		t.Fatalf("Write returned (%d, %v), want (7, nil)", n, err)
	}
	if got := tb.String(); got != "23456" {
		t.Fatalf("tail buffer = %q, want %q", got, "23456")
	}
}

func TestRunLogWriter(t *testing.T) {
	defer resetTestHooks()
	logger, err := NewLoggerWithSuffix("logwriter")
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	setLogger(logger)

	lw := newLogWriter("TEST: ", 10)
	if _, err := lw.Write([]byte("hello\n")); err != nil {
		t.Fatalf("write hello failed: %v", err)
	}
	if _, err := lw.Write([]byte("world-is-long")); err != nil {
		t.Fatalf("write world failed: %v", err)
	}
	lw.Flush()

	logger.Flush()
	logger.Close()

	data, err := os.ReadFile(logger.Path())
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "TEST: hello") {
		t.Fatalf("log missing hello entry: %s", text)
	}
	if !strings.Contains(text, "TEST: world-i...") {
		t.Fatalf("log missing truncated entry: %s", text)
	}
	os.Remove(logger.Path())
}

func TestRunDiscardInvalidJSON(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("bad line\n{\"type\":\"ok\"}\n"))
	next, err := discardInvalidJSON(nil, reader)
	if err != nil {
		t.Fatalf("discardInvalidJSON error: %v", err)
	}
	line, err := next.ReadString('\n')
	if err != nil {
		t.Fatalf("failed to read next line: %v", err)
	}
	if strings.TrimSpace(line) != `{"type":"ok"}` {
		t.Fatalf("unexpected remaining line: %q", line)
	}

	t.Run("no newline", func(t *testing.T) {
		reader := bufio.NewReader(strings.NewReader("partial"))
		decoder := json.NewDecoder(strings.NewReader(""))
		if _, err := discardInvalidJSON(decoder, reader); !errors.Is(err, io.EOF) {
			t.Fatalf("expected EOF when no newline, got %v", err)
		}
	})
}

func TestRunForwardSignals(t *testing.T) {
	defer resetTestHooks()

	if runtime.GOOS == "windows" {
		t.Skip("sleep command not available on Windows")
	}

	cmd := exec.Command("sleep", "5")
	if err := cmd.Start(); err != nil {
		t.Skipf("unable to start sleep command: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		cmd.Wait()
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	forceKillDelay.Store(0)
	defer forceKillDelay.Store(5)

	ready := make(chan struct{})
	var captured chan<- os.Signal
	signalNotifyFn = func(ch chan<- os.Signal, sig ...os.Signal) {
		captured = ch
		close(ready)
	}
	signalStopFn = func(ch chan<- os.Signal) {}
	defer func() {
		signalNotifyFn = signal.Notify
		signalStopFn = signal.Stop
	}()

	var mu sync.Mutex
	var logs []string
	forwardSignals(ctx, cmd, func(msg string) {
		mu.Lock()
		defer mu.Unlock()
		logs = append(logs, msg)
	})

	select {
	case <-ready:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("signalNotifyFn not invoked")
	}

	captured <- syscall.SIGINT

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("process did not exit after forwarded signal")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(logs) == 0 {
		t.Fatalf("expected log entry for forwarded signal")
	}
}

func TestRun_CLI_Success(t *testing.T) {
	defer resetTestHooks()
	os.Args = []string{"codex-wrapper", "do-things"}
	stdinReader = strings.NewReader("")
	isTerminalFn = func() bool { return true }

	codexCommand = "echo"
	buildCodexArgsFn = func(cfg *Config, targetArg string) []string {
		return []string{`{"type":"thread.started","thread_id":"cli-session"}` + "\n" + `{"type":"item.completed","item":{"type":"agent_message","text":"ok"}}`}
	}

	var exitCode int
	output := captureOutput(t, func() { exitCode = run() })

	if exitCode != 0 {
		t.Fatalf("run() exit=%d, want 0", exitCode)
	}
	if !strings.Contains(output, "ok") || !strings.Contains(output, "SESSION_ID: cli-session") {
		t.Fatalf("unexpected output: %q", output)
	}
}
