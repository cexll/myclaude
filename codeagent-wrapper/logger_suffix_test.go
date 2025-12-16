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

func TestLoggerWithSuffixReturnsErrorWhenTempDirMissing(t *testing.T) {
	missingTempDir := filepath.Join(t.TempDir(), "does-not-exist")
	setTempDirEnv(t, missingTempDir)

	logger, err := NewLoggerWithSuffix("task-err")
	if err == nil {
		_ = logger.Close()
		t.Fatalf("expected error, got nil")
	}
}
