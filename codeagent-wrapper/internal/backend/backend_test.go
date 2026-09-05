package backend

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	config "codeagent-wrapper/internal/config"
)

func TestClaudeBuildArgs_ModesAndPermissions(t *testing.T) {
	backend := ClaudeBackend{}

	t.Run("new mode omits skip-permissions when env disabled", func(t *testing.T) {
		t.Setenv("CODEAGENT_SKIP_PERMISSIONS", "false")
		cfg := &config.Config{Mode: "new", WorkDir: "/repo"}
		got := backend.BuildArgs(cfg, "todo")
		want := []string{"-p", "--setting-sources", "", "--output-format", "stream-json", "--verbose", "todo"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("new mode includes skip-permissions by default", func(t *testing.T) {
		cfg := &config.Config{Mode: "new", SkipPermissions: false}
		got := backend.BuildArgs(cfg, "-")
		want := []string{"-p", "--dangerously-skip-permissions", "--setting-sources", "", "--output-format", "stream-json", "--verbose", "-"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("resume mode includes session id", func(t *testing.T) {
		t.Setenv("CODEAGENT_SKIP_PERMISSIONS", "false")
		cfg := &config.Config{Mode: "resume", SessionID: "sid-123", WorkDir: "/ignored"}
		got := backend.BuildArgs(cfg, "resume-task")
		want := []string{"-p", "--setting-sources", "", "-r", "sid-123", "--output-format", "stream-json", "--verbose", "resume-task"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("resume mode without session still returns base flags", func(t *testing.T) {
		t.Setenv("CODEAGENT_SKIP_PERMISSIONS", "false")
		cfg := &config.Config{Mode: "resume", WorkDir: "/ignored"}
		got := backend.BuildArgs(cfg, "follow-up")
		want := []string{"-p", "--setting-sources", "", "--output-format", "stream-json", "--verbose", "follow-up"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("resume mode can opt-in skip permissions", func(t *testing.T) {
		cfg := &config.Config{Mode: "resume", SessionID: "sid-123", SkipPermissions: true}
		got := backend.BuildArgs(cfg, "resume-task")
		want := []string{"-p", "--dangerously-skip-permissions", "--setting-sources", "", "-r", "sid-123", "--output-format", "stream-json", "--verbose", "resume-task"}
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

func TestBackendBuildArgs_Model(t *testing.T) {
	t.Run("claude includes --model when set", func(t *testing.T) {
		t.Setenv("CODEAGENT_SKIP_PERMISSIONS", "false")
		backend := ClaudeBackend{}
		cfg := &config.Config{Mode: "new", Model: "opus"}
		got := backend.BuildArgs(cfg, "todo")
		want := []string{"-p", "--setting-sources", "", "--model", "opus", "--output-format", "stream-json", "--verbose", "todo"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("gemini includes -m when set", func(t *testing.T) {
		backend := GeminiBackend{}
		cfg := &config.Config{Mode: "new", Model: "gemini-3-pro-preview"}
		got := backend.BuildArgs(cfg, "task")
		want := []string{"-o", "stream-json", "-y", "-m", "gemini-3-pro-preview", "task"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("codex includes --model when set", func(t *testing.T) {
		const key = "CODEX_BYPASS_SANDBOX"
		t.Setenv(key, "false")

		backend := CodexBackend{}
		cfg := &config.Config{Mode: "new", WorkDir: "/tmp", Model: "o3"}
		got := backend.BuildArgs(cfg, "task")
		want := []string{"e", "--model", "o3", "--skip-git-repo-check", "-C", "/tmp", "--json", "task"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})
}

func TestClaudeBuildArgs_GeminiAndCodexModes(t *testing.T) {
	t.Run("gemini new mode defaults workdir", func(t *testing.T) {
		backend := GeminiBackend{}
		cfg := &config.Config{Mode: "new", WorkDir: "/workspace"}
		got := backend.BuildArgs(cfg, "task")
		want := []string{"-o", "stream-json", "-y", "task"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("gemini resume mode uses session id", func(t *testing.T) {
		backend := GeminiBackend{}
		cfg := &config.Config{Mode: "resume", SessionID: "sid-999"}
		got := backend.BuildArgs(cfg, "resume")
		want := []string{"-o", "stream-json", "-y", "-r", "sid-999", "resume"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("gemini resume mode without session omits identifier", func(t *testing.T) {
		backend := GeminiBackend{}
		cfg := &config.Config{Mode: "resume"}
		got := backend.BuildArgs(cfg, "resume")
		want := []string{"-o", "stream-json", "-y", "resume"}
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

	t.Run("gemini stdin mode uses -p flag", func(t *testing.T) {
		backend := GeminiBackend{}
		cfg := &config.Config{Mode: "new"}
		got := backend.BuildArgs(cfg, "-")
		want := []string{"-o", "stream-json", "-y", "-p", "-"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("codex build args omits bypass flag by default", func(t *testing.T) {
		const key = "CODEX_BYPASS_SANDBOX"
		t.Setenv(key, "false")

		backend := CodexBackend{}
		cfg := &config.Config{Mode: "new", WorkDir: "/tmp"}
		got := backend.BuildArgs(cfg, "task")
		want := []string{"e", "--skip-git-repo-check", "-C", "/tmp", "--json", "task"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("codex build args includes bypass flag when enabled", func(t *testing.T) {
		const key = "CODEX_BYPASS_SANDBOX"
		t.Setenv(key, "true")

		backend := CodexBackend{}
		cfg := &config.Config{Mode: "new", WorkDir: "/tmp"}
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

func TestLoadMinimalEnvSettings(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	t.Run("missing file returns empty", func(t *testing.T) {
		if got := LoadMinimalEnvSettings(); len(got) != 0 {
			t.Fatalf("got %v, want empty", got)
		}
	})

	t.Run("valid env returns string map", func(t *testing.T) {
		dir := filepath.Join(home, ".claude")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		path := filepath.Join(dir, "settings.json")
		data := []byte(`{"env":{"ANTHROPIC_API_KEY":"secret","FOO":"bar"}}`)
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		got := LoadMinimalEnvSettings()
		if got["ANTHROPIC_API_KEY"] != "secret" || got["FOO"] != "bar" {
			t.Fatalf("got %v, want keys present", got)
		}
	})

	t.Run("non-string values are ignored", func(t *testing.T) {
		dir := filepath.Join(home, ".claude")
		path := filepath.Join(dir, "settings.json")
		data := []byte(`{"env":{"GOOD":"ok","BAD":123,"ALSO_BAD":true}}`)
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}

		got := LoadMinimalEnvSettings()
		if got["GOOD"] != "ok" {
			t.Fatalf("got %v, want GOOD=ok", got)
		}
		if _, ok := got["BAD"]; ok {
			t.Fatalf("got %v, want BAD omitted", got)
		}
		if _, ok := got["ALSO_BAD"]; ok {
			t.Fatalf("got %v, want ALSO_BAD omitted", got)
		}
	})

	t.Run("oversized file returns empty", func(t *testing.T) {
		dir := filepath.Join(home, ".claude")
		path := filepath.Join(dir, "settings.json")
		data := bytes.Repeat([]byte("a"), MaxClaudeSettingsBytes+1)
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		if got := LoadMinimalEnvSettings(); len(got) != 0 {
			t.Fatalf("got %v, want empty", got)
		}
	})
}

func TestOpencodeBackend_BuildArgs(t *testing.T) {
	backend := OpencodeBackend{}

	t.Run("basic", func(t *testing.T) {
		cfg := &config.Config{Mode: "new"}
		got := backend.BuildArgs(cfg, "hello")
		want := []string{"run", "--format", "json", "hello"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("with model", func(t *testing.T) {
		cfg := &config.Config{Mode: "new", Model: "opencode/grok-code"}
		got := backend.BuildArgs(cfg, "task")
		want := []string{"run", "-m", "opencode/grok-code", "--format", "json", "task"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("resume mode", func(t *testing.T) {
		cfg := &config.Config{Mode: "resume", SessionID: "ses_123", Model: "opencode/grok-code"}
		got := backend.BuildArgs(cfg, "follow-up")
		want := []string{"run", "-m", "opencode/grok-code", "-s", "ses_123", "--format", "json", "follow-up"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("resume without session", func(t *testing.T) {
		cfg := &config.Config{Mode: "resume"}
		got := backend.BuildArgs(cfg, "task")
		want := []string{"run", "--format", "json", "task"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("stdin mode omits dash", func(t *testing.T) {
		cfg := &config.Config{Mode: "new"}
		got := backend.BuildArgs(cfg, "-")
		want := []string{"run", "--format", "json"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})
}

func TestOpencodeBackend_Interface(t *testing.T) {
	backend := OpencodeBackend{}

	if backend.Name() != "opencode" {
		t.Errorf("Name() = %q, want %q", backend.Name(), "opencode")
	}
	if backend.Command() != "opencode" {
		t.Errorf("Command() = %q, want %q", backend.Command(), "opencode")
	}
}
