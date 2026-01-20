package executor

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWrapTaskWithAgentPrompt(t *testing.T) {
	got := WrapTaskWithAgentPrompt("P", "do")
	want := "<agent-prompt>\nP\n</agent-prompt>\n\ndo"
	if got != want {
		t.Fatalf("wrapTaskWithAgentPrompt mismatch:\n got=%q\nwant=%q", got, want)
	}
}

func TestReadAgentPromptFile_EmptyPath(t *testing.T) {
	for _, allowOutside := range []bool{false, true} {
		got, err := ReadAgentPromptFile("   ", allowOutside)
		if err != nil {
			t.Fatalf("unexpected error (allowOutside=%v): %v", allowOutside, err)
		}
		if got != "" {
			t.Fatalf("expected empty result (allowOutside=%v), got %q", allowOutside, got)
		}
	}
}

func TestReadAgentPromptFile_ExplicitAbsolutePath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prompt.md")
	if err := os.WriteFile(path, []byte("LINE1\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := ReadAgentPromptFile(path, true)
	if err != nil {
		t.Fatalf("readAgentPromptFile error: %v", err)
	}
	if got != "LINE1" {
		t.Fatalf("got %q, want %q", got, "LINE1")
	}
}

func TestReadAgentPromptFile_ExplicitTildeExpansion(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	path := filepath.Join(home, "prompt.md")
	if err := os.WriteFile(path, []byte("P\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := ReadAgentPromptFile("~/prompt.md", true)
	if err != nil {
		t.Fatalf("readAgentPromptFile error: %v", err)
	}
	if got != "P" {
		t.Fatalf("got %q, want %q", got, "P")
	}
}

func TestReadAgentPromptFile_RestrictedAllowsClaudeDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	path := filepath.Join(claudeDir, "prompt.md")
	if err := os.WriteFile(path, []byte("OK\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := ReadAgentPromptFile("~/.claude/prompt.md", false)
	if err != nil {
		t.Fatalf("readAgentPromptFile error: %v", err)
	}
	if got != "OK" {
		t.Fatalf("got %q, want %q", got, "OK")
	}
}

func TestReadAgentPromptFile_RestrictedAllowsCodeagentAgentsDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	agentDir := filepath.Join(home, ".codeagent", "agents")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	path := filepath.Join(agentDir, "sarsh.md")
	if err := os.WriteFile(path, []byte("OK\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := ReadAgentPromptFile("~/.codeagent/agents/sarsh.md", false)
	if err != nil {
		t.Fatalf("readAgentPromptFile error: %v", err)
	}
	if got != "OK" {
		t.Fatalf("got %q, want %q", got, "OK")
	}
}

func TestReadAgentPromptFile_RestrictedRejectsOutsideClaudeDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	path := filepath.Join(home, "prompt.md")
	if err := os.WriteFile(path, []byte("NO\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := ReadAgentPromptFile("~/prompt.md", false); err == nil {
		t.Fatalf("expected error for prompt file outside ~/.claude, got nil")
	}
}

func TestReadAgentPromptFile_RestrictedRejectsTraversal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	path := filepath.Join(home, "secret.md")
	if err := os.WriteFile(path, []byte("SECRET\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := ReadAgentPromptFile("~/.claude/../secret.md", false); err == nil {
		t.Fatalf("expected traversal to be rejected, got nil")
	}
}

func TestReadAgentPromptFile_NotFound(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	_, err := ReadAgentPromptFile("~/.claude/missing.md", false)
	if err == nil || !os.IsNotExist(err) {
		t.Fatalf("expected not-exist error, got %v", err)
	}
}

func TestReadAgentPromptFile_PermissionDenied(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod-based permission test is not reliable on Windows")
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	path := filepath.Join(claudeDir, "private.md")
	if err := os.WriteFile(path, []byte("PRIVATE\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.Chmod(path, 0o000); err != nil {
		t.Fatalf("Chmod: %v", err)
	}

	_, err := ReadAgentPromptFile("~/.claude/private.md", false)
	if err == nil {
		t.Fatalf("expected permission error, got nil")
	}
	if !os.IsPermission(err) && !strings.Contains(strings.ToLower(err.Error()), "permission") {
		t.Fatalf("expected permission denied, got: %v", err)
	}
}
