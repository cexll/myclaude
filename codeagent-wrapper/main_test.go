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
	buildCodexArgsFn = buildCodexArgs
	selectBackendFn = selectBackend
	commandContext = exec.CommandContext
	jsonMarshal = json.Marshal
	forceKillDelay = 5
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

type testBackend struct {
	name    string
	command string
	argsFn  func(*Config, string) []string
}

func (t testBackend) Name() string {
	if t.name != "" {
		return t.name
	}
	return "test-backend"
}

func (t testBackend) BuildArgs(cfg *Config, targetArg string) []string {
	if t.argsFn != nil {
		return t.argsFn(cfg, targetArg)
	}
	return []string{targetArg}
}

func (t testBackend) Command() string {
	if t.command != "" {
		return t.command
	}
	return "echo"
}

func withBackend(command string, argsFn func(*Config, string) []string) func() {
	prev := selectBackendFn
	selectBackendFn = func(name string) (Backend, error) {
		return testBackend{name: name, command: command, argsFn: argsFn}, nil
	}
	return func() { selectBackendFn = prev }
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
			args: []string{"codeagent-wrapper", "analyze code"},
			want: &Config{Mode: "new", Task: "analyze code", WorkDir: ".", ExplicitStdin: false, Backend: defaultBackendName},
		},
		{
			name: "task with workdir",
			args: []string{"codeagent-wrapper", "analyze code", "/path/to/dir"},
			want: &Config{Mode: "new", Task: "analyze code", WorkDir: "/path/to/dir", ExplicitStdin: false, Backend: defaultBackendName},
		},
		{
			name: "explicit stdin mode",
			args: []string{"codeagent-wrapper", "-"},
			want: &Config{Mode: "new", Task: "-", WorkDir: ".", ExplicitStdin: true, Backend: defaultBackendName},
		},
		{
			name: "stdin with workdir",
			args: []string{"codeagent-wrapper", "-", "/some/dir"},
			want: &Config{Mode: "new", Task: "-", WorkDir: "/some/dir", ExplicitStdin: true, Backend: defaultBackendName},
		},
		{name: "no args", args: []string{"codeagent-wrapper"}, wantErr: true},
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
			if cfg.Backend != tt.want.Backend {
				t.Errorf("Backend = %v, want %v", cfg.Backend, tt.want.Backend)
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
			args: []string{"codeagent-wrapper", "resume", "session-123", "continue task"},
			want: &Config{Mode: "resume", SessionID: "session-123", Task: "continue task", WorkDir: ".", ExplicitStdin: false, Backend: defaultBackendName},
		},
		{
			name: "resume with workdir",
			args: []string{"codeagent-wrapper", "resume", "session-456", "task", "/work"},
			want: &Config{Mode: "resume", SessionID: "session-456", Task: "task", WorkDir: "/work", ExplicitStdin: false, Backend: defaultBackendName},
		},
		{
			name: "resume with stdin",
			args: []string{"codeagent-wrapper", "resume", "session-789", "-"},
			want: &Config{Mode: "resume", SessionID: "session-789", Task: "-", WorkDir: ".", ExplicitStdin: true, Backend: defaultBackendName},
		},
		{name: "resume missing session_id", args: []string{"codeagent-wrapper", "resume"}, wantErr: true},
		{name: "resume missing task", args: []string{"codeagent-wrapper", "resume", "session-123"}, wantErr: true},
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
			if cfg.Backend != tt.want.Backend {
				t.Errorf("Backend = %v, want %v", cfg.Backend, tt.want.Backend)
			}
		})
	}
}

func TestRunParseArgs_BackendFlag(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    string
		wantErr bool
	}{
		{
			name: "claude backend",
			args: []string{"codeagent-wrapper", "--backend", "claude", "task"},
			want: "claude",
		},
		{
			name: "gemini resume",
			args: []string{"codeagent-wrapper", "--backend", "gemini", "resume", "sid", "task"},
			want: "gemini",
		},
		{
			name: "backend equals syntax",
			args: []string{"codeagent-wrapper", "--backend=claude", "task"},
			want: "claude",
		},
		{
			name:    "missing backend value",
			args:    []string{"codeagent-wrapper", "--backend"},
			wantErr: true,
		},
		{
			name:    "backend equals missing value",
			args:    []string{"codeagent-wrapper", "--backend=", "task"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args
			cfg, err := parseArgs()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.Backend != tt.want {
				t.Fatalf("Backend = %q, want %q", cfg.Backend, tt.want)
			}
		})
	}
}

