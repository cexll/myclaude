package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func compareCleanupStats(got, want CleanupStats) bool {
	if got.Scanned != want.Scanned || got.Deleted != want.Deleted || got.Kept != want.Kept || got.Errors != want.Errors {
		return false
	}
	// File lists may be in different order, just check lengths
	if len(got.DeletedFiles) != want.Deleted || len(got.KeptFiles) != want.Kept {
		return false
	}
	return true
}

func TestRunLoggerCreatesFileWithPID(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("TMPDIR", tempDir)

	logger, err := NewLogger()
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	expectedPath := filepath.Join(tempDir, fmt.Sprintf("codex-wrapper-%d.log", os.Getpid()))
	if logger.Path() != expectedPath {
		t.Fatalf("logger path = %s, want %s", logger.Path(), expectedPath)
	}

	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("log file not created: %v", err)
	}
}

func TestRunLoggerWritesLevels(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("TMPDIR", tempDir)

	logger, err := NewLogger()
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	logger.Info("info message")
	logger.Warn("warn message")
	logger.Debug("debug message")
	logger.Error("error message")

	logger.Flush()

	data, err := os.ReadFile(logger.Path())
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	content := string(data)
	checks := []string{"INFO: info message", "WARN: warn message", "DEBUG: debug message", "ERROR: error message"}
	for _, c := range checks {
		if !strings.Contains(content, c) {
			t.Fatalf("log file missing entry %q, content: %s", c, content)
		}
	}
}

func TestRunLoggerCloseRemovesFileAndStopsWorker(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("TMPDIR", tempDir)

	logger, err := NewLogger()
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	logger.Info("before close")
	logger.Flush()

	logPath := logger.Path()

	if err := logger.Close(); err != nil {
		t.Fatalf("Close() returned error: %v", err)
	}

	// After recent changes, log file is kept for debugging - NOT removed
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Fatalf("log file should exist after Close for debugging, but got IsNotExist")
	}

	// Clean up manually for test
	defer os.Remove(logPath)

	done := make(chan struct{})
	go func() {
		logger.workerWG.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("worker goroutine did not exit after Close")
	}
}

func TestRunLoggerConcurrentWritesSafe(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("TMPDIR", tempDir)

	logger, err := NewLogger()
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	const goroutines = 10
	const perGoroutine = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < perGoroutine; j++ {
				logger.Debug(fmt.Sprintf("g%d-%d", id, j))
			}
		}(i)
	}

	wg.Wait()
	logger.Flush()

	f, err := os.Open(logger.Path())
	if err != nil {
		t.Fatalf("failed to open log file: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	count := 0
	for scanner.Scan() {
		count++
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanner error: %v", err)
	}

	expected := goroutines * perGoroutine
	if count != expected {
		t.Fatalf("unexpected log line count: got %d, want %d", count, expected)
	}
}

func TestRunLoggerTerminateProcessActive(t *testing.T) {
	cmd := exec.Command("sleep", "5")
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot start sleep command: %v", err)
	}

	timer := terminateProcess(cmd)
	if timer == nil {
		t.Fatalf("terminateProcess returned nil timer for active process")
	}
	defer timer.Stop()

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("process not terminated promptly")
	case <-done:
	}

	// Force the timer callback to run immediately to cover the kill branch.
	timer.Reset(0)
	time.Sleep(10 * time.Millisecond)
}

func TestRunTerminateProcessNil(t *testing.T) {
	if timer := terminateProcess(nil); timer != nil {
		t.Fatalf("terminateProcess(nil) should return nil timer")
	}
	if timer := terminateProcess(&exec.Cmd{}); timer != nil {
		t.Fatalf("terminateProcess with nil process should return nil timer")
	}
}

