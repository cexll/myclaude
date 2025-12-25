package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Logger writes log messages asynchronously to a temp file.
// It is intentionally minimal: a buffered channel + single worker goroutine
// to avoid contention while keeping ordering guarantees.
type Logger struct {
	path         string
	file         *os.File
	writer       *bufio.Writer
	ch           chan logEntry
	flushReq     chan chan struct{}
	done         chan struct{}
	closed       atomic.Bool
	closeOnce    sync.Once
	workerWG     sync.WaitGroup
	pendingWG    sync.WaitGroup
	flushMu      sync.Mutex
	workerErr    error
	errorEntries []string // Cache of recent ERROR/WARN entries
	errorMu      sync.Mutex
}

type logEntry struct {
	msg     string
	isError bool // true for ERROR or WARN levels
}

// CleanupStats captures the outcome of a cleanupOldLogs run.
type CleanupStats struct {
	Scanned      int
	Deleted      int
	Kept         int
	Errors       int
	DeletedFiles []string
	KeptFiles    []string
}

var (
	processRunningCheck = isProcessRunning
	processStartTimeFn  = getProcessStartTime
	removeLogFileFn     = os.Remove
	globLogFiles        = filepath.Glob
	fileStatFn          = os.Lstat // Use Lstat to detect symlinks
	evalSymlinksFn      = filepath.EvalSymlinks
)

const maxLogSuffixLen = 64

var logSuffixCounter atomic.Uint64

// NewLogger creates the async logger and starts the worker goroutine.
// The log file is created under os.TempDir() using the required naming scheme.
func NewLogger() (*Logger, error) {
	return NewLoggerWithSuffix("")
}

// NewLoggerWithSuffix creates a logger with an optional suffix in the filename.
// Useful for tests that need isolated log files within the same process.
func NewLoggerWithSuffix(suffix string) (*Logger, error) {
	pid := os.Getpid()
	filename := fmt.Sprintf("%s-%d", primaryLogPrefix(), pid)
	var safeSuffix string
	if suffix != "" {
		safeSuffix = sanitizeLogSuffix(suffix)
	}
	if safeSuffix != "" {
		filename += "-" + safeSuffix
	}
	filename += ".log"

	path := filepath.Clean(filepath.Join(os.TempDir(), filename))

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, err
	}

	l := &Logger{
		path:     path,
		file:     f,
		writer:   bufio.NewWriterSize(f, 4096),
		ch:       make(chan logEntry, 1000),
		flushReq: make(chan chan struct{}, 1),
		done:     make(chan struct{}),
	}

	l.workerWG.Add(1)
	go l.run()

	return l, nil
}

func sanitizeLogSuffix(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallbackLogSuffix()
	}

	var b strings.Builder
	changed := false
	for _, r := range trimmed {
		if isSafeLogRune(r) {
			b.WriteRune(r)
		} else {
			changed = true
			b.WriteByte('-')
		}
		if b.Len() >= maxLogSuffixLen {
			changed = true
			break
		}
	}

	sanitized := strings.Trim(b.String(), "-.")
	if sanitized != b.String() {
		changed = true // Mark if trim removed any characters
	}
	if sanitized == "" {
		return fallbackLogSuffix()
	}

	if changed || len(sanitized) > maxLogSuffixLen {
		hash := crc32.ChecksumIEEE([]byte(trimmed))
		hashStr := fmt.Sprintf("%x", hash)

		maxPrefix := maxLogSuffixLen - len(hashStr) - 1
		if maxPrefix < 1 {
			maxPrefix = 1
		}
		if len(sanitized) > maxPrefix {
			sanitized = sanitized[:maxPrefix]
		}

		sanitized = fmt.Sprintf("%s-%s", sanitized, hashStr)
	}

	return sanitized
}

func fallbackLogSuffix() string {
	next := logSuffixCounter.Add(1)
	return fmt.Sprintf("task-%d", next)
}

func isSafeLogRune(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z':
		return true
	case r >= 'A' && r <= 'Z':
		return true
	case r >= '0' && r <= '9':
		return true
	case r == '-', r == '_', r == '.':
		return true
	default:
		return false
	}
}

// Path returns the underlying log file path (useful for tests/inspection).
func (l *Logger) Path() string {
	if l == nil {
		return ""
	}
	return l.path
}

// Info logs at INFO level.
func (l *Logger) Info(msg string) { l.log("INFO", msg) }

