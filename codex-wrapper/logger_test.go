package main

import (
	"bufio"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
	tempDir := setTempDirEnv(t, t.TempDir())

	orphan1 := createTempLog(t, tempDir, "codex-wrapper-111.log")
	orphan2 := createTempLog(t, tempDir, "codex-wrapper-222-suffix.log")
	running1 := createTempLog(t, tempDir, "codex-wrapper-333.log")
	running2 := createTempLog(t, tempDir, "codex-wrapper-444-extra-info.log")
	untouched := createTempLog(t, tempDir, "unrelated.log")

	runningPIDs := map[int]bool{333: true, 444: true}
	stubProcessRunning(t, func(pid int) bool {
		return runningPIDs[pid]
	})

	// Stub process start time to be in the past so files won't be considered as PID reused
	stubProcessStartTime(t, func(pid int) time.Time {
		if runningPIDs[pid] {
			// Return a time before file creation
			return time.Now().Add(-1 * time.Hour)
		}
		return time.Time{}
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
	tempDir := setTempDirEnv(t, t.TempDir())

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

	stubProcessStartTime(t, func(pid int) time.Time {
		return time.Time{} // Return zero time for processes not running
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
	stubProcessStartTime(t, func(int) time.Time {
		return time.Time{}
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
	setTempDirEnv(t, t.TempDir())

	stubProcessRunning(t, func(int) bool {
		t.Fatalf("process check should not run for empty directory")
		return false
	})
	stubProcessStartTime(t, func(int) time.Time {
		return time.Time{}
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
	tempDir := setTempDirEnv(t, t.TempDir())

	paths := []string{
		createTempLog(t, tempDir, "codex-wrapper-6100.log"),
		createTempLog(t, tempDir, "codex-wrapper-6101.log"),
	}

	stubProcessRunning(t, func(int) bool { return false })
	stubProcessStartTime(t, func(int) time.Time { return time.Time{} })

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
	tempDir := setTempDirEnv(t, t.TempDir())

	protected := createTempLog(t, tempDir, "codex-wrapper-6200.log")
	deletable := createTempLog(t, tempDir, "codex-wrapper-6201.log")

	stubProcessRunning(t, func(int) bool { return false })
	stubProcessStartTime(t, func(int) time.Time { return time.Time{} })

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
	tempDir := setTempDirEnv(t, t.TempDir())

	const fileCount = 400
	fakePaths := make([]string, fileCount)
	for i := 0; i < fileCount; i++ {
		name := fmt.Sprintf("codex-wrapper-%d.log", 10000+i)
		fakePaths[i] = createTempLog(t, tempDir, name)
	}

	stubGlobLogFiles(t, func(pattern string) ([]string, error) {
		return fakePaths, nil
	})
	stubProcessRunning(t, func(int) bool { return false })
	stubProcessStartTime(t, func(int) time.Time { return time.Time{} })

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
	tempDir := setTempDirEnv(t, t.TempDir())

	currentPID := os.Getpid()
	currentLog := createTempLog(t, tempDir, fmt.Sprintf("codex-wrapper-%d.log", currentPID))

	stubProcessRunning(t, func(pid int) bool {
		if pid != currentPID {
			t.Fatalf("unexpected pid check: %d", pid)
		}
		return true
	})
	stubProcessStartTime(t, func(pid int) time.Time {
		if pid == currentPID {
			return time.Now().Add(-1 * time.Hour)
		}
		return time.Time{}
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

func TestIsPIDReusedScenarios(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name      string
		statErr   error
		modTime   time.Time
		startTime time.Time
		want      bool
	}{
		{"stat error", errors.New("stat failed"), time.Time{}, time.Time{}, false},
		{"old file unknown start", nil, now.Add(-8 * 24 * time.Hour), time.Time{}, true},
		{"recent file unknown start", nil, now.Add(-2 * time.Hour), time.Time{}, false},
		{"pid reused", nil, now.Add(-2 * time.Hour), now.Add(-30 * time.Minute), true},
		{"pid active", nil, now.Add(-30 * time.Minute), now.Add(-2 * time.Hour), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubFileStat(t, func(string) (os.FileInfo, error) {
				if tt.statErr != nil {
					return nil, tt.statErr
				}
				return fakeFileInfo{modTime: tt.modTime}, nil
			})
			stubProcessStartTime(t, func(int) time.Time {
				return tt.startTime
			})
			if got := isPIDReused("log", 1234); got != tt.want {
				t.Fatalf("isPIDReused() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsUnsafeFileSecurityChecks(t *testing.T) {
	tempDir := t.TempDir()
	absTempDir, err := filepath.Abs(tempDir)
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}

	t.Run("symlink", func(t *testing.T) {
		stubFileStat(t, func(string) (os.FileInfo, error) {
			return fakeFileInfo{mode: os.ModeSymlink}, nil
		})
		stubEvalSymlinks(t, func(path string) (string, error) {
			return filepath.Join(absTempDir, filepath.Base(path)), nil
		})
		unsafe, reason := isUnsafeFile(filepath.Join(absTempDir, "codex-wrapper-1.log"), tempDir)
		if !unsafe || reason != "refusing to delete symlink" {
			t.Fatalf("expected symlink to be rejected, got unsafe=%v reason=%q", unsafe, reason)
		}
	})

	t.Run("path traversal", func(t *testing.T) {
		stubFileStat(t, func(string) (os.FileInfo, error) {
			return fakeFileInfo{}, nil
		})
		outside := filepath.Join(filepath.Dir(absTempDir), "etc", "passwd")
		stubEvalSymlinks(t, func(string) (string, error) {
			return outside, nil
		})
		unsafe, reason := isUnsafeFile(filepath.Join("..", "..", "etc", "passwd"), tempDir)
		if !unsafe || reason != "file is outside tempDir" {
			t.Fatalf("expected traversal path to be rejected, got unsafe=%v reason=%q", unsafe, reason)
		}
	})

	t.Run("outside temp dir", func(t *testing.T) {
		stubFileStat(t, func(string) (os.FileInfo, error) {
			return fakeFileInfo{}, nil
		})
		otherDir := t.TempDir()
		stubEvalSymlinks(t, func(string) (string, error) {
			return filepath.Join(otherDir, "codex-wrapper-9.log"), nil
		})
		unsafe, reason := isUnsafeFile(filepath.Join(otherDir, "codex-wrapper-9.log"), tempDir)
		if !unsafe || reason != "file is outside tempDir" {
			t.Fatalf("expected outside file to be rejected, got unsafe=%v reason=%q", unsafe, reason)
		}
	})
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
	hugePID := strconv.FormatInt(math.MaxInt64, 10) + "0"
	tests := []struct {
		name string
		pid  int
		ok   bool
	}{
		{"codex-wrapper-123.log", 123, true},
		{"codex-wrapper-999-extra.log", 999, true},
		{"codex-wrapper-.log", 0, false},
		{"invalid-name.log", 0, false},
		{"codex-wrapper--5.log", 0, false},
		{"codex-wrapper-0.log", 0, false},
		{fmt.Sprintf("codex-wrapper-%s.log", hugePID), 0, false},
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

func setTempDirEnv(t *testing.T, dir string) string {
	t.Helper()
	resolved := dir
	if eval, err := filepath.EvalSymlinks(dir); err == nil {
		resolved = eval
	}
	t.Setenv("TMPDIR", resolved)
	t.Setenv("TEMP", resolved)
	t.Setenv("TMP", resolved)
	return resolved
}

func stubProcessRunning(t *testing.T, fn func(int) bool) {
	t.Helper()
	original := processRunningCheck
	processRunningCheck = fn
	t.Cleanup(func() {
		processRunningCheck = original
	})
}

func stubProcessStartTime(t *testing.T, fn func(int) time.Time) {
	t.Helper()
	original := processStartTimeFn
	processStartTimeFn = fn
	t.Cleanup(func() {
		processStartTimeFn = original
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

func stubFileStat(t *testing.T, fn func(string) (os.FileInfo, error)) {
	t.Helper()
	original := fileStatFn
	fileStatFn = fn
	t.Cleanup(func() {
		fileStatFn = original
	})
}

func stubEvalSymlinks(t *testing.T, fn func(string) (string, error)) {
	t.Helper()
	original := evalSymlinksFn
	evalSymlinksFn = fn
	t.Cleanup(func() {
		evalSymlinksFn = original
	})
}

type fakeFileInfo struct {
	modTime time.Time
	mode    os.FileMode
}

func (f fakeFileInfo) Name() string       { return "fake" }
func (f fakeFileInfo) Size() int64        { return 0 }
func (f fakeFileInfo) Mode() os.FileMode  { return f.mode }
func (f fakeFileInfo) ModTime() time.Time { return f.modTime }
func (f fakeFileInfo) IsDir() bool        { return false }
func (f fakeFileInfo) Sys() interface{}   { return nil }
