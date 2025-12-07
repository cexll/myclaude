package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
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
	path      string
	file      *os.File
	writer    *bufio.Writer
	ch        chan logEntry
	flushReq  chan chan struct{}
	done      chan struct{}
	closed    atomic.Bool
	closeOnce sync.Once
	workerWG  sync.WaitGroup
	pendingWG sync.WaitGroup
}

type logEntry struct {
	level string
	msg   string
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
	removeLogFileFn     = os.Remove
	globLogFiles        = filepath.Glob
)

// NewLogger creates the async logger and starts the worker goroutine.
// The log file is created under os.TempDir() using the required naming scheme.
func NewLogger() (*Logger, error) {
	return NewLoggerWithSuffix("")
}

// NewLoggerWithSuffix creates a logger with an optional suffix in the filename.
// Useful for tests that need isolated log files within the same process.
func NewLoggerWithSuffix(suffix string) (*Logger, error) {
	filename := fmt.Sprintf("codex-wrapper-%d", os.Getpid())
	if suffix != "" {
		filename += "-" + suffix
	}
	filename += ".log"

	path := filepath.Join(os.TempDir(), filename)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
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

// Close stops the worker and syncs the log file.
// The log file is NOT removed, allowing inspection after program exit.
// It is safe to call multiple times.
// Returns after a 5-second timeout if worker doesn't stop gracefully.
func (l *Logger) Close() error {
	if l == nil {
		return nil
	}

	var closeErr error

	l.closeOnce.Do(func() {
		l.closed.Store(true)
		close(l.done)
		close(l.ch)

		// Wait for worker with timeout
		workerDone := make(chan struct{})
		go func() {
			l.workerWG.Wait()
			close(workerDone)
		}()

		select {
		case <-workerDone:
			// Worker stopped gracefully
		case <-time.After(5 * time.Second):
			// Worker timeout - proceed with cleanup anyway
			closeErr = fmt.Errorf("logger worker timeout during close")
		}

		if err := l.writer.Flush(); err != nil && closeErr == nil {
			closeErr = err
		}

		if err := l.file.Sync(); err != nil && closeErr == nil {
			closeErr = err
		}

		if err := l.file.Close(); err != nil && closeErr == nil {
			closeErr = err
		}

		// Log file is kept for debugging - NOT removed
		// Users can manually clean up /tmp/codex-wrapper-*.log files
	})

	return closeErr
}

// RemoveLogFile removes the log file. Should only be called after Close().
func (l *Logger) RemoveLogFile() error {
	if l == nil {
		return nil
	}
	return os.Remove(l.path)
}

// Flush waits for all pending log entries to be written. Primarily for tests.
// Returns after a 5-second timeout to prevent indefinite blocking.
func (l *Logger) Flush() {
	if l == nil {
		return
	}

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

	entry := logEntry{level: level, msg: msg}
	l.pendingWG.Add(1)

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

	for {
		select {
		case entry, ok := <-l.ch:
			if !ok {
				// Channel closed, final flush
				l.writer.Flush()
				return
			}
			timestamp := time.Now().Format("2006-01-02 15:04:05.000")
			pid := os.Getpid()
			fmt.Fprintf(l.writer, "[%s] [PID:%d] %s: %s\n", timestamp, pid, entry.level, entry.msg)
			l.pendingWG.Done()

		case <-ticker.C:
			l.writer.Flush()

		case flushDone := <-l.flushReq:
			// Explicit flush request - flush writer and sync to disk
			l.writer.Flush()
			l.file.Sync()
			close(flushDone)
		}
	}
}

// cleanupOldLogs scans os.TempDir() for codex-wrapper-*.log files and removes those
// whose owning process is no longer running (i.e., orphaned logs).
func cleanupOldLogs() (CleanupStats, error) {
	var stats CleanupStats
	tempDir := os.TempDir()
	pattern := filepath.Join(tempDir, "codex-wrapper-*.log")

	matches, err := globLogFiles(pattern)
	if err != nil {
		logWarn(fmt.Sprintf("cleanupOldLogs: failed to list logs: %v", err))
		return stats, fmt.Errorf("cleanupOldLogs: %w", err)
	}

	var removeErr error

	for _, path := range matches {
		stats.Scanned++
		filename := filepath.Base(path)
		pid, ok := parsePIDFromLog(path)
		if !ok {
			stats.Kept++
			stats.KeptFiles = append(stats.KeptFiles, filename)
			continue
		}
		if processRunningCheck(pid) {
			stats.Kept++
			stats.KeptFiles = append(stats.KeptFiles, filename)
			continue
		}
		if err := removeLogFileFn(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				stats.Deleted++
				stats.DeletedFiles = append(stats.DeletedFiles, filename)
				continue
			}
			stats.Errors++
			logWarn(fmt.Sprintf("cleanupOldLogs: failed to remove %s: %v", filename, err))
			removeErr = errors.Join(removeErr, fmt.Errorf("failed to remove %s: %w", filename, err))
			continue
		}
		stats.Deleted++
		stats.DeletedFiles = append(stats.DeletedFiles, filename)
	}

	if removeErr != nil {
		return stats, fmt.Errorf("cleanupOldLogs: %w", removeErr)
	}

	return stats, nil
}

func parsePIDFromLog(path string) (int, bool) {
	name := filepath.Base(path)
	if !strings.HasPrefix(name, "codex-wrapper-") || !strings.HasSuffix(name, ".log") {
		return 0, false
	}

	core := strings.TrimSuffix(strings.TrimPrefix(name, "codex-wrapper-"), ".log")
	if core == "" {
		return 0, false
	}

	pidPart := core
	if idx := strings.IndexRune(core, '-'); idx != -1 {
		pidPart = core[:idx]
	}

	if pidPart == "" {
		return 0, false
	}

	pid, err := strconv.Atoi(pidPart)
	if err != nil || pid <= 0 {
		return 0, false
	}

	return pid, true
}