// Warn logs at WARN level.
func (l *Logger) Warn(msg string) { l.log("WARN", msg) }

// Debug logs at DEBUG level.
func (l *Logger) Debug(msg string) { l.log("DEBUG", msg) }

// Error logs at ERROR level.
func (l *Logger) Error(msg string) { l.log("ERROR", msg) }

// Close signals the worker to flush and close the log file.
// The log file is NOT removed, allowing inspection after program exit.
// It is safe to call multiple times.
// Waits up to CODEAGENT_LOGGER_CLOSE_TIMEOUT_MS (default: 5000) for shutdown; set to 0 to wait indefinitely.
// Returns an error if shutdown doesn't complete within the timeout.
func (l *Logger) Close() error {
	if l == nil {
		return nil
	}

	var closeErr error

	l.closeOnce.Do(func() {
		l.closed.Store(true)
		close(l.done)

		timeout := loggerCloseTimeout()
		workerDone := make(chan struct{})
		go func() {
			l.workerWG.Wait()
			close(workerDone)
		}()

		if timeout > 0 {
			select {
			case <-workerDone:
				// Worker stopped gracefully
			case <-time.After(timeout):
				closeErr = fmt.Errorf("logger worker timeout during close")
				return
			}
		} else {
			<-workerDone
		}

		if l.workerErr != nil && closeErr == nil {
			closeErr = l.workerErr
		}
	})

	return closeErr
}

func loggerCloseTimeout() time.Duration {
	const defaultTimeout = 5 * time.Second

	raw := strings.TrimSpace(os.Getenv("CODEAGENT_LOGGER_CLOSE_TIMEOUT_MS"))
	if raw == "" {
		return defaultTimeout
	}
	ms, err := strconv.Atoi(raw)
	if err != nil {
		return defaultTimeout
	}
	if ms <= 0 {
		return 0
	}
	return time.Duration(ms) * time.Millisecond
}

// RemoveLogFile removes the log file. Should only be called after Close().
func (l *Logger) RemoveLogFile() error {
	if l == nil {
		return nil
	}
	return os.Remove(l.path)
}

// ExtractRecentErrors returns the most recent ERROR and WARN entries from memory cache.
// Returns up to maxEntries entries in chronological order.
func (l *Logger) ExtractRecentErrors(maxEntries int) []string {
	if l == nil || maxEntries <= 0 {
		return nil
	}

	l.errorMu.Lock()
	defer l.errorMu.Unlock()

	if len(l.errorEntries) == 0 {
		return nil
	}

	// Return last N entries
	start := 0
	if len(l.errorEntries) > maxEntries {
		start = len(l.errorEntries) - maxEntries
	}

	result := make([]string, len(l.errorEntries)-start)
	copy(result, l.errorEntries[start:])
	return result
}