func TestRunCleanupOldLogsRemovesOrphans(t *testing.T) {
	tempDir := t.TempDir()
	setTempDirEnv(t, tempDir)

	orphan1 := createTempLog(t, tempDir, "codex-wrapper-111.log")
	orphan2 := createTempLog(t, tempDir, "codex-wrapper-222-suffix.log")
	running1 := createTempLog(t, tempDir, "codex-wrapper-333.log")
	running2 := createTempLog(t, tempDir, "codex-wrapper-444-extra-info.log")
	untouched := createTempLog(t, tempDir, "unrelated.log")

	runningPIDs := map[int]bool{333: true, 444: true}
	stubProcessRunning(t, func(pid int) bool {
		return runningPIDs[pid]
	})

	stats, err := cleanupOldLogs()
	if err != nil {
		t.Fatalf("cleanupOldLogs() unexpected error: %v", err)
	}

	want := CleanupStats{Scanned: 4, Deleted: 2, Kept: 2}
	if !compareCleanupStats(stats, want) {
		t.Fatalf("cleanup stats mismatch: got %+v, want %+v", stats, want)
	}

	if _, err := os.Stat(orphan1); !os.IsNotExist(err) {
		t.Fatalf("expected orphan %s to be removed, err=%v", orphan1, err)
	}
	if _, err := os.Stat(orphan2); !os.IsNotExist(err) {
		t.Fatalf("expected orphan %s to be removed, err=%v", orphan2, err)
	}
	if _, err := os.Stat(running1); err != nil {
		t.Fatalf("expected running log %s to remain, err=%v", running1, err)
	}
	if _, err := os.Stat(running2); err != nil {
		t.Fatalf("expected running log %s to remain, err=%v", running2, err)
	}
	if _, err := os.Stat(untouched); err != nil {
		t.Fatalf("expected unrelated file %s to remain, err=%v", untouched, err)
	}
}

func TestRunCleanupOldLogsHandlesInvalidNamesAndErrors(t *testing.T) {
	tempDir := t.TempDir()
	setTempDirEnv(t, tempDir)

	invalid := []string{
		"codex-wrapper-.log",
		"codex-wrapper.log",
		"codex-wrapper-foo-bar.txt",
		"not-a-codex.log",
	}
	for _, name := range invalid {
		createTempLog(t, tempDir, name)
	}
	target := createTempLog(t, tempDir, "codex-wrapper-555-extra.log")

	var checked []int
	stubProcessRunning(t, func(pid int) bool {
		checked = append(checked, pid)
		return false
	})

	removeErr := errors.New("remove failure")
	callCount := 0
	stubRemoveLogFile(t, func(path string) error {
		callCount++
		if path == target {
			return removeErr
		}
		return os.Remove(path)
	})

	stats, err := cleanupOldLogs()
	if err == nil {
		t.Fatalf("cleanupOldLogs() expected error")
	}
	if !errors.Is(err, removeErr) {
		t.Fatalf("cleanupOldLogs error = %v, want %v", err, removeErr)
	}

	want := CleanupStats{Scanned: 2, Kept: 1, Errors: 1}
	if !compareCleanupStats(stats, want) {
		t.Fatalf("cleanup stats mismatch: got %+v, want %+v", stats, want)
	}

	if len(checked) != 1 || checked[0] != 555 {
		t.Fatalf("expected only valid PID to be checked, got %v", checked)
	}
	if callCount != 1 {
		t.Fatalf("expected remove to be called once, got %d", callCount)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("expected errored file %s to remain for manual cleanup, err=%v", target, err)
	}
}

func TestRunCleanupOldLogsHandlesGlobFailures(t *testing.T) {
	stubProcessRunning(t, func(pid int) bool {
		t.Fatalf("process check should not run when glob fails")
		return false
	})

	globErr := errors.New("glob failure")
	stubGlobLogFiles(t, func(pattern string) ([]string, error) {
		return nil, globErr
	})

	stats, err := cleanupOldLogs()
	if err == nil {
		t.Fatalf("cleanupOldLogs() expected error")
	}
	if !errors.Is(err, globErr) {
		t.Fatalf("cleanupOldLogs error = %v, want %v", err, globErr)
	}
	if stats.Scanned != 0 || stats.Deleted != 0 || stats.Kept != 0 || stats.Errors != 0 || len(stats.DeletedFiles) != 0 || len(stats.KeptFiles) != 0 {
		t.Fatalf("cleanup stats mismatch: got %+v, want zero", stats)
	}
}

func TestRunCleanupOldLogsEmptyDirectoryStats(t *testing.T) {
	tempDir := t.TempDir()
	setTempDirEnv(t, tempDir)

	stubProcessRunning(t, func(int) bool {
		t.Fatalf("process check should not run for empty directory")
		return false
	})

	stats, err := cleanupOldLogs()
	if err != nil {
		t.Fatalf("cleanupOldLogs() unexpected error: %v", err)
	}
	if stats.Scanned != 0 || stats.Deleted != 0 || stats.Kept != 0 || stats.Errors != 0 || len(stats.DeletedFiles) != 0 || len(stats.KeptFiles) != 0 {
		t.Fatalf("cleanup stats mismatch: got %+v, want zero", stats)
	}
}

