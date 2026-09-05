package cleanup

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	logger "codeagent-wrapper/internal/logger"
)

func TestExecuteRequiresRunner(t *testing.T) {
	_, err := Execute(Deps{})
	if err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("Execute() error = %v, want not configured", err)
	}
}

func TestRender(t *testing.T) {
	var out bytes.Buffer

	Render(&out, logger.CleanupStats{
		Scanned:      2,
		Deleted:      1,
		Kept:         1,
		Errors:       1,
		DeletedFiles: []string{"old.log"},
		KeptFiles:    []string{"keep.log"},
	})

	got := out.String()
	wantParts := []string{
		"Cleanup completed\n",
		"Files scanned: 2\n",
		"Files deleted: 1\n",
		"  - old.log\n",
		"Files kept: 1\n",
		"  - keep.log\n",
		"Deletion errors: 1\n",
	}
	for _, want := range wantParts {
		if !strings.Contains(got, want) {
			t.Fatalf("Render() missing %q in %q", want, got)
		}
	}
}

func TestExecuteDelegates(t *testing.T) {
	want := errors.New("boom")
	_, err := Execute(Deps{
		Run: func() (logger.CleanupStats, error) { return logger.CleanupStats{}, want },
	})
	if !errors.Is(err, want) {
		t.Fatalf("Execute() error = %v, want %v", err, want)
	}
}