// Flush waits for all pending log entries to be written. Primarily for tests.
// Returns after a 5-second timeout to prevent indefinite blocking.
func (l *Logger) Flush() {
	if l == nil {
		return
	}

	l.flushMu.Lock()
	defer l.flushMu.Unlock()

	// Wait for pending entries with timeout
	done := make(chan struct{})
	go func() {
		l.pendingWG.Wait()
		close(done)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	select {
	case <-done:
		// All pending entries processed
	case <-ctx.Done():
		// Timeout - return without full flush
		return
	}

	// Trigger writer flush
	flushDone := make(chan struct{})
	select {
	case l.flushReq <- flushDone:
		// Wait for flush to complete
		select {
		case <-flushDone:
			// Flush completed
		case <-time.After(1 * time.Second):
			// Flush timeout
		}
	case <-l.done:
		// Logger is closing
	case <-time.After(1 * time.Second):
		// Timeout sending flush request
	}
}

func (l *Logger) log(level, msg string) {
	if l == nil {
		return
	}
	if l.closed.Load() {
		return
	}

	isError := level == "WARN" || level == "ERROR"
	entry := logEntry{msg: msg, isError: isError}
	l.flushMu.Lock()
	l.pendingWG.Add(1)
	l.flushMu.Unlock()

	select {
	case l.ch <- entry:
		// Successfully sent to channel
	case <-l.done:
		// Logger is closing, drop this entry
		l.pendingWG.Done()
		return
	}
}

func (l *Logger) run() {
	defer l.workerWG.Done()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	writeEntry := func(entry logEntry) {
		timestamp := time.Now().Format("2006-01-02 15:04:05.000")
		fmt.Fprintf(l.writer, "[%s] %s\n", timestamp, entry.msg)

		// Cache error/warn entries in memory for fast extraction
		if entry.isError {
			l.errorMu.Lock()
			l.errorEntries = append(l.errorEntries, entry.msg)
			if len(l.errorEntries) > 100 { // Keep last 100
				l.errorEntries = l.errorEntries[1:]
			}
			l.errorMu.Unlock()
		}

		l.pendingWG.Done()
	}

	finalize := func() {
		if err := l.writer.Flush(); err != nil && l.workerErr == nil {
			l.workerErr = err
		}
		if err := l.file.Sync(); err != nil && l.workerErr == nil {
			l.workerErr = err
		}
		if err := l.file.Close(); err != nil && l.workerErr == nil {
			l.workerErr = err
		}
	}

	for {
		select {
		case entry, ok := <-l.ch:
			if !ok {
				finalize()
				return
			}
			writeEntry(entry)

		case <-ticker.C:
			_ = l.writer.Flush()

		case flushDone := <-l.flushReq:
			// Explicit flush request - flush writer and sync to disk
			_ = l.writer.Flush()
			_ = l.file.Sync()
			close(flushDone)

		case <-l.done:
			for {
				select {
				case entry, ok := <-l.ch:
					if !ok {
						finalize()
						return
					}
					writeEntry(entry)
				default:
					finalize()
					return
				}
			}
		}
	}
}

// cleanupOldLogs scans os.TempDir() for wrapper log files and removes those
// whose owning process is no longer running (i.e., orphaned logs).
// It includes safety checks for:
// - PID reuse: Compares file modification time with process start time
// - Symlink attacks: Ensures files are within TempDir and not symlinks
func cleanupOldLogs() (CleanupStats, error) {
	var stats CleanupStats
	tempDir := os.TempDir()

	prefixes := logPrefixes()
	if len(prefixes) == 0 {
		prefixes = []string{defaultWrapperName}
	}

	seen := make(map[string]struct{})
	var matches []string
	for _, prefix := range prefixes {
		pattern := filepath.Join(tempDir, fmt.Sprintf("%s-*.log", prefix))
		found, err := globLogFiles(pattern)
		if err != nil {
			logWarn(fmt.Sprintf("cleanupOldLogs: failed to list logs: %v", err))
			return stats, fmt.Errorf("cleanupOldLogs: %w", err)
		}
		for _, path := range found {
			if _, ok := seen[path]; ok {
				continue
			}
			seen[path] = struct{}{}
			matches = append(matches, path)
		}
	}

	var removeErr error

	for _, path := range matches {
		stats.Scanned++
		filename := filepath.Base(path)

		// Security check: Verify file is not a symlink and is within tempDir
		if shouldSkipFile, reason := isUnsafeFile(path, tempDir); shouldSkipFile {
			stats.Kept++
			stats.KeptFiles = append(stats.KeptFiles, filename)
			if reason != "" {
				logWarn(fmt.Sprintf("cleanupOldLogs: skipping %s: %s", filename, reason))
			}
			continue
		}

		pid, ok := parsePIDFromLog(path)
		if !ok {
			stats.Kept++
			stats.KeptFiles = append(stats.KeptFiles, filename)
			continue
		}

		// Check if process is running
		if !processRunningCheck(pid) {
			// Process not running, safe to delete
			if err := removeLogFileFn(path); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					// File already deleted by another process, don't count as success
					stats.Kept++
					stats.KeptFiles = append(stats.KeptFiles, filename+" (already deleted)")
					continue
				}
				stats.Errors++
				logWarn(fmt.Sprintf("cleanupOldLogs: failed to remove %s: %v", filename, err))
				removeErr = errors.Join(removeErr, fmt.Errorf("failed to remove %s: %w", filename, err))
				continue
			}
			stats.Deleted++
			stats.DeletedFiles = append(stats.DeletedFiles, filename)
			continue
		}

		// Process is running, check for PID reuse
		if isPIDReused(path, pid) {
			// PID was reused, the log file is orphaned
			if err := removeLogFileFn(path); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					stats.Kept++
					stats.KeptFiles = append(stats.KeptFiles, filename+" (already deleted)")
					continue
				}
				stats.Errors++
				logWarn(fmt.Sprintf("cleanupOldLogs: failed to remove %s (PID reused): %v", filename, err))
				removeErr = errors.Join(removeErr, fmt.Errorf("failed to remove %s: %w", filename, err))
				continue
			}
			stats.Deleted++
			stats.DeletedFiles = append(stats.DeletedFiles, filename)
			continue
		}

		// Process is running and owns this log file
		stats.Kept++
		stats.KeptFiles = append(stats.KeptFiles, filename)
	}

	if removeErr != nil {
		return stats, fmt.Errorf("cleanupOldLogs: %w", removeErr)
	}

	return stats, nil
}