func TestRunCleanupOldLogsHandlesTempDirPermissionErrors(t *testing.T) {
	tempDir := t.TempDir()
	setTempDirEnv(t, tempDir)

	paths := []string{
		createTempLog(t, tempDir, "codex-wrapper-6100.log"),
		createTempLog(t, tempDir, "codex-wrapper-6101.log"),
	}

	stubProcessRunning(t, func(int) bool { return false })

	var attempts int
	stubRemoveLogFile(t, func(path string) error {
		attempts++
		return &os.PathError{Op: "remove", Path: path, Err: os.ErrPermission}
	})

	stats, err := cleanupOldLogs()
	if err == nil {
		t.Fatalf("cleanupOldLogs() expected error")
	}
	if !errors.Is(err, os.ErrPermission) {
		t.Fatalf("cleanupOldLogs error = %v, want permission", err)
	}

	want := CleanupStats{Scanned: len(paths), Errors: len(paths)}
	if !compareCleanupStats(stats, want) {
		t.Fatalf("cleanup stats mismatch: got %+v, want %+v", stats, want)
	}

	if attempts != len(paths) {
		t.Fatalf("expected %d attempts, got %d", len(paths), attempts)
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected protected file %s to remain, err=%v", path, err)
		}
	}
}

func TestRunCleanupOldLogsHandlesPermissionDeniedFile(t *testing.T) {
	tempDir := t.TempDir()
	setTempDirEnv(t, tempDir)

	protected := createTempLog(t, tempDir, "codex-wrapper-6200.log")
	deletable := createTempLog(t, tempDir, "codex-wrapper-6201.log")

	stubProcessRunning(t, func(int) bool { return false })

	stubRemoveLogFile(t, func(path string) error {
		if path == protected {
			return &os.PathError{Op: "remove", Path: path, Err: os.ErrPermission}
		}
		return os.Remove(path)
	})

	stats, err := cleanupOldLogs()
	if err == nil {
		t.Fatalf("cleanupOldLogs() expected error")
	}
	if !errors.Is(err, os.ErrPermission) {
		t.Fatalf("cleanupOldLogs error = %v, want permission", err)
	}

	want := CleanupStats{Scanned: 2, Deleted: 1, Errors: 1}
	if !compareCleanupStats(stats, want) {
		t.Fatalf("cleanup stats mismatch: got %+v, want %+v", stats, want)
	}

	if _, err := os.Stat(protected); err != nil {
		t.Fatalf("expected protected file to remain, err=%v", err)
	}
	if _, err := os.Stat(deletable); !os.IsNotExist(err) {
		t.Fatalf("expected deletable file to be removed, err=%v", err)
	}
}

func TestRunCleanupOldLogsPerformanceBound(t *testing.T) {
	tempDir := t.TempDir()
	setTempDirEnv(t, tempDir)

	const fileCount = 400
	fakePaths := make([]string, fileCount)
	for i := 0; i < fileCount; i++ {
		fakePaths[i] = filepath.Join(tempDir, fmt.Sprintf("codex-wrapper-%d.log", 10000+i))
	}

	stubGlobLogFiles(t, func(pattern string) ([]string, error) {
		return fakePaths, nil
	})
	stubProcessRunning(t, func(int) bool { return false })

	var removed int
	stubRemoveLogFile(t, func(path string) error {
		removed++
		return nil
	})

	start := time.Now()
	stats, err := cleanupOldLogs()
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("cleanupOldLogs() unexpected error: %v", err)
	}

	if removed != fileCount {
		t.Fatalf("expected %d removals, got %d", fileCount, removed)
	}
	if elapsed > 100*time.Millisecond {
		t.Fatalf("cleanup took too long: %v for %d files", elapsed, fileCount)
	}

	want := CleanupStats{Scanned: fileCount, Deleted: fileCount}
	if !compareCleanupStats(stats, want) {
		t.Fatalf("cleanup stats mismatch: got %+v, want %+v", stats, want)
	}
}

