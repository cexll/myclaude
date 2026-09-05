package janitor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParsePIDFromLogAlias(t *testing.T) {
	if pid, ok := ParsePIDFromLog("codeagent-wrapper-123.log"); !ok || pid != 123 {
		t.Fatalf("ParsePIDFromLog() = (%d, %v), want (123, true)", pid, ok)
	}
}

func TestIsUnsafeFileAliasOnMissingPath(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "missing.log")

	unsafe, reason := IsUnsafeFile(path, tempDir)
	if !unsafe {
		t.Fatalf("IsUnsafeFile(%q) = false, want true", path)
	}
	_ = reason
}

func TestCleanupOldLogsAliasSmoke(t *testing.T) {
	tempDir := t.TempDir()
	oldTmp := os.Getenv("TMPDIR")
	t.Setenv("TMPDIR", tempDir)
	if oldTmp == "" {
		t.Setenv("TMP", "")
		t.Setenv("TEMP", "")
	}

	stats, err := CleanupOldLogs()
	if err != nil {
		t.Fatalf("CleanupOldLogs(): %v", err)
	}
	if stats.Scanned != 0 || stats.Deleted != 0 || stats.Kept != 0 || stats.Errors != 0 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}
