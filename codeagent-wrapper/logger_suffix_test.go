package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoggerWithSuffixNamingAndIsolation(t *testing.T) {
	tempDir := setTempDirEnv(t, t.TempDir())

	taskA := "task-1"
	taskB := "task-2"

	loggerA, err := NewLoggerWithSuffix(taskA)
	if err != nil {
		t.Fatalf("NewLoggerWithSuffix(%q) error = %v", taskA, err)
	}
	defer loggerA.Close()

	loggerB, err := NewLoggerWithSuffix(taskB)
	if err != nil {
		t.Fatalf("NewLoggerWithSuffix(%q) error = %v", taskB, err)
	}
	defer loggerB.Close()

	wantA := filepath.Join(tempDir, fmt.Sprintf("%s-%d-%s.log", primaryLogPrefix(), os.Getpid(), taskA))
	if loggerA.Path() != wantA {
		t.Fatalf("loggerA path = %q, want %q", loggerA.Path(), wantA)
	}

	wantB := filepath.Join(tempDir, fmt.Sprintf("%s-%d-%s.log", primaryLogPrefix(), os.Getpid(), taskB))
	if loggerB.Path() != wantB {
		t.Fatalf("loggerB path = %q, want %q", loggerB.Path(), wantB)
	}

	if loggerA.Path() == loggerB.Path() {
		t.Fatalf("expected different log files, got %q", loggerA.Path())
	}

	loggerA.Info("from taskA")
	loggerB.Info("from taskB")
	loggerA.Flush()
	loggerB.Flush()

	dataA, err := os.ReadFile(loggerA.Path())
	if err != nil {
		t.Fatalf("failed to read loggerA file: %v", err)
	}
	dataB, err := os.ReadFile(loggerB.Path())
	if err != nil {
		t.Fatalf("failed to read loggerB file: %v", err)
	}

	if !strings.Contains(string(dataA), "from taskA") {
		t.Fatalf("loggerA missing its message, got: %q", string(dataA))
	}
	if strings.Contains(string(dataA), "from taskB") {
		t.Fatalf("loggerA contains loggerB message, got: %q", string(dataA))
	}
	if !strings.Contains(string(dataB), "from taskB") {
		t.Fatalf("loggerB missing its message, got: %q", string(dataB))
	}
	if strings.Contains(string(dataB), "from taskA") {
		t.Fatalf("loggerB contains loggerA message, got: %q", string(dataB))
	}
}

func TestLoggerWithSuffixReturnsErrorWhenTempDirNotWritable(t *testing.T) {
	base := t.TempDir()
	noWrite := filepath.Join(base, "ro")
	if err := os.Mkdir(noWrite, 0o500); err != nil {
		t.Fatalf("failed to create read-only temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(noWrite, 0o700) })
	setTempDirEnv(t, noWrite)

	logger, err := NewLoggerWithSuffix("task-err")
	if err == nil {
		_ = logger.Close()
		t.Fatalf("expected error when temp dir is not writable")
	}
}

func TestLoggerWithSuffixSanitizesUnsafeSuffix(t *testing.T) {
	tempDir := setTempDirEnv(t, t.TempDir())

	raw := "../bad id/with?chars"
	safe := sanitizeLogSuffix(raw)
	if safe == "" {
		t.Fatalf("sanitizeLogSuffix returned empty string")
	}
	if strings.ContainsAny(safe, "/\\") {
		t.Fatalf("sanitized suffix should not contain path separators, got %q", safe)
	}

	logger, err := NewLoggerWithSuffix(raw)
	if err != nil {
		t.Fatalf("NewLoggerWithSuffix(%q) error = %v", raw, err)
	}
	t.Cleanup(func() {
		_ = logger.Close()
		_ = os.Remove(logger.Path())
	})

	wantBase := fmt.Sprintf("%s-%d-%s.log", primaryLogPrefix(), os.Getpid(), safe)
	if gotBase := filepath.Base(logger.Path()); gotBase != wantBase {
		t.Fatalf("log filename = %q, want %q", gotBase, wantBase)
	}
	if dir := filepath.Dir(logger.Path()); dir != tempDir {
		t.Fatalf("logger path dir = %q, want %q", dir, tempDir)
	}
}
