package logger

import (
	"bufio"
	"errors"
	"fmt"
	"math"
	"os"
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

func TestLoggerCreatesFileWithPID(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("TMPDIR", tempDir)

	logger, err := NewLogger()
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	expectedPath := filepath.Join(tempDir, fmt.Sprintf("codeagent-wrapper-%d.log", os.Getpid()))
	if logger.Path() != expectedPath {
		t.Fatalf("logger path = %s, want %s", logger.Path(), expectedPath)
	}

	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("log file not created: %v", err)
	}
}

func TestLoggerWritesLevels(t *testing.T) {
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
	checks := []string{"info message", "warn message", "debug message", "error message"}
	for _, c := range checks {
		if !strings.Contains(content, c) {
			t.Fatalf("log file missing entry %q, content: %s", c, content)
		}
	}
}

func TestLoggerCloseStopsWorkerAndKeepsFile(t *testing.T) {
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
}

func TestLoggerConcurrentWritesSafe(t *testing.T) {
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

func TestLoggerCleanupOldLogsRemovesOrphans(t *testing.T) {
	tempDir := setTempDirEnv(t, t.TempDir())

	orphan1 := createTempLog(t, tempDir, "codeagent-wrapper-111.log")
	orphan2 := createTempLog(t, tempDir, "codeagent-wrapper-222-suffix.log")
	running1 := createTempLog(t, tempDir, "codeagent-wrapper-333.log")
	running2 := createTempLog(t, tempDir, "codeagent-wrapper-444-extra-info.log")
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

func TestLoggerCleanupOldLogsHandlesInvalidNamesAndErrors(t *testing.T) {
	tempDir := setTempDirEnv(t, t.TempDir())

	invalid := []string{
		"codeagent-wrapper-.log",
		"codeagent-wrapper.log",
		"codeagent-wrapper-foo-bar.txt",
		"not-a-codex.log",
	}
	for _, name := range invalid {
		createTempLog(t, tempDir, name)
	}
	target := createTempLog(t, tempDir, "codeagent-wrapper-555-extra.log")

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

func TestLoggerCleanupOldLogsHandlesGlobFailures(t *testing.T) {
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

func TestLoggerCleanupOldLogsEmptyDirectoryStats(t *testing.T) {
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

func TestLoggerCleanupOldLogsHandlesTempDirPermissionErrors(t *testing.T) {
	tempDir := setTempDirEnv(t, t.TempDir())

	paths := []string{
		createTempLog(t, tempDir, "codeagent-wrapper-6100.log"),
		createTempLog(t, tempDir, "codeagent-wrapper-6101.log"),
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

func TestLoggerCleanupOldLogsHandlesPermissionDeniedFile(t *testing.T) {
	tempDir := setTempDirEnv(t, t.TempDir())

	protected := createTempLog(t, tempDir, "codeagent-wrapper-6200.log")
	deletable := createTempLog(t, tempDir, "codeagent-wrapper-6201.log")

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

func TestLoggerCleanupOldLogsPerformanceBound(t *testing.T) {
	tempDir := setTempDirEnv(t, t.TempDir())

	const fileCount = 400
	fakePaths := make([]string, fileCount)
	for i := 0; i < fileCount; i++ {
		name := fmt.Sprintf("codeagent-wrapper-%d.log", 10000+i)
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

func TestLoggerCleanupOldLogsKeepsCurrentProcessLog(t *testing.T) {
	tempDir := setTempDirEnv(t, t.TempDir())

	currentPID := os.Getpid()
	currentLog := createTempLog(t, tempDir, fmt.Sprintf("codeagent-wrapper-%d.log", currentPID))

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

func TestLoggerIsPIDReusedScenarios(t *testing.T) {
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

func TestLoggerIsUnsafeFileSecurityChecks(t *testing.T) {
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
		unsafe, reason := isUnsafeFile(filepath.Join(absTempDir, "codeagent-wrapper-1.log"), tempDir)
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
			return filepath.Join(otherDir, "codeagent-wrapper-9.log"), nil
		})
		unsafe, reason := isUnsafeFile(filepath.Join(otherDir, "codeagent-wrapper-9.log"), tempDir)
		if !unsafe || reason != "file is outside tempDir" {
			t.Fatalf("expected outside file to be rejected, got unsafe=%v reason=%q", unsafe, reason)
		}
	})
}

func TestLoggerPathAndRemove(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("TMPDIR", tempDir)

	logger, err := NewLoggerWithSuffix("sample")
	if err != nil {
		t.Fatalf("NewLoggerWithSuffix() error = %v", err)
	}
	path := logger.Path()
	if path == "" {
		_ = logger.Close()
		t.Fatalf("logger.Path() returned empty path")
	}
	if err := logger.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
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

func TestLoggerParsePIDFromLog(t *testing.T) {
	hugePID := strconv.FormatInt(math.MaxInt64, 10) + "0"
	tests := []struct {
		name string
		pid  int
		ok   bool
	}{
		{"codeagent-wrapper-123.log", 123, true},
		{"codeagent-wrapper-999-extra.log", 999, true},
		{"codeagent-wrapper-.log", 0, false},
		{"invalid-name.log", 0, false},
		{"codeagent-wrapper--5.log", 0, false},
		{"codeagent-wrapper-0.log", 0, false},
		{fmt.Sprintf("codeagent-wrapper-%s.log", hugePID), 0, false},
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
	t.Cleanup(SetProcessRunningCheck(fn))
}

func stubProcessStartTime(t *testing.T, fn func(int) time.Time) {
	t.Helper()
	t.Cleanup(SetProcessStartTimeFn(fn))
}

func stubRemoveLogFile(t *testing.T, fn func(string) error) {
	t.Helper()
	t.Cleanup(SetRemoveLogFileFn(fn))
}

func stubGlobLogFiles(t *testing.T, fn func(string) ([]string, error)) {
	t.Helper()
	t.Cleanup(SetGlobLogFilesFn(fn))
}

func stubFileStat(t *testing.T, fn func(string) (os.FileInfo, error)) {
	t.Helper()
	t.Cleanup(SetFileStatFn(fn))
}

func stubEvalSymlinks(t *testing.T, fn func(string) (string, error)) {
	t.Helper()
	t.Cleanup(SetEvalSymlinksFn(fn))
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

func TestLoggerExtractRecentErrors(t *testing.T) {
	tests := []struct {
		name       string
		logs       []struct{ level, msg string }
		maxEntries int
		want       []string
	}{
		{
			name:       "empty log",
			logs:       nil,
			maxEntries: 10,
			want:       nil,
		},
		{
			name: "no errors",
			logs: []struct{ level, msg string }{
				{"INFO", "started"},
				{"DEBUG", "processing"},
			},
			maxEntries: 10,
			want:       nil,
		},
		{
			name: "single error",
			logs: []struct{ level, msg string }{
				{"INFO", "started"},
				{"ERROR", "something failed"},
			},
			maxEntries: 10,
			want:       []string{"something failed"},
		},
		{
			name: "error and warn",
			logs: []struct{ level, msg string }{
				{"INFO", "started"},
				{"WARN", "warning message"},
				{"ERROR", "error message"},
			},
			maxEntries: 10,
			want: []string{
				"warning message",
				"error message",
			},
		},
		{
			name: "truncate to max",
			logs: []struct{ level, msg string }{
				{"ERROR", "error 1"},
				{"ERROR", "error 2"},
				{"ERROR", "error 3"},
				{"ERROR", "error 4"},
				{"ERROR", "error 5"},
			},
			maxEntries: 3,
			want: []string{
				"error 3",
				"error 4",
				"error 5",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewLoggerWithSuffix("extract-test")
			if err != nil {
				t.Fatalf("NewLoggerWithSuffix() error = %v", err)
			}
			defer logger.Close()
			defer func() { _ = logger.RemoveLogFile() }()

			// Write logs using logger methods
			for _, entry := range tt.logs {
				switch entry.level {
				case "INFO":
					logger.Info(entry.msg)
				case "WARN":
					logger.Warn(entry.msg)
				case "ERROR":
					logger.Error(entry.msg)
				case "DEBUG":
					logger.Debug(entry.msg)
				}
			}

			logger.Flush()

			got := logger.ExtractRecentErrors(tt.maxEntries)

			if len(got) != len(tt.want) {
				t.Fatalf("ExtractRecentErrors() got %d entries, want %d", len(got), len(tt.want))
			}
			for i, entry := range got {
				if entry != tt.want[i] {
					t.Errorf("entry[%d] = %q, want %q", i, entry, tt.want[i])
				}
			}
		})
	}
}

func TestLoggerExtractRecentErrorsNilLogger(t *testing.T) {
	var logger *Logger
	if got := logger.ExtractRecentErrors(10); got != nil {
		t.Fatalf("nil logger ExtractRecentErrors() should return nil, got %v", got)
	}
}

func TestLoggerExtractRecentErrorsEmptyPath(t *testing.T) {
	logger := &Logger{}
	if got := logger.ExtractRecentErrors(10); got != nil {
		t.Fatalf("empty path ExtractRecentErrors() should return nil, got %v", got)
	}
}

func TestLoggerExtractRecentErrorsFileNotExist(t *testing.T) {
	logger := &Logger{}
	if got := logger.ExtractRecentErrors(10); got != nil {
		t.Fatalf("nonexistent file ExtractRecentErrors() should return nil, got %v", got)
	}
}

func TestSanitizeLogSuffixNoDuplicates(t *testing.T) {
	testCases := []string{
		"task",
		"task.",
		".task",
		"-task",
		"task-",
		"--task--",
		"..task..",
	}

	seen := make(map[string]string)
	for _, input := range testCases {
		result := sanitizeLogSuffix(input)
		if result == "" {
			t.Fatalf("sanitizeLogSuffix(%q) returned empty string", input)
		}

		if prev, exists := seen[result]; exists {
			t.Fatalf("collision detected: %q and %q both produce %q", input, prev, result)
		}
		seen[result] = input

		// Verify result is safe for file names
		if strings.ContainsAny(result, "/\\:*?\"<>|") {
			t.Fatalf("sanitizeLogSuffix(%q) = %q contains unsafe characters", input, result)
		}
	}
}

func TestExtractRecentErrorsBoundaryCheck(t *testing.T) {
	logger, err := NewLoggerWithSuffix("boundary-test")
	if err != nil {
		t.Fatalf("NewLoggerWithSuffix() error = %v", err)
	}
	defer logger.Close()
	defer func() { _ = logger.RemoveLogFile() }()

	// Write some errors
	logger.Error("error 1")
	logger.Warn("warn 1")
	logger.Error("error 2")
	logger.Flush()

	// Test zero
	result := logger.ExtractRecentErrors(0)
	if result != nil {
		t.Fatalf("ExtractRecentErrors(0) should return nil, got %v", result)
	}

	// Test negative
	result = logger.ExtractRecentErrors(-5)
	if result != nil {
		t.Fatalf("ExtractRecentErrors(-5) should return nil, got %v", result)
	}

	// Test positive still works
	result = logger.ExtractRecentErrors(10)
	if len(result) != 3 {
		t.Fatalf("ExtractRecentErrors(10) expected 3 entries, got %d", len(result))
	}
}

func TestErrorEntriesMaxLimit(t *testing.T) {
	logger, err := NewLoggerWithSuffix("max-limit-test")
	if err != nil {
		t.Fatalf("NewLoggerWithSuffix() error = %v", err)
	}
	defer logger.Close()
	defer func() { _ = logger.RemoveLogFile() }()

	// Write 150 error/warn entries
	for i := 1; i <= 150; i++ {
		if i%2 == 0 {
			logger.Error(fmt.Sprintf("error-%03d", i))
		} else {
			logger.Warn(fmt.Sprintf("warn-%03d", i))
		}
	}
	logger.Flush()

	// Extract all cached errors
	result := logger.ExtractRecentErrors(200) // Request more than cache size

	// Should only have last 100 entries (entries 51-150 in sequence)
	if len(result) != 100 {
		t.Fatalf("expected 100 cached entries, got %d", len(result))
	}

	// Verify entries are the last 100 (entries 51-150)
	if !strings.Contains(result[0], "051") {
		t.Fatalf("first cached entry should be entry 51, got: %s", result[0])
	}
	if !strings.Contains(result[99], "150") {
		t.Fatalf("last cached entry should be entry 150, got: %s", result[99])
	}

	// Verify order is preserved - simplified logic
	for i := 0; i < len(result)-1; i++ {
		expectedNum := 51 + i
		nextNum := 51 + i + 1

		expectedEntry := fmt.Sprintf("%03d", expectedNum)
		nextEntry := fmt.Sprintf("%03d", nextNum)

		if !strings.Contains(result[i], expectedEntry) {
			t.Fatalf("entry at index %d should contain %s, got: %s", i, expectedEntry, result[i])
		}
		if !strings.Contains(result[i+1], nextEntry) {
			t.Fatalf("entry at index %d should contain %s, got: %s", i+1, nextEntry, result[i+1])
		}
	}
}
