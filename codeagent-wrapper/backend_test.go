package main

import (
	"reflect"
	"testing"
)

func TestClaudeBuildArgs_ModesAndPermissions(t *testing.T) {
	backend := ClaudeBackend{}

	t.Run("new mode uses workdir without skip by default", func(t *testing.T) {
		cfg := &Config{Mode: "new", WorkDir: "/repo"}
		got := backend.BuildArgs(cfg, "todo")
		want := []string{"-p", "--output-format", "stream-json", "--verbose", "todo"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("new mode opt-in skip permissions with default workdir", func(t *testing.T) {
		cfg := &Config{Mode: "new", SkipPermissions: true}
		got := backend.BuildArgs(cfg, "-")
		want := []string{"-p", "--dangerously-skip-permissions", "--output-format", "stream-json", "--verbose", "-"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("resume mode uses session id and omits workdir", func(t *testing.T) {
		cfg := &Config{Mode: "resume", SessionID: "sid-123", WorkDir: "/ignored"}
		got := backend.BuildArgs(cfg, "resume-task")
		want := []string{"-p", "-r", "sid-123", "--output-format", "stream-json", "--verbose", "resume-task"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("resume mode without session still returns base flags", func(t *testing.T) {
		cfg := &Config{Mode: "resume", WorkDir: "/ignored"}
		got := backend.BuildArgs(cfg, "follow-up")
		want := []string{"-p", "--output-format", "stream-json", "--verbose", "follow-up"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
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

	t.Run("codex build args passthrough remains intact", func(t *testing.T) {
		backend := CodexBackend{}
		cfg := &Config{Mode: "new", WorkDir: "/tmp"}
		got := backend.BuildArgs(cfg, "task")
		want := []string{"e", "--skip-git-repo-check", "-C", "/tmp", "--json", "task"}
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