// isUnsafeFile checks if a file is unsafe to delete (symlink or outside tempDir).
// Returns (true, reason) if the file should be skipped.
func isUnsafeFile(path string, tempDir string) (bool, string) {
	// Check if file is a symlink
	info, err := fileStatFn(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return true, "" // File disappeared, skip silently
		}
		return true, fmt.Sprintf("stat failed: %v", err)
	}

	// Check if it's a symlink
	if info.Mode()&os.ModeSymlink != 0 {
		return true, "refusing to delete symlink"
	}

	// Resolve any path traversal and verify it's within tempDir
	resolvedPath, err := evalSymlinksFn(path)
	if err != nil {
		return true, fmt.Sprintf("path resolution failed: %v", err)
	}

	// Get absolute path of tempDir
	absTempDir, err := filepath.Abs(tempDir)
	if err != nil {
		return true, fmt.Sprintf("tempDir resolution failed: %v", err)
	}

	// Ensure resolved path is within tempDir
	relPath, err := filepath.Rel(absTempDir, resolvedPath)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return true, "file is outside tempDir"
	}

	return false, ""
}

// isPIDReused checks if a PID has been reused by comparing file modification time
// with process start time. Returns true if the log file was created by a different
// process that previously had the same PID.
func isPIDReused(logPath string, pid int) bool {
	// Get file modification time (when log was last written)
	info, err := fileStatFn(logPath)
	if err != nil {
		// If we can't stat the file, be conservative and keep it
		return false
	}
	fileModTime := info.ModTime()

	// Get process start time
	procStartTime := processStartTimeFn(pid)
	if procStartTime.IsZero() {
		// Can't determine process start time
		// Check if file is very old (>7 days), likely from a dead process
		if time.Since(fileModTime) > 7*24*time.Hour {
			return true // File is old enough to be from a different process
		}
		return false // Be conservative for recent files
	}

	// If the log file was modified before the process started, PID was reused
	// Add a small buffer (1 second) to account for clock skew and file system timing
	return fileModTime.Add(1 * time.Second).Before(procStartTime)
}

func parsePIDFromLog(path string) (int, bool) {
	name := filepath.Base(path)
	prefixes := logPrefixes()
	if len(prefixes) == 0 {
		prefixes = []string{defaultWrapperName}
	}

	for _, prefix := range prefixes {
		prefixWithDash := fmt.Sprintf("%s-", prefix)
		if !strings.HasPrefix(name, prefixWithDash) || !strings.HasSuffix(name, ".log") {
			continue
		}

		core := strings.TrimSuffix(strings.TrimPrefix(name, prefixWithDash), ".log")
		if core == "" {
			continue
		}

		pidPart := core
		if idx := strings.IndexRune(core, '-'); idx != -1 {
			pidPart = core[:idx]
		}

		if pidPart == "" {
			continue
		}

		pid, err := strconv.Atoi(pidPart)
		if err != nil || pid <= 0 {
			continue
		}
		return pid, true
	}

	return 0, false
}

func logConcurrencyPlanning(limit, total int) {
	logger := activeLogger()
	if logger == nil {
		return
	}
	logger.Info(fmt.Sprintf("parallel: worker_limit=%s total_tasks=%d", renderWorkerLimit(limit), total))
}

func logConcurrencyState(event, taskID string, active, limit int) {
	logger := activeLogger()
	if logger == nil {
		return
	}
	logger.Debug(fmt.Sprintf("parallel: %s task=%s active=%d limit=%s", event, taskID, active, renderWorkerLimit(limit)))
}

func renderWorkerLimit(limit int) string {
	if limit <= 0 {
		return "unbounded"
	}
	return strconv.Itoa(limit)
}
