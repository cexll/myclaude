package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
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

func captureStdout() *capturedStdout {
	r, w, _ := os.Pipe()
	state := &capturedStdout{old: os.Stdout, reader: r, writer: w}
	os.Stdout = w
	return state
}

func restoreStdout(c *capturedStdout) {
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
			want: &Config{
				Mode:          "new",
				Task:          "analyze code",
				WorkDir:       ".",
				ExplicitStdin: false,
			},
		},
		{
			name: "task with workdir",
			args: []string{"codex-wrapper", "analyze code", "/path/to/dir"},
			want: &Config{
				Mode:          "new",
				Task:          "analyze code",
				WorkDir:       "/path/to/dir",
				ExplicitStdin: false,
			},
		},
		{
			name: "explicit stdin mode",
			args: []string{"codex-wrapper", "-"},
			want: &Config{
				Mode:          "new",
				Task:          "-",
				WorkDir:       ".",
				ExplicitStdin: true,
			},
		},
		{
			name: "stdin with workdir",
			args: []string{"codex-wrapper", "-", "/some/dir"},
			want: &Config{
				Mode:          "new",
				Task:          "-",
				WorkDir:       "/some/dir",
				ExplicitStdin: true,
			},
		},
		{
			name:    "no args",
			args:    []string{"codex-wrapper"},
			wantErr: true,
		},
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
				t.Errorf("parseArgs() unexpected error: %v", err)
				return
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
			want: &Config{
				Mode:          "resume",
				SessionID:     "session-123",
				Task:          "continue task",
				WorkDir:       ".",
				ExplicitStdin: false,
			},
		},
		{
			name: "resume with workdir",
			args: []string{"codex-wrapper", "resume", "session-456", "task", "/work"},
			want: &Config{
				Mode:          "resume",
				SessionID:     "session-456",
				Task:          "task",
				WorkDir:       "/work",
				ExplicitStdin: false,
			},
		},
		{
			name: "resume with stdin",
			args: []string{"codex-wrapper", "resume", "session-789", "-"},
			want: &Config{
				Mode:          "resume",
				SessionID:     "session-789",
				Task:          "-",
				WorkDir:       ".",
				ExplicitStdin: true,
			},
		},
		{
			name:    "resume missing session_id",
			args:    []string{"codex-wrapper", "resume"},
			wantErr: true,
		},
		{
			name:    "resume missing task",
			args:    []string{"codex-wrapper", "resume", "session-123"},
			wantErr: true,
		},
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
				t.Errorf("parseArgs() unexpected error: %v", err)
				return
			}

			if cfg.Mode != tt.want.Mode {
				t.Errorf("Mode = %v, want %v", cfg.Mode, tt.want.Mode)
			}
			if cfg.SessionID != tt.want.SessionID {
				t.Errorf("SessionID = %v, want %v", cfg.SessionID, tt.want.SessionID)
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
	cfg := &Config{
		Mode:    "new",
		WorkDir: "/test/dir",
	}

	args := buildCodexArgs(cfg, "my task")

	expected := []string{
		"e",
		"--skip-git-repo-check",
		"-C", "/test/dir",
		"--json",
		"my task",
	}

	if len(args) != len(expected) {
		t.Errorf("buildCodexArgs() returned %d args, want %d", len(args), len(expected))
		return
	}

	for i, arg := range args {
		if arg != expected[i] {
			t.Errorf("buildCodexArgs()[%d] = %v, want %v", i, arg, expected[i])
		}
	}
}

