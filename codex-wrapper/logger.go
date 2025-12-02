package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
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
	flushReq  chan struct{}
	done      chan struct{}
	closed    atomic.Bool
	closeOnce sync.Once
	workerWG  sync.WaitGroup
	pendingWG sync.WaitGroup
	flushMu   sync.Mutex
}

type logEntry struct {
	level string
	msg   string
}

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
		flushReq: make(chan struct{}, 1),
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
	select {
	case l.flushReq <- struct{}{}:
		// Wait for flush to complete (with mutex)
		flushDone := make(chan struct{})
		go func() {
			l.flushMu.Lock()
			l.flushMu.Unlock()
			close(flushDone)
		}()

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
	case <-l.done:
		l.pendingWG.Done()
		return
	default:
		// Channel is full; drop the entry to avoid blocking callers.
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

		case <-l.flushReq:
			// Explicit flush request
			l.flushMu.Lock()
			l.writer.Flush()
			l.flushMu.Unlock()
		}
	}
}