func TestParseParallelConfig_Success(t *testing.T) {
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

func TestParseParallelConfig_InvalidFormat(t *testing.T) {
	if _, err := parseParallelConfig([]byte("invalid format")); err == nil {
		t.Fatalf("expected error for invalid format, got nil")
	}
}

func TestParseParallelConfig_EmptyTasks(t *testing.T) {
	input := `---TASK---
id: empty
---CONTENT---
`
	if _, err := parseParallelConfig([]byte(input)); err == nil {
		t.Fatalf("expected error for empty tasks array, got nil")
	}
}

func TestParseParallelConfig_MissingID(t *testing.T) {
	input := `---TASK---
---CONTENT---
do something`
	if _, err := parseParallelConfig([]byte(input)); err == nil {
		t.Fatalf("expected error for missing id, got nil")
	}
}

func TestParseParallelConfig_MissingTask(t *testing.T) {
	input := `---TASK---
id: task-1
---CONTENT---
`
	if _, err := parseParallelConfig([]byte(input)); err == nil {
		t.Fatalf("expected error for missing task, got nil")
	}
}

func TestParseParallelConfig_DuplicateID(t *testing.T) {
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

func TestParseParallelConfig_DelimiterFormat(t *testing.T) {
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

func TestShouldUseStdin(t *testing.T) {
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

func TestSelectBackend(t *testing.T) {
	tests := []struct {
		name string
		in   string
		kind Backend
	}{
		{"codex", "codex", CodexBackend{}},
		{"claude mixed case", "ClAuDe", ClaudeBackend{}},
		{"gemini", "gemini", GeminiBackend{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := selectBackend(tt.in)
			if err != nil {
				t.Fatalf("selectBackend() error = %v", err)
			}
			switch tt.kind.(type) {
			case CodexBackend:
				if _, ok := got.(CodexBackend); !ok {
					t.Fatalf("expected CodexBackend, got %T", got)
				}
			case ClaudeBackend:
				if _, ok := got.(ClaudeBackend); !ok {
					t.Fatalf("expected ClaudeBackend, got %T", got)
				}
			case GeminiBackend:
				if _, ok := got.(GeminiBackend); !ok {
					t.Fatalf("expected GeminiBackend, got %T", got)
				}
			}
		})
	}
}

func TestSelectBackend_Invalid(t *testing.T) {
	if _, err := selectBackend("unknown"); err == nil {
		t.Fatalf("expected error for invalid backend")
	}
}

func TestSelectBackend_DefaultOnEmpty(t *testing.T) {
	backend, err := selectBackend("")
	if err != nil {
		t.Fatalf("selectBackend(\"\") error = %v", err)
	}
	if _, ok := backend.(CodexBackend); !ok {
		t.Fatalf("expected default CodexBackend, got %T", backend)
	}
}

func TestBackendBuildArgs_CodexBackend(t *testing.T) {
	backend := CodexBackend{}
	cfg := &Config{Mode: "new", WorkDir: "/test/dir"}
	got := backend.BuildArgs(cfg, "task")
	want := []string{"e", "--skip-git-repo-check", "-C", "/test/dir", "--json", "task"}
	if len(got) != len(want) {
		t.Fatalf("length mismatch")
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d got %s want %s", i, got[i], want[i])
		}
	}
}

func TestBackendBuildArgs_ClaudeBackend(t *testing.T) {
	backend := ClaudeBackend{}
	cfg := &Config{Mode: "new", WorkDir: defaultWorkdir}
	got := backend.BuildArgs(cfg, "todo")
	want := []string{"-p", "--dangerously-skip-permissions", "--output-format", "stream-json", "--verbose", "todo"}
	if len(got) != len(want) {
		t.Fatalf("length mismatch")
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d got %s want %s", i, got[i], want[i])
		}
	}

	if backend.BuildArgs(nil, "ignored") != nil {
		t.Fatalf("nil config should return nil args")
	}
}

func TestClaudeBackendBuildArgs_OutputValidation(t *testing.T) {
	backend := ClaudeBackend{}
	cfg := &Config{Mode: "resume"}
	target := "ensure-flags"

	args := backend.BuildArgs(cfg, target)
	expectedPrefix := []string{"-p", "--dangerously-skip-permissions", "--output-format", "stream-json", "--verbose"}

	if len(args) != len(expectedPrefix)+1 {
		t.Fatalf("args length=%d, want %d", len(args), len(expectedPrefix)+1)
	}
	for i, val := range expectedPrefix {
		if args[i] != val {
			t.Fatalf("args[%d]=%q, want %q", i, args[i], val)
		}
	}
	if args[len(args)-1] != target {
		t.Fatalf("last arg=%q, want target %q", args[len(args)-1], target)
	}
}

func TestBackendBuildArgs_GeminiBackend(t *testing.T) {
	backend := GeminiBackend{}
	cfg := &Config{Mode: "new"}
	got := backend.BuildArgs(cfg, "task")
	want := []string{"-o", "stream-json", "-y", "-p", "task"}
	if len(got) != len(want) {
		t.Fatalf("length mismatch")
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d got %s want %s", i, got[i], want[i])
		}
	}

	if backend.BuildArgs(nil, "ignored") != nil {
		t.Fatalf("nil config should return nil args")
	}
}

func TestGeminiBackendBuildArgs_OutputValidation(t *testing.T) {
	backend := GeminiBackend{}
	cfg := &Config{Mode: "resume"}
	target := "prompt-data"

	args := backend.BuildArgs(cfg, target)
	expected := []string{"-o", "stream-json", "-y", "-p"}

	if len(args) != len(expected)+1 {
		t.Fatalf("args length=%d, want %d", len(args), len(expected)+1)
	}
	for i, val := range expected {
		if args[i] != val {
			t.Fatalf("args[%d]=%q, want %q", i, args[i], val)
		}
	}
	if args[len(args)-1] != target {
		t.Fatalf("last arg=%q, want target %q", args[len(args)-1], target)
	}
}

func TestBackendNamesAndCommands(t *testing.T) {
	tests := []Backend{CodexBackend{}, ClaudeBackend{}, GeminiBackend{}}
	expected := []struct {
		name    string
		command string
	}{
		{"codex", "codex"},
		{"claude", "claude"},
		{"gemini", "gemini"},
	}

	for i, backend := range tests {
		if backend.Name() != expected[i].name {
			t.Fatalf("backend %d name = %s, want %s", i, backend.Name(), expected[i].name)
		}
		if backend.Command() != expected[i].command {
			t.Fatalf("backend %d command = %s, want %s", i, backend.Command(), expected[i].command)
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

func TestParseJSONStream_ClaudeEvents(t *testing.T) {
	input := `{"type":"system","subtype":"init","session_id":"abc123"}
{"type":"result","subtype":"success","result":"Hello!","session_id":"abc123"}`

	message, threadID := parseJSONStream(strings.NewReader(input))

	if message != "Hello!" {
		t.Fatalf("message=%q, want %q", message, "Hello!")
	}
	if threadID != "abc123" {
		t.Fatalf("threadID=%q, want %q", threadID, "abc123")
	}
}

func TestParseJSONStream_GeminiEvents(t *testing.T) {
	input := `{"type":"init","session_id":"xyz789"}
{"type":"message","role":"assistant","content":"Hi","delta":true,"session_id":"xyz789"}
{"type":"message","role":"assistant","content":" there","delta":true}
{"type":"result","status":"success","session_id":"xyz789"}`

	message, threadID := parseJSONStream(strings.NewReader(input))

	if message != "Hi there" {
		t.Fatalf("message=%q, want %q", message, "Hi there")
	}
	if threadID != "xyz789" {
		t.Fatalf("threadID=%q, want %q", threadID, "xyz789")
	}
}

func TestParseJSONStreamWithWarn_InvalidLine(t *testing.T) {
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

func TestDiscardInvalidJSON(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("line1\nline2\n"))
	newReader, err := discardInvalidJSON(nil, reader)
	if err != nil && !errors.Is(err, io.EOF) {
		t.Fatalf("unexpected error: %v", err)
	}
	line, _ := newReader.ReadString('\n')
	if strings.TrimSpace(line) != "line2" {
		t.Fatalf("expected to continue with remaining data, got %q", line)
	}

	readerNoNewline := bufio.NewReader(strings.NewReader("no newline"))
	if _, err := discardInvalidJSON(nil, readerNoNewline); err == nil {
		t.Fatalf("expected error when no newline present")
	}
}

func TestHasKey(t *testing.T) {
	raw := map[string]json.RawMessage{
		"present": json.RawMessage(`true`),
	}

	if !hasKey(raw, "present") {
		t.Fatalf("expected key 'present' to be found")
	}
	if hasKey(raw, "absent") {
		t.Fatalf("did not expect key 'absent' to be found")
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
		{"negative maxLen", "hello", -1, ""},
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

func TestTailBufferWrite(t *testing.T) {
	buf := &tailBuffer{limit: 5}
	if n, _ := buf.Write([]byte("123")); n != 3 || buf.String() != "123" {
		t.Fatalf("unexpected buffer content %q", buf.String())
	}

	if _, _ = buf.Write([]byte("4567")); buf.String() != "34567" {
		t.Fatalf("overflow case mismatch, got %q", buf.String())
	}

	if _, _ = buf.Write([]byte("abcdefgh")); buf.String() != "defgh" {
		t.Fatalf("len>=limit case mismatch, got %q", buf.String())
	}

	noLimit := &tailBuffer{limit: 0}
	if _, _ = noLimit.Write([]byte("ignored")); noLimit.String() != "" {
		t.Fatalf("limit<=0 should not retain data")
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

func TestLoggerPathAndRemoveNil(t *testing.T) {
	var logger *Logger
	if logger.Path() != "" {
		t.Fatalf("nil logger path should be empty")
	}
	if err := logger.RemoveLogFile(); err != nil {
		t.Fatalf("expected nil logger RemoveLogFile to be no-op, got %v", err)
	}
}

func TestLoggerLogDropOnDone(t *testing.T) {
	logger := &Logger{
		ch:   make(chan logEntry),
		done: make(chan struct{}),
	}
	close(logger.done)
	logger.log("INFO", "dropped")
	logger.pendingWG.Wait()
}

func TestLoggerLogAfterClose(t *testing.T) {
	defer resetTestHooks()
	logger, err := NewLogger()
	if err != nil {
		t.Fatalf("NewLogger error: %v", err)
	}
	if err := logger.Close(); err != nil {
		t.Fatalf("Close error: %v", err)
	}
	logger.log("INFO", "should be ignored")
}

func TestLogWriterLogLine(t *testing.T) {
	defer resetTestHooks()
	logger, err := NewLogger()
	if err != nil {
		t.Fatalf("NewLogger error: %v", err)
	}
	setLogger(logger)
	lw := &logWriter{prefix: "P:", maxLen: 3}
	lw.buf.WriteString("abcdef")
	lw.logLine(false)
	lw.logLine(false) // empty buffer path
	logger.Flush()
	data, _ := os.ReadFile(logger.Path())
	if !strings.Contains(string(data), "P:abc") {
		t.Fatalf("log output missing truncated entry, got %q", string(data))
	}
	closeLogger()
}

func TestNewLogWriterDefaultMaxLen(t *testing.T) {
	lw := newLogWriter("X:", 0)
	if lw.maxLen != codexLogLineLimit {
		t.Fatalf("expected default maxLen %d, got %d", codexLogLineLimit, lw.maxLen)
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

	expected := []string{"codeagent-wrapper", "Usage:", "resume", "CODEX_TIMEOUT", "Exit Codes:"}
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
	if res.ExitCode != 1 || !strings.Contains(res.Error, "failed to start") {
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

func TestRunCodexProcess_WithStdin(t *testing.T) {
	defer resetTestHooks()
	codexCommand = "cat"
	jsonOutput := `{"type":"thread.started","thread_id":"proc"}`
	jsonOutput += "\n"
	jsonOutput += `{"type":"item.completed","item":{"type":"agent_message","text":"ok"}}`

	msg, tid, exit := runCodexProcess(context.Background(), []string{}, jsonOutput, true, 5)
	if exit != 0 {
		t.Fatalf("exit code %d, want 0", exit)
	}
	if msg != "ok" || tid != "proc" {
		t.Fatalf("unexpected output msg=%q tid=%q", msg, tid)
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

func TestForwardSignals_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	forwardSignals(ctx, &exec.Cmd{}, func(string) {})
	cancel()
	time.Sleep(10 * time.Millisecond)
}

func TestCancelReason(t *testing.T) {
	if got := cancelReason(nil); got != "Context cancelled" {
		t.Fatalf("cancelReason(nil) = %q, want %q", got, "Context cancelled")
	}

	ctxTimeout, cancelTimeout := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancelTimeout()
	<-ctxTimeout.Done()
	wantTimeout := fmt.Sprintf("%s execution timeout", codexCommand)
	if got := cancelReason(ctxTimeout); got != wantTimeout {
		t.Fatalf("cancelReason(deadline) = %q, want %q", got, wantTimeout)
	}

	ctxCancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if got := cancelReason(ctxCancelled); got != "Execution cancelled, terminating codex process" {
		t.Fatalf("cancelReason(cancelled) = %q, want %q", got, "Execution cancelled, terminating codex process")
	}
}

func TestSilentMode(t *testing.T) {
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

func TestGenerateFinalOutput(t *testing.T) {
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

func TestTopologicalSort_LinearChain(t *testing.T) {
	tasks := []TaskSpec{{ID: "a"}, {ID: "b", Dependencies: []string{"a"}}, {ID: "c", Dependencies: []string{"b"}}}
	layers, err := topologicalSort(tasks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(layers) != 3 {
		t.Fatalf("expected 3 layers, got %d", len(layers))
	}
}

func TestTopologicalSort_Branching(t *testing.T) {
	tasks := []TaskSpec{{ID: "root"}, {ID: "left", Dependencies: []string{"root"}}, {ID: "right", Dependencies: []string{"root"}}, {ID: "leaf", Dependencies: []string{"left", "right"}}}
	layers, err := topologicalSort(tasks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(layers) != 3 || len(layers[1]) != 2 {
		t.Fatalf("unexpected layers: %+v", layers)
	}
}

func TestTopologicalSort_ParallelTasks(t *testing.T) {
	tasks := []TaskSpec{{ID: "a"}, {ID: "b"}, {ID: "c"}}
	layers, err := topologicalSort(tasks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(layers) != 1 || len(layers[0]) != 3 {
		t.Fatalf("unexpected result: %+v", layers)
	}
}

func TestShouldSkipTask(t *testing.T) {
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

func TestTopologicalSort_CycleDetection(t *testing.T) {
	tasks := []TaskSpec{{ID: "a", Dependencies: []string{"b"}}, {ID: "b", Dependencies: []string{"a"}}}
	if _, err := topologicalSort(tasks); err == nil || !strings.Contains(err.Error(), "cycle detected") {
		t.Fatalf("expected cycle error, got %v", err)
	}
}

func TestTopologicalSort_IndirectCycle(t *testing.T) {
	tasks := []TaskSpec{{ID: "a", Dependencies: []string{"c"}}, {ID: "b", Dependencies: []string{"a"}}, {ID: "c", Dependencies: []string{"b"}}}
	if _, err := topologicalSort(tasks); err == nil || !strings.Contains(err.Error(), "cycle detected") {
		t.Fatalf("expected cycle error, got %v", err)
	}
}

func TestTopologicalSort_MissingDependency(t *testing.T) {
	tasks := []TaskSpec{{ID: "a", Dependencies: []string{"missing"}}}
	if _, err := topologicalSort(tasks); err == nil || !strings.Contains(err.Error(), "dependency \"missing\" not found") {
		t.Fatalf("expected missing dependency error, got %v", err)
	}
}

func TestTopologicalSort_LargeGraph(t *testing.T) {
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

func TestExecuteConcurrent_ParallelExecution(t *testing.T) {
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

func TestExecuteConcurrent_LayerOrdering(t *testing.T) {
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

func TestExecuteConcurrent_ErrorIsolation(t *testing.T) {
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

func TestExecuteConcurrent_PanicRecovered(t *testing.T) {
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

func TestExecuteConcurrent_LargeFanout(t *testing.T) {
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

	os.Args = []string{"codeagent-wrapper", "--parallel"}
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

func TestRun_Version(t *testing.T) {
	defer resetTestHooks()
	os.Args = []string{"codeagent-wrapper", "--version"}
	if code := run(); code != 0 {
		t.Errorf("exit = %d, want 0", code)
	}
}

func TestRun_VersionShort(t *testing.T) {
	defer resetTestHooks()
	os.Args = []string{"codeagent-wrapper", "-v"}
	if code := run(); code != 0 {
		t.Errorf("exit = %d, want 0", code)
	}
}

func TestRun_Help(t *testing.T) {
	defer resetTestHooks()
	os.Args = []string{"codeagent-wrapper", "--help"}
	if code := run(); code != 0 {
		t.Errorf("exit = %d, want 0", code)
	}
}

func TestRun_HelpShort(t *testing.T) {
	defer resetTestHooks()
	os.Args = []string{"codeagent-wrapper", "-h"}
	if code := run(); code != 0 {
		t.Errorf("exit = %d, want 0", code)
	}
}

func TestRun_NoArgs(t *testing.T) {
	defer resetTestHooks()
	os.Args = []string{"codeagent-wrapper"}
	if code := run(); code != 1 {
		t.Errorf("exit = %d, want 1", code)
	}
}

func TestRun_ExplicitStdinEmpty(t *testing.T) {
	defer resetTestHooks()
	os.Args = []string{"codeagent-wrapper", "-"}
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
	logPath := filepath.Join(tempDir, fmt.Sprintf("codeagent-wrapper-%d.log", os.Getpid()))

	var logOutput string
	cleanupHook = func() {
		data, err := os.ReadFile(logPath)
		if err == nil {
			logOutput = string(data)
		}
	}

	os.Args = []string{"codeagent-wrapper", "-"}
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
	os.Args = []string{"codeagent-wrapper", "task"}
	stdinReader = strings.NewReader("")
	isTerminalFn = func() bool { return true }
	restore := withBackend("false", func(cfg *Config, targetArg string) []string { return []string{} })
	defer restore()
	if code := run(); code == 0 {
		t.Errorf("expected non-zero")
	}
}

func TestRun_InvalidBackend(t *testing.T) {
	defer resetTestHooks()
	os.Args = []string{"codeagent-wrapper", "--backend", "unknown", "task"}
	stdinReader = strings.NewReader("")
	isTerminalFn = func() bool { return true }
	if code := run(); code == 0 {
		t.Fatalf("expected non-zero exit for invalid backend")
	}
}

func TestRun_SuccessfulExecution(t *testing.T) {
	defer resetTestHooks()
	stdout := captureStdoutPipe()

	restore := withBackend(createFakeCodexScript(t, "tid-123", "ok"), buildCodexArgs)
	defer restore()
	stdinReader = strings.NewReader("")
	isTerminalFn = func() bool { return true }
	os.Args = []string{"codeagent-wrapper", "task"}

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

	restore := withBackend(createFakeCodexScript(t, "tid-stdin", "from-stdin"), buildCodexArgs)
	defer restore()
	stdinReader = strings.NewReader("line1\nline2")
	isTerminalFn = func() bool { return false }
	os.Args = []string{"codeagent-wrapper", "-"}

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
	logPath := filepath.Join(tempDir, fmt.Sprintf("codeagent-wrapper-%d.log", os.Getpid()))

	var logOutput string
	cleanupHook = func() {
		data, err := os.ReadFile(logPath)
		if err == nil {
			logOutput = string(data)
		}
	}

	restore := withBackend(createFakeCodexScript(t, "tid-pipe", "piped-task"), buildCodexArgs)
	defer restore()
	isTerminalFn = func() bool { return false }
	stdinReader = errReader{errors.New("pipe failure")}
	os.Args = []string{"codeagent-wrapper", "cli-task"}

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

	restore := withBackend(createFakeCodexScript(t, "tid-pipe", "piped-task"), buildCodexArgs)
	defer restore()
	isTerminalFn = func() bool { return false }
	stdinReader = strings.NewReader("piped task text")
	os.Args = []string{"codeagent-wrapper", "cli-task"}

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
	logPath := filepath.Join(tempDir, fmt.Sprintf("codeagent-wrapper-%d.log", os.Getpid()))

	stdout := captureStdoutPipe()

	restore := withBackend(createFakeCodexScript(t, "tid-logger", "ok"), buildCodexArgs)
	defer restore()
	isTerminalFn = func() bool { return true }
	stdinReader = strings.NewReader("")
	os.Args = []string{"codeagent-wrapper", "task"}

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
	forceKillDelay = 1

	tempDir := t.TempDir()
	t.Setenv("TMPDIR", tempDir)
	logPath := filepath.Join(tempDir, fmt.Sprintf("codeagent-wrapper-%d.log", os.Getpid()))

	scriptPath := filepath.Join(tempDir, "sleepy-codex.sh")
	script := `#!/bin/sh
printf '%s\n' '{"type":"thread.started","thread_id":"sig-thread"}'
sleep 2
printf '%s\n' '{"type":"item.completed","item":{"type":"agent_message","text":"late"}}'`
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}

	restore := withBackend(scriptPath, buildCodexArgs)
	defer restore()
	isTerminalFn = func() bool { return true }
	stdinReader = strings.NewReader("")
	os.Args = []string{"codeagent-wrapper", "task"}

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
	restore := withBackend("echo", func(cfg *Config, targetArg string) []string {
		return []string{`{"type":"thread.started","thread_id":"x"}
{"type":"item.completed","item":{"type":"agent_message","text":"ok"}}`}
	})
	defer restore()
	os.Args = []string{"codeagent-wrapper", "task"}
	if exitCode := run(); exitCode != 0 {
		t.Fatalf("exit = %d, want 0", exitCode)
	}
	if !called {
		t.Fatalf("cleanup hook not invoked")
	}
}

// Coverage helper reused by logger_test to keep focused runs exercising core paths.
func TestParseJSONStream_CoverageSuite(t *testing.T) {
	suite := []struct {
		name string
		fn   func(*testing.T)
	}{
		{"TestParseJSONStream", TestParseJSONStream},
		{"TestRunNormalizeText", TestRunNormalizeText},
		{"TestRunTruncate", TestRunTruncate},
		{"TestRunMin", TestRunMin},
		{"TestRunGetEnv", TestRunGetEnv},
	}

	for _, tc := range suite {
		t.Run(tc.name, tc.fn)
	}
}

func TestHello(t *testing.T) {
	if got := hello(); got != "hello world" {
		t.Fatalf("hello() = %q, want %q", got, "hello world")
	}
}

func TestGreet(t *testing.T) {
	if got := greet("Linus"); got != "hello Linus" {
		t.Fatalf("greet() = %q, want %q", got, "hello Linus")
	}
}

func TestFarewell(t *testing.T) {
	if got := farewell("Linus"); got != "goodbye Linus" {
		t.Fatalf("farewell() = %q, want %q", got, "goodbye Linus")
	}
}

func TestFarewellEmpty(t *testing.T) {
	if got := farewell(""); got != "goodbye " {
		t.Fatalf("farewell(\"\") = %q, want %q", got, "goodbye ")
	}
}

func TestRun_CLI_Success(t *testing.T) {
	defer resetTestHooks()
	os.Args = []string{"codeagent-wrapper", "do-things"}
	stdinReader = strings.NewReader("")
	isTerminalFn = func() bool { return true }

	restore := withBackend("echo", func(cfg *Config, targetArg string) []string {
		return []string{`{"type":"thread.started","thread_id":"cli-session"}` + "\n" + `{"type":"item.completed","item":{"type":"agent_message","text":"ok"}}`}
	})
	defer restore()

	var exitCode int
	output := captureOutput(t, func() { exitCode = run() })

	if exitCode != 0 {
		t.Fatalf("run() exit=%d, want 0", exitCode)
	}
	if !strings.Contains(output, "ok") || !strings.Contains(output, "SESSION_ID: cli-session") {
		t.Fatalf("unexpected output: %q", output)
	}
}