func TestRunBuildCodexArgs_ResumeMode(t *testing.T) {
	cfg := &Config{
		Mode:      "resume",
		SessionID: "session-abc",
	}

	args := buildCodexArgs(cfg, "-")

	expected := []string{
		"e",
		"--skip-git-repo-check",
		"--json",
		"resume",
		"session-abc",
		"-",
	}

	if len(args) != len(expected) {
		t.Errorf("buildCodexArgs() returned %d args, want %d", len(args), len(expected))
		return
	}

	for i, arg := range args {
		if arg != expected[i] {
			t.Errorf("buildCodexArgs()[%d] = %v, want %v", i, arg, expected[i])
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

func TestParseJSONStream(t *testing.T) {
	type testCase struct {
		name         string
		input        string
		wantMessage  string
		wantThreadID string
	}

	longText := strings.Repeat("a", 2*1024*1024) // >1MB agent_message payload

	tests := []testCase{
		{
			name: "thread started and agent message",
			input: `{"type":"thread.started","thread_id":"abc-123"}
{"type":"item.completed","item":{"type":"agent_message","text":"Hello world"}}`,
			wantMessage:  "Hello world",
			wantThreadID: "abc-123",
		},
		{
			name: "multiple agent messages (last wins)",
			input: `{"type":"item.completed","item":{"type":"agent_message","text":"First"}}
{"type":"item.completed","item":{"type":"agent_message","text":"Second"}}`,
			wantMessage:  "Second",
			wantThreadID: "",
		},
		{
			name:         "text as array",
			input:        `{"type":"item.completed","item":{"type":"agent_message","text":["Hello"," ","World"]}}`,
			wantMessage:  "Hello World",
			wantThreadID: "",
		},
		{
			name: "ignore other event types",
			input: `{"type":"other.event","data":"ignored"}
{"type":"item.completed","item":{"type":"other_type","text":"ignored"}}
{"type":"item.completed","item":{"type":"agent_message","text":"Valid"}}`,
			wantMessage:  "Valid",
			wantThreadID: "",
		},
		{
			name:         "super long single line (>1MB)",
			input:        `{"type":"item.completed","item":{"type":"agent_message","text":"` + longText + `"}}`,
			wantMessage:  longText,
			wantThreadID: "",
		},
		{
			name:         "empty input",
			input:        "",
			wantMessage:  "",
			wantThreadID: "",
		},
		{
			name: "item completed with nil item",
			input: strings.Join([]string{
				`{"type":"thread.started","thread_id":"nil-item-thread"}`,
				`{"type":"item.completed","item":null}`,
			}, "\n"),
			wantMessage:  "",
			wantThreadID: "nil-item-thread",
		},
		{
			name:         "agent message with non-string text",
			input:        `{"type":"item.completed","item":{"type":"agent_message","text":12345}}`,
			wantMessage:  "",
			wantThreadID: "",
		},
		{
			name: "corrupted json does not break stream",
			input: strings.Join([]string{
				`{"type":"item.completed","item":{"type":"agent_message","text":"before"}}`,
				`{"type":"item.completed","item":{"type":"agent_message","text":"broken"}`,
				`{"type":"thread.started","thread_id":"after-thread"}`,
				`{"type":"item.completed","item":{"type":"agent_message","text":"after"}}`,
			}, "\n"),
			wantMessage:  "after",
			wantThreadID: "after-thread",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMessage, gotThreadID := parseJSONStream(strings.NewReader(tt.input))

			if gotMessage != tt.wantMessage {
				t.Errorf("parseJSONStream() message = %q, want %q", gotMessage, tt.wantMessage)
			}
			if gotThreadID != tt.wantThreadID {
				t.Errorf("parseJSONStream() threadID = %q, want %q", gotThreadID, tt.wantThreadID)
			}
		})
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
}

func TestRunMin(t *testing.T) {
	tests := []struct {
		a, b, want int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{5, 5, 5},
		{-1, 0, -1},
		{0, -1, -1},
	}

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
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printHelp()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	expectedPhrases := []string{
		"codex-wrapper",
		"Usage:",
		"resume",
		"CODEX_TIMEOUT",
		"Exit Codes:",
	}

	for _, phrase := range expectedPhrases {
		if !strings.Contains(output, phrase) {
			t.Errorf("printHelp() missing phrase %q", phrase)
		}
	}
}

// Tests for isTerminal with mock
func TestRunIsTerminal(t *testing.T) {
	defer resetTestHooks()

	tests := []struct {
		name   string
		mockFn func() bool
		want   bool
	}{
		{"is terminal", func() bool { return true }, true},
		{"is not terminal", func() bool { return false }, false},
	}

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

// Tests for readPipedTask with mock
func TestReadPipedTask(t *testing.T) {
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
					t.Fatalf("readPipedTask() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("readPipedTask() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("readPipedTask() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseJSONStream_CoverageSuite(t *testing.T) {
	suite := []struct {
		name string
		fn   func(*testing.T)
	}{
		{"TestRunParseArgs_NewMode", TestRunParseArgs_NewMode},
		{"TestRunParseArgs_ResumeMode", TestRunParseArgs_ResumeMode},
		{"TestRunShouldUseStdin", TestRunShouldUseStdin},
		{"TestRunBuildCodexArgs_NewMode", TestRunBuildCodexArgs_NewMode},
		{"TestRunBuildCodexArgs_ResumeMode", TestRunBuildCodexArgs_ResumeMode},
		{"TestRunResolveTimeout", TestRunResolveTimeout},
		{"TestRunNormalizeText", TestRunNormalizeText},
		{"TestParseJSONStream", TestParseJSONStream},
		{"TestRunGetEnv", TestRunGetEnv},
		{"TestRunTruncate", TestRunTruncate},
		{"TestRunMin", TestRunMin},
		{"TestRunLogFunctions", TestRunLogFunctions},
		{"TestRunPrintHelp", TestRunPrintHelp},
		{"TestRunIsTerminal", TestRunIsTerminal},
		{"TestRunCodexProcess_CommandNotFound", TestRunCodexProcess_CommandNotFound},
		{"TestRunCodexProcess_WithEcho", TestRunCodexProcess_WithEcho},
		{"TestRunCodexProcess_NoMessage", TestRunCodexProcess_NoMessage},
		{"TestRunCodexProcess_WithStdin", TestRunCodexProcess_WithStdin},
		{"TestRunCodexProcess_ExitError", TestRunCodexProcess_ExitError},
		{"TestRunCodexProcess_ContextTimeout", TestRunCodexProcess_ContextTimeout},
		{"TestRunCodexProcess_SignalCancellation", TestRunCodexProcess_SignalCancellation},
		{"TestRunCancelReason", TestRunCancelReason},
		{"TestRunDefaultIsTerminal", TestRunDefaultIsTerminal},
		{"TestRunTerminateProcess_NoProcess", TestRunTerminateProcess_NoProcess},
		{"TestRun_Version", TestRun_Version},
		{"TestRun_VersionShort", TestRun_VersionShort},
		{"TestRun_Help", TestRun_Help},
		{"TestRun_HelpShort", TestRun_HelpShort},
		{"TestRun_NoArgs", TestRun_NoArgs},
		{"TestRun_ExplicitStdinEmpty", TestRun_ExplicitStdinEmpty},
		{"TestRun_ExplicitStdinReadError", TestRun_ExplicitStdinReadError},
		{"TestRun_CommandFails", TestRun_CommandFails},
		{"TestRun_SuccessfulExecution", TestRun_SuccessfulExecution},
		{"TestRun_ExplicitStdinSuccess", TestRun_ExplicitStdinSuccess},
		{"TestRun_PipedTaskReadError", TestRun_PipedTaskReadError},
		{"TestRun_PipedTaskSuccess", TestRun_PipedTaskSuccess},
		{"TestRun_CleanupHookAlwaysCalled", TestRun_CleanupHookAlwaysCalled},
	}

	for _, tt := range suite {
		t.Run(tt.name, tt.fn)
	}
}

// Tests for runCodexProcess with mock command
func TestRunCodexProcess_CommandNotFound(t *testing.T) {
	defer resetTestHooks()

	codexCommand = "nonexistent-command-xyz"

	_, _, exitCode := runCodexProcess(context.Background(), []string{"arg1"}, "task", false, 10)

	if exitCode != 127 {
		t.Errorf("runCodexProcess() exitCode = %d, want 127 for command not found", exitCode)
	}
}

func TestRunCodexProcess_WithEcho(t *testing.T) {
	defer resetTestHooks()

	// Use echo to simulate codex output
	codexCommand = "echo"

	jsonOutput := `{"type":"thread.started","thread_id":"test-session"}
{"type":"item.completed","item":{"type":"agent_message","text":"Test output"}}`

	message, threadID, exitCode := runCodexProcess(context.Background(), []string{jsonOutput}, "", false, 10)

	if exitCode != 0 {
		t.Errorf("runCodexProcess() exitCode = %d, want 0", exitCode)
	}
	if message != "Test output" {
		t.Errorf("runCodexProcess() message = %q, want %q", message, "Test output")
	}
	if threadID != "test-session" {
		t.Errorf("runCodexProcess() threadID = %q, want %q", threadID, "test-session")
	}
}

func TestRunCodexProcess_NoMessage(t *testing.T) {
	defer resetTestHooks()

	codexCommand = "echo"

	// Output without agent_message
	jsonOutput := `{"type":"thread.started","thread_id":"test-session"}`

	_, _, exitCode := runCodexProcess(context.Background(), []string{jsonOutput}, "", false, 10)

	if exitCode != 1 {
		t.Errorf("runCodexProcess() exitCode = %d, want 1 for no message", exitCode)
	}
}

func TestRunCodexProcess_WithStdin(t *testing.T) {
	defer resetTestHooks()

	// Use cat to echo stdin back
	codexCommand = "cat"

	message, _, exitCode := runCodexProcess(context.Background(), []string{}, `{"type":"item.completed","item":{"type":"agent_message","text":"from stdin"}}`, true, 10)

	if exitCode != 0 {
		t.Errorf("runCodexProcess() exitCode = %d, want 0", exitCode)
	}
	if message != "from stdin" {
		t.Errorf("runCodexProcess() message = %q, want %q", message, "from stdin")
	}
}

func TestRunCodexProcess_ExitError(t *testing.T) {
	defer resetTestHooks()

	// Use false command which exits with code 1
	codexCommand = "false"

	_, _, exitCode := runCodexProcess(context.Background(), []string{}, "", false, 10)

	if exitCode == 0 {
		t.Errorf("runCodexProcess() exitCode = 0, want non-zero for failed command")
	}
}

func TestRunCodexProcess_ContextTimeout(t *testing.T) {
	defer resetTestHooks()

	codexCommand = "sleep"

	_, _, exitCode := runCodexProcess(context.Background(), []string{"2"}, "", false, 1)

	if exitCode != 124 {
		t.Fatalf("runCodexProcess() exitCode = %d, want 124 on timeout", exitCode)
	}
}

func TestRunCodexProcess_SignalCancellation(t *testing.T) {
	defer resetTestHooks()
	defer signal.Reset(syscall.SIGINT, syscall.SIGTERM)

	codexCommand = "sleep"
	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = syscall.Kill(os.Getpid(), syscall.SIGINT)
	}()

	_, _, exitCode := runCodexProcess(sigCtx, []string{"5"}, "", false, 10)

	if exitCode != 130 {
		t.Fatalf("runCodexProcess() exitCode = %d, want 130 on signal", exitCode)
	}
}

func TestRunCancelReason(t *testing.T) {
	if got := cancelReason(nil); got != "Context cancelled" {
		t.Fatalf("cancelReason(nil) = %q, want Context cancelled", got)
	}
}

func TestRunDefaultIsTerminal(t *testing.T) {
	// This test just ensures defaultIsTerminal doesn't panic
	// The actual result depends on the test environment
	_ = defaultIsTerminal()
}

func TestRunTerminateProcess_NoProcess(t *testing.T) {
	timer := terminateProcess(nil)

	if timer != nil {
		t.Fatalf("terminateProcess(nil) expected nil timer, got non-nil")
	}
}

// Tests for run() function
func TestRun_Version(t *testing.T) {
	defer resetTestHooks()

	os.Args = []string{"codex-wrapper", "--version"}
	exitCode := run()
	if exitCode != 0 {
		t.Errorf("run() with --version returned %d, want 0", exitCode)
	}
}

func TestRun_VersionShort(t *testing.T) {
	defer resetTestHooks()

	os.Args = []string{"codex-wrapper", "-v"}
	exitCode := run()
	if exitCode != 0 {
		t.Errorf("run() with -v returned %d, want 0", exitCode)
	}
}

func TestRun_Help(t *testing.T) {
	defer resetTestHooks()

	os.Args = []string{"codex-wrapper", "--help"}
	exitCode := run()
	if exitCode != 0 {
		t.Errorf("run() with --help returned %d, want 0", exitCode)
	}
}

func TestRun_HelpShort(t *testing.T) {
	defer resetTestHooks()

	os.Args = []string{"codex-wrapper", "-h"}
	exitCode := run()
	if exitCode != 0 {
		t.Errorf("run() with -h returned %d, want 0", exitCode)
	}
}

func TestRun_NoArgs(t *testing.T) {
	defer resetTestHooks()

	os.Args = []string{"codex-wrapper"}
	exitCode := run()
	if exitCode != 1 {
		t.Errorf("run() with no args returned %d, want 1", exitCode)
	}
}

func TestRun_ExplicitStdinEmpty(t *testing.T) {
	defer resetTestHooks()

	os.Args = []string{"codex-wrapper", "-"}
	stdinReader = strings.NewReader("")
	isTerminalFn = func() bool { return false }

	exitCode := run()
	if exitCode != 1 {
		t.Errorf("run() with empty stdin returned %d, want 1", exitCode)
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
		t.Fatalf("run() with stdin read error returned %d, want 1", exitCode)
	}
	if !strings.Contains(logOutput, "Failed to read stdin: broken stdin") {
		t.Fatalf("log missing read error entry, got %q", logOutput)
	}
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatalf("log file still exists after run, err=%v", err)
	}
}

func TestRun_CommandFails(t *testing.T) {
	defer resetTestHooks()

	os.Args = []string{"codex-wrapper", "task"}
	stdinReader = strings.NewReader("")
	isTerminalFn = func() bool { return true }
	codexCommand = "false"

	exitCode := run()
	if exitCode == 0 {
		t.Errorf("run() with failing command returned 0, want non-zero")
	}
}

func TestRun_SuccessfulExecution(t *testing.T) {
	defer resetTestHooks()

	stdout := captureStdout()

	codexCommand = createFakeCodexScript(t, "tid-123", "ok")
	stdinReader = strings.NewReader("")
	isTerminalFn = func() bool { return true }
	os.Args = []string{"codex-wrapper", "task"}

	exitCode := run()
	if exitCode != 0 {
		t.Fatalf("run() returned %d, want 0", exitCode)
	}

	restoreStdout(stdout)
	output := stdout.String()
	if !strings.Contains(output, "ok") {
		t.Fatalf("stdout missing agent message, got %q", output)
	}
	if !strings.Contains(output, "SESSION_ID: tid-123") {
		t.Fatalf("stdout missing session id, got %q", output)
	}
}

func TestRun_ExplicitStdinSuccess(t *testing.T) {
	defer resetTestHooks()

	stdout := captureStdout()

	codexCommand = createFakeCodexScript(t, "tid-stdin", "from-stdin")
	stdinReader = strings.NewReader("line1\nline2")
	isTerminalFn = func() bool { return false }
	os.Args = []string{"codex-wrapper", "-"}

	exitCode := run()
	restoreStdout(stdout)
	if exitCode != 0 {
		t.Fatalf("run() returned %d, want 0", exitCode)
	}

	output := stdout.String()
	if !strings.Contains(output, "from-stdin") {
		t.Fatalf("stdout missing agent message for stdin, got %q", output)
	}
	if !strings.Contains(output, "SESSION_ID: tid-stdin") {
		t.Fatalf("stdout missing session id for stdin, got %q", output)
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
		t.Fatalf("run() with piped read error returned %d, want 1", exitCode)
	}
	if !strings.Contains(logOutput, "Failed to read piped stdin: read stdin: pipe failure") {
		t.Fatalf("log missing piped read error entry, got %q", logOutput)
	}
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatalf("log file still exists after run, err=%v", err)
	}
}

func TestRun_PipedTaskSuccess(t *testing.T) {
	defer resetTestHooks()

	stdout := captureStdout()

	codexCommand = createFakeCodexScript(t, "tid-pipe", "piped-task")
	isTerminalFn = func() bool { return false }
	stdinReader = strings.NewReader("piped task text")
	os.Args = []string{"codex-wrapper", "cli-task"}

	exitCode := run()
	restoreStdout(stdout)
	if exitCode != 0 {
		t.Fatalf("run() returned %d, want 0", exitCode)
	}

	output := stdout.String()
	if !strings.Contains(output, "piped-task") {
		t.Fatalf("stdout missing agent message for piped task, got %q", output)
	}
	if !strings.Contains(output, "SESSION_ID: tid-pipe") {
		t.Fatalf("stdout missing session id for piped task, got %q", output)
	}
}

func TestRun_LoggerLifecycle(t *testing.T) {
	defer resetTestHooks()

	tempDir := t.TempDir()
	t.Setenv("TMPDIR", tempDir)
	logPath := filepath.Join(tempDir, fmt.Sprintf("codex-wrapper-%d.log", os.Getpid()))

	stdout := captureStdout()

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
	restoreStdout(stdout)

	if exitCode != 0 {
		t.Fatalf("run() returned %d, want 0", exitCode)
	}
	if !fileExisted {
		t.Fatalf("log file was not present during run")
	}
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatalf("log file still exists after run, err=%v", err)
	}
}

func TestRun_LoggerRemovedOnSignal(t *testing.T) {
	defer resetTestHooks()
	defer signal.Reset(syscall.SIGINT, syscall.SIGTERM)

	tempDir := t.TempDir()
	t.Setenv("TMPDIR", tempDir)
	logPath := filepath.Join(tempDir, fmt.Sprintf("codex-wrapper-%d.log", os.Getpid()))

	scriptPath := filepath.Join(tempDir, "sleepy-codex.sh")
	script := `#!/bin/sh
printf '%s\n' '{"type":"thread.started","thread_id":"sig-thread"}'
sleep 5
printf '%s\n' '{"type":"item.completed","item":{"type":"agent_message","text":"late"}}'`
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	codexCommand = scriptPath
	isTerminalFn = func() bool { return true }
	stdinReader = strings.NewReader("")
	os.Args = []string{"codex-wrapper", "task"}

	exitCh := make(chan int, 1)
	go func() {
		exitCh <- run()
	}()

	deadline := time.Now().Add(2 * time.Second)
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
	case <-time.After(3 * time.Second):
		t.Fatalf("run() did not return after signal")
	}

	if exitCode != 130 {
		t.Fatalf("run() exit code = %d, want 130 on signal", exitCode)
	}
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatalf("log file still exists after signal exit, err=%v", err)
	}
}

func TestRun_CleanupHookAlwaysCalled(t *testing.T) {
	defer resetTestHooks()

	called := false
	cleanupHook = func() { called = true }

	os.Args = []string{"codex-wrapper", "--version"}

	exitCode := run()
	if exitCode != 0 {
		t.Fatalf("run() with --version returned %d, want 0", exitCode)
	}

	if !called {
		t.Fatalf("cleanup hook was not invoked")
	}
}
