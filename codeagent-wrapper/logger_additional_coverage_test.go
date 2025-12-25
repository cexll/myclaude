package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoggerNilReceiverNoop(t *testing.T) {
	var logger *Logger
	logger.Info("info")
	logger.Warn("warn")
	logger.Debug("debug")
	logger.Error("error")
	logger.Flush()
	if err := logger.Close(); err != nil {
		t.Fatalf("Close() on nil logger should return nil, got %v", err)
	}
}

func TestLoggerConcurrencyLogHelpers(t *testing.T) {
	setTempDirEnv(t, t.TempDir())

	logger, err := NewLoggerWithSuffix("concurrency")
	if err != nil {
		t.Fatalf("NewLoggerWithSuffix error: %v", err)
	}
	setLogger(logger)
	defer closeLogger()

	logConcurrencyPlanning(0, 2)
	logConcurrencyPlanning(3, 2)
	logConcurrencyState("start", "task-1", 1, 0)
	logConcurrencyState("done", "task-1", 0, 3)
	logger.Flush()

	data, err := os.ReadFile(logger.Path())
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	output := string(data)

	checks := []string{
		"parallel: worker_limit=unbounded total_tasks=2",
		"parallel: worker_limit=3 total_tasks=2",
		"parallel: start task=task-1 active=1 limit=unbounded",
		"parallel: done task=task-1 active=0 limit=3",
	}
	for _, c := range checks {
		if !strings.Contains(output, c) {
			t.Fatalf("log output missing %q, got: %s", c, output)
		}
	}
}

func TestLoggerConcurrencyLogHelpersNoopWithoutActiveLogger(t *testing.T) {
	_ = closeLogger()
	logConcurrencyPlanning(1, 1)
	logConcurrencyState("start", "task-1", 0, 1)
}

func TestLoggerCleanupOldLogsSkipsUnsafeAndHandlesAlreadyDeleted(t *testing.T) {
	tempDir := setTempDirEnv(t, t.TempDir())

	unsafePath := createTempLog(t, tempDir, fmt.Sprintf("%s-%d.log", primaryLogPrefix(), 222))
	orphanPath := createTempLog(t, tempDir, fmt.Sprintf("%s-%d.log", primaryLogPrefix(), 111))

	stubFileStat(t, func(path string) (os.FileInfo, error) {
		if path == unsafePath {
			return fakeFileInfo{mode: os.ModeSymlink}, nil
		}
		return os.Lstat(path)
	})

	stubProcessRunning(t, func(pid int) bool {
		if pid == 111 {
			_ = os.Remove(orphanPath)
		}
		return false
	})

	stats, err := cleanupOldLogs()
	if err != nil {
		t.Fatalf("cleanupOldLogs() unexpected error: %v", err)
	}

	if stats.Scanned != 2 {
		t.Fatalf("scanned = %d, want %d", stats.Scanned, 2)
	}
	if stats.Deleted != 0 {
		t.Fatalf("deleted = %d, want %d", stats.Deleted, 0)
	}
	if stats.Kept != 2 {
		t.Fatalf("kept = %d, want %d", stats.Kept, 2)
	}
	if stats.Errors != 0 {
		t.Fatalf("errors = %d, want %d", stats.Errors, 0)
	}

	hasSkip := false
	hasAlreadyDeleted := false
	for _, name := range stats.KeptFiles {
		if strings.Contains(name, "already deleted") {
			hasAlreadyDeleted = true
		}
		if strings.Contains(name, filepath.Base(unsafePath)) {
			hasSkip = true
		}
	}
	if !hasSkip {
		t.Fatalf("expected kept files to include unsafe log %q, got %+v", filepath.Base(unsafePath), stats.KeptFiles)
	}
	if !hasAlreadyDeleted {
		t.Fatalf("expected kept files to include already deleted marker, got %+v", stats.KeptFiles)
	}
}

func TestLoggerIsUnsafeFileErrorPaths(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("stat ErrNotExist", func(t *testing.T) {
		stubFileStat(t, func(string) (os.FileInfo, error) {
			return nil, os.ErrNotExist
		})

		unsafe, reason := isUnsafeFile("missing.log", tempDir)
		if !unsafe || reason != "" {
			t.Fatalf("expected missing file to be skipped silently, got unsafe=%v reason=%q", unsafe, reason)
		}
	})

	t.Run("stat error", func(t *testing.T) {
		stubFileStat(t, func(string) (os.FileInfo, error) {
			return nil, fmt.Errorf("boom")
		})

		unsafe, reason := isUnsafeFile("broken.log", tempDir)
		if !unsafe || !strings.Contains(reason, "stat failed") {
			t.Fatalf("expected stat failure to be unsafe, got unsafe=%v reason=%q", unsafe, reason)
		}
	})

	t.Run("EvalSymlinks error", func(t *testing.T) {
		stubFileStat(t, func(string) (os.FileInfo, error) {
			return fakeFileInfo{}, nil
		})
		stubEvalSymlinks(t, func(string) (string, error) {
			return "", fmt.Errorf("resolve failed")
		})

		unsafe, reason := isUnsafeFile("cannot-resolve.log", tempDir)
		if !unsafe || !strings.Contains(reason, "path resolution failed") {
			t.Fatalf("expected resolution failure to be unsafe, got unsafe=%v reason=%q", unsafe, reason)
		}
	})
}