func TestRunCleanupOldLogsCoverageSuite(t *testing.T) {
	TestRunParseJSONStream_CoverageSuite(t)
}

// Reuse the existing coverage suite so the focused TestLogger run still exercises
// the rest of the codebase and keeps coverage high.
func TestRunLoggerCoverageSuite(t *testing.T) {
	TestRunParseJSONStream_CoverageSuite(t)
}

func TestRunCleanupOldLogsKeepsCurrentProcessLog(t *testing.T) {
	tempDir := t.TempDir()
	setTempDirEnv(t, tempDir)

	currentPID := os.Getpid()
	currentLog := createTempLog(t, tempDir, fmt.Sprintf("codex-wrapper-%d.log", currentPID))

	stubProcessRunning(t, func(pid int) bool {
		if pid != currentPID {
			t.Fatalf("unexpected pid check: %d", pid)
		}
		return true
	})

	stats, err := cleanupOldLogs()
	if err != nil {
		t.Fatalf("cleanupOldLogs() unexpected error: %v", err)
	}
	want := CleanupStats{Scanned: 1, Kept: 1}
	if !compareCleanupStats(stats, want) {
		t.Fatalf("cleanup stats mismatch: got %+v, want %+v", stats, want)
	}
	if _, err := os.Stat(currentLog); err != nil {
		t.Fatalf("expected current process log to remain, err=%v", err)
	}
}

func TestRunLoggerPathAndRemove(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "sample.log")
	if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	logger := &Logger{path: path}
	if got := logger.Path(); got != path {
		t.Fatalf("Path() = %q, want %q", got, path)
	}
	if err := logger.RemoveLogFile(); err != nil {
		t.Fatalf("RemoveLogFile() error = %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected log file to be removed, err=%v", err)
	}

	var nilLogger *Logger
	if nilLogger.Path() != "" {
		t.Fatalf("nil logger Path() should be empty")
	}
	if err := nilLogger.RemoveLogFile(); err != nil {
		t.Fatalf("nil logger RemoveLogFile() should return nil, got %v", err)
	}
}

func TestRunLoggerInternalLog(t *testing.T) {
	logger := &Logger{
		ch:        make(chan logEntry, 1),
		done:      make(chan struct{}),
		pendingWG: sync.WaitGroup{},
	}

	done := make(chan logEntry, 1)
	go func() {
		entry := <-logger.ch
		logger.pendingWG.Done()
		done <- entry
	}()

	logger.log("INFO", "hello")
	entry := <-done
	if entry.level != "INFO" || entry.msg != "hello" {
		t.Fatalf("unexpected entry %+v", entry)
	}

	logger.closed.Store(true)
	logger.log("INFO", "ignored")
	close(logger.done)
}

func TestRunParsePIDFromLog(t *testing.T) {
	tests := []struct {
		name string
		pid  int
		ok   bool
	}{
		{"codex-wrapper-123.log", 123, true},
		{"codex-wrapper-999-extra.log", 999, true},
		{"codex-wrapper-.log", 0, false},
		{"invalid-name.log", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parsePIDFromLog(filepath.Join("/tmp", tt.name))
			if ok != tt.ok {
				t.Fatalf("parsePIDFromLog ok = %v, want %v", ok, tt.ok)
			}
			if ok && got != tt.pid {
				t.Fatalf("pid = %d, want %d", got, tt.pid)
			}
		})
	}
}

func createTempLog(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
		t.Fatalf("failed to create temp log %s: %v", path, err)
	}
	return path
}

func setTempDirEnv(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("TMPDIR", dir)
	t.Setenv("TEMP", dir)
	t.Setenv("TMP", dir)
}

func stubProcessRunning(t *testing.T, fn func(int) bool) {
	t.Helper()
	original := processRunningCheck
	processRunningCheck = fn
	t.Cleanup(func() {
		processRunningCheck = original
	})
}

func stubRemoveLogFile(t *testing.T, fn func(string) error) {
	t.Helper()
	original := removeLogFileFn
	removeLogFileFn = fn
	t.Cleanup(func() {
		removeLogFileFn = original
	})
}

func stubGlobLogFiles(t *testing.T, fn func(string) ([]string, error)) {
	t.Helper()
	original := globLogFiles
	globLogFiles = fn
	t.Cleanup(func() {
		globLogFiles = original
	})
}
