package wrapper

import (
	"os"
	"testing"
)

func TestDefaultIsTerminalCoverage(t *testing.T) {
	oldStdin := os.Stdin
	t.Cleanup(func() { os.Stdin = oldStdin })

	f, err := os.CreateTemp(t.TempDir(), "stdin-*")
	if err != nil {
		t.Fatalf("os.CreateTemp() error = %v", err)
	}
	defer os.Remove(f.Name())

	os.Stdin = f
	if got := defaultIsTerminal(); got {
		t.Fatalf("defaultIsTerminal() = %v, want false for regular file", got)
	}

	if err := f.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	os.Stdin = f
	if got := defaultIsTerminal(); !got {
		t.Fatalf("defaultIsTerminal() = %v, want true when Stat fails", got)
	}
}
