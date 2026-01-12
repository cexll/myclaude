package main

import (
	"os"
	"strings"
	"testing"
)

func TestLogWriterWriteLimitsBuffer(t *testing.T) {
	defer resetTestHooks()

	logger, err := NewLogger()
	if err != nil {
		t.Fatalf("NewLogger error: %v", err)
	}
	setLogger(logger)
	defer closeLogger()

	lw := newLogWriter("P:", 10)
	_, _ = lw.Write([]byte(strings.Repeat("a", 100)))

	if lw.buf.Len() != 10 {
		t.Fatalf("logWriter buffer len=%d, want %d", lw.buf.Len(), 10)
	}
	if !lw.dropped {
		t.Fatalf("expected logWriter to drop overlong line bytes")
	}

	lw.Flush()
	logger.Flush()
	data, err := os.ReadFile(logger.Path())
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if !strings.Contains(string(data), "P:aaaaaaa...") {
		t.Fatalf("log output missing truncated entry, got %q", string(data))
	}
}
