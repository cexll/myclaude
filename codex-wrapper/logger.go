package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

// Logger writes log messages asynchronously to a temp file.
// It is intentionally minimal: a buffered channel + single worker goroutine
// to avoid contention while keeping ordering guarantees.
type Logger struct {
	path      string
	file      *os.File
	ch        chan logEntry
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

// NewLogger creates the async logger and starts the worker goroutine.
// The log file is created under os.TempDir() using the required naming scheme.
func NewLogger() (*Logger, error) {
	path := filepath.Join(os.TempDir(), fmt.Sprintf("codex-wrapper-%d.log", os.Getpid()))

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}

	l := &Logger{
		path: path,
		file: f,
		ch:   make(chan logEntry, 100),
		done: make(chan struct{}),
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

// Close stops the worker, syncs and removes the log file.
// It is safe to call multiple times.
func (l *Logger) Close() error {
	if l == nil {
		return nil
	}

	var closeErr error

	l.closeOnce.Do(func() {
		l.closed.Store(true)
		close(l.done)
		close(l.ch)

		l.workerWG.Wait()

		if err := l.file.Sync(); err != nil {
			closeErr = err
		}

		if err := l.file.Close(); err != nil && closeErr == nil {
			closeErr = err
		}

		if err := os.Remove(l.path); err != nil && !os.IsNotExist(err) && closeErr == nil {
			closeErr = err
		}
	})

	return closeErr
}

// Flush waits for all pending log entries to be written. Primarily for tests.
func (l *Logger) Flush() {
	if l == nil {
		return
	}
	l.pendingWG.Wait()
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
	case <-l.done:
		l.pendingWG.Done()
		return
	case l.ch <- entry:
	}
}

func (l *Logger) run() {
	defer l.workerWG.Done()

	for entry := range l.ch {
		fmt.Fprintf(l.file, "%s: %s\n", entry.level, entry.msg)
		l.pendingWG.Done()
	}
}
