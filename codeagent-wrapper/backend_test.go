package main

import (
	"os"
	"reflect"
	"testing"
)

func TestClaudeBuildArgs_ModesAndPermissions(t *testing.T) {
	backend := ClaudeBackend{}

	t.Run("new mode omits skip-permissions by default", func(t *testing.T) {
		cfg := &Config{Mode: "new", WorkDir: "/repo"}
		got := backend.BuildArgs(cfg, "todo")
		wantPrefix := []string{"-p", "--setting-sources", ""}
		wantSuffix := []string{"--output-format", "stream-json", "--verbose", "todo"}
		assertArgsWithOptionalSettings(t, got, wantPrefix, wantSuffix)
	})

	t.Run("new mode can opt-in skip-permissions", func(t *testing.T) {
		cfg := &Config{Mode: "new", SkipPermissions: true}
		got := backend.BuildArgs(cfg, "-")
		wantPrefix := []string{"-p", "--dangerously-skip-permissions", "--setting-sources", ""}
		wantSuffix := []string{"--output-format", "stream-json", "--verbose", "-"}
		assertArgsWithOptionalSettings(t, got, wantPrefix, wantSuffix)
	})

	t.Run("resume mode includes session id", func(t *testing.T) {
		cfg := &Config{Mode: "resume", SessionID: "sid-123", WorkDir: "/ignored"}
		got := backend.BuildArgs(cfg, "resume-task")
		wantPrefix := []string{"-p", "--setting-sources", ""}
		wantSuffix := []string{"-r", "sid-123", "--output-format", "stream-json", "--verbose", "resume-task"}
		assertArgsWithOptionalSettings(t, got, wantPrefix, wantSuffix)
	})

	t.Run("resume mode without session still returns base flags", func(t *testing.T) {
		cfg := &Config{Mode: "resume", WorkDir: "/ignored"}
		got := backend.BuildArgs(cfg, "follow-up")
		wantPrefix := []string{"-p", "--setting-sources", ""}
		wantSuffix := []string{"--output-format", "stream-json", "--verbose", "follow-up"}
		assertArgsWithOptionalSettings(t, got, wantPrefix, wantSuffix)
	})

	t.Run("resume mode can opt-in skip permissions", func(t *testing.T) {
		cfg := &Config{Mode: "resume", SessionID: "sid-123", SkipPermissions: true}
		got := backend.BuildArgs(cfg, "resume-task")
		wantPrefix := []string{"-p", "--dangerously-skip-permissions", "--setting-sources", ""}
		wantSuffix := []string{"-r", "sid-123", "--output-format", "stream-json", "--verbose", "resume-task"}
		assertArgsWithOptionalSettings(t, got, wantPrefix, wantSuffix)
	})

	t.Run("nil config returns nil", func(t *testing.T) {
		if backend.BuildArgs(nil, "ignored") != nil {
			t.Fatalf("nil config should return nil args")
		}
	})
}

func TestClaudeBuildArgs_GeminiAndCodexModes(t *testing.T) {
	t.Run("gemini new mode defaults workdir", func(t *testing.T) {
		backend := GeminiBackend{}
		cfg := &Config{Mode: "new", WorkDir: "/workspace"}
		got := backend.BuildArgs(cfg, "task")
		want := []string{"-o", "stream-json", "-y", "-p", "task"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("gemini resume mode uses session id", func(t *testing.T) {
		backend := GeminiBackend{}
		cfg := &Config{Mode: "resume", SessionID: "sid-999"}
		got := backend.BuildArgs(cfg, "resume")
		want := []string{"-o", "stream-json", "-y", "-r", "sid-999", "-p", "resume"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("gemini resume mode without session omits identifier", func(t *testing.T) {
		backend := GeminiBackend{}
		cfg := &Config{Mode: "resume"}
		got := backend.BuildArgs(cfg, "resume")
		want := []string{"-o", "stream-json", "-y", "-p", "resume"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("gemini nil config returns nil", func(t *testing.T) {
		backend := GeminiBackend{}
		if backend.BuildArgs(nil, "ignored") != nil {
			t.Fatalf("nil config should return nil args")
		}
	})

	t.Run("codex build args omits bypass flag by default", func(t *testing.T) {
		const key = "CODEX_BYPASS_SANDBOX"
		t.Cleanup(func() { os.Unsetenv(key) })
		os.Unsetenv(key)

		backend := CodexBackend{}
		cfg := &Config{Mode: "new", WorkDir: "/tmp"}
		got := backend.BuildArgs(cfg, "task")
		want := []string{"e", "--skip-git-repo-check", "-C", "/tmp", "--json", "task"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("codex build args includes bypass flag when enabled", func(t *testing.T) {
		const key = "CODEX_BYPASS_SANDBOX"
		t.Cleanup(func() { os.Unsetenv(key) })
		os.Setenv(key, "true")

		backend := CodexBackend{}
		cfg := &Config{Mode: "new", WorkDir: "/tmp"}
		got := backend.BuildArgs(cfg, "task")
		want := []string{"e", "--dangerously-bypass-approvals-and-sandbox", "--skip-git-repo-check", "-C", "/tmp", "--json", "task"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})
}

func TestClaudeBuildArgs_BackendMetadata(t *testing.T) {
	tests := []struct {
		backend Backend
		name    string
		command string
	}{
		{backend: CodexBackend{}, name: "codex", command: "codex"},
		{backend: ClaudeBackend{}, name: "claude", command: "claude"},
		{backend: GeminiBackend{}, name: "gemini", command: "gemini"},
	}

	for _, tt := range tests {
		if got := tt.backend.Name(); got != tt.name {
			t.Fatalf("Name() = %s, want %s", got, tt.name)
		}
		if got := tt.backend.Command(); got != tt.command {
			t.Fatalf("Command() = %s, want %s", got, tt.command)
		}
	}
}

func assertArgsWithOptionalSettings(t *testing.T, got, wantPrefix, wantSuffix []string) {
	t.Helper()
	if len(got) < len(wantPrefix)+len(wantSuffix) {
		t.Fatalf("args too short: got %v", got)
	}
	if !hasPrefix(got, wantPrefix) {
		t.Fatalf("args prefix mismatch\ngot:  %v\nwant: %v", got, wantPrefix)
	}
	if !hasSuffix(got, wantSuffix) {
		t.Fatalf("args suffix mismatch\ngot:  %v\nwant: %v", got, wantSuffix)
	}

	settingsIdx := findArg(got, "--settings")
	if settingsIdx != -1 {
		if settingsIdx+1 >= len(got) {
			t.Fatalf("--settings missing value in %v", got)
		}
		if settingsIdx < len(wantPrefix) {
			t.Fatalf("--settings at wrong position %d in %v", settingsIdx, got)
		}
		suffixStart := len(got) - len(wantSuffix)
		if settingsIdx >= suffixStart {
			t.Fatalf("--settings placed inside suffix at %d in %v", settingsIdx, got)
		}
	}
}

func hasPrefix(args, prefix []string) bool {
	if len(args) < len(prefix) {
		return false
	}
	for i := range prefix {
		if args[i] != prefix[i] {
			return false
		}
	}
	return true
}

func hasSuffix(args, suffix []string) bool {
	if len(args) < len(suffix) {
		return false
	}
	offset := len(args) - len(suffix)
	for i := range suffix {
		if args[offset+i] != suffix[i] {
			return false
		}
	}
	return true
}

func findArg(args []string, target string) int {
	for i, a := range args {
		if a == target {
			return i
		}
	}
	return -1
}
