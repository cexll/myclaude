package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestResolveAgentConfig_Defaults(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	// Test that default agents resolve correctly without config file
	tests := []struct {
		agent          string
		wantBackend    string
		wantModel      string
		wantPromptFile string
	}{
			{"sisyphus", "claude", "claude-sonnet-4-20250514", "~/.claude/skills/omo/references/sisyphus.md"},
			{"oracle", "claude", "claude-sonnet-4-20250514", "~/.claude/skills/omo/references/oracle.md"},
			{"librarian", "claude", "claude-sonnet-4-5-20250514", "~/.claude/skills/omo/references/librarian.md"},
			{"explore", "opencode", "opencode/grok-code", "~/.claude/skills/omo/references/explore.md"},
			{"frontend-ui-ux-engineer", "gemini", "", "~/.claude/skills/omo/references/frontend-ui-ux-engineer.md"},
			{"document-writer", "gemini", "", "~/.claude/skills/omo/references/document-writer.md"},
		}

	for _, tt := range tests {
		t.Run(tt.agent, func(t *testing.T) {
			backend, model, promptFile, _, _ := resolveAgentConfig(tt.agent)
			if backend != tt.wantBackend {
				t.Errorf("backend = %q, want %q", backend, tt.wantBackend)
			}
			if model != tt.wantModel {
				t.Errorf("model = %q, want %q", model, tt.wantModel)
			}
			if promptFile != tt.wantPromptFile {
				t.Errorf("promptFile = %q, want %q", promptFile, tt.wantPromptFile)
			}
		})
	}
}

func TestResolveAgentConfig_UnknownAgent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	backend, model, promptFile, _, _ := resolveAgentConfig("unknown-agent")
	if backend != "opencode" {
		t.Errorf("unknown agent backend = %q, want %q", backend, "opencode")
	}
	if model != "opencode/grok-code" {
		t.Errorf("unknown agent model = %q, want %q", model, "opencode/grok-code")
	}
	if promptFile != "" {
		t.Errorf("unknown agent promptFile = %q, want empty", promptFile)
	}
}

func TestLoadModelsConfig_NoFile(t *testing.T) {
	home := "/nonexistent/path/that/does/not/exist"
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	cfg := loadModelsConfig()
	if cfg.DefaultBackend != "opencode" {
		t.Errorf("DefaultBackend = %q, want %q", cfg.DefaultBackend, "opencode")
	}
	if len(cfg.Agents) != 7 {
		t.Errorf("len(Agents) = %d, want 7", len(cfg.Agents))
	}
}

func TestLoadModelsConfig_WithFile(t *testing.T) {
	// Create temp dir and config file
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".codeagent")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	configContent := `{
		"default_backend": "claude",
		"default_model": "claude-opus-4",
		"agents": {
			"custom-agent": {
				"backend": "codex",
				"model": "gpt-4o",
				"description": "Custom agent"
			}
		}
	}`
	configPath := filepath.Join(configDir, "models.json")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	cfg := loadModelsConfig()

	if cfg.DefaultBackend != "claude" {
		t.Errorf("DefaultBackend = %q, want %q", cfg.DefaultBackend, "claude")
	}
	if cfg.DefaultModel != "claude-opus-4" {
		t.Errorf("DefaultModel = %q, want %q", cfg.DefaultModel, "claude-opus-4")
	}

	// Check custom agent
	if agent, ok := cfg.Agents["custom-agent"]; !ok {
		t.Error("custom-agent not found")
	} else {
		if agent.Backend != "codex" {
			t.Errorf("custom-agent.Backend = %q, want %q", agent.Backend, "codex")
		}
		if agent.Model != "gpt-4o" {
			t.Errorf("custom-agent.Model = %q, want %q", agent.Model, "gpt-4o")
		}
	}

	// Check that defaults are merged
	if _, ok := cfg.Agents["sisyphus"]; !ok {
		t.Error("default agent sisyphus should be merged")
	}
}

func TestLoadModelsConfig_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".codeagent")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write invalid JSON
	configPath := filepath.Join(configDir, "models.json")
	if err := os.WriteFile(configPath, []byte("invalid json {"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	cfg := loadModelsConfig()
	// Should fall back to defaults
	if cfg.DefaultBackend != "opencode" {
		t.Errorf("invalid JSON should fallback, got DefaultBackend = %q", cfg.DefaultBackend)
	}
}

func TestOpencodeBackend_BuildArgs(t *testing.T) {
	backend := OpencodeBackend{}

	t.Run("basic", func(t *testing.T) {
		cfg := &Config{Mode: "new"}
		got := backend.BuildArgs(cfg, "hello")
		want := []string{"run", "--format", "json", "hello"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("with model", func(t *testing.T) {
		cfg := &Config{Mode: "new", Model: "opencode/grok-code"}
		got := backend.BuildArgs(cfg, "task")
		want := []string{"run", "-m", "opencode/grok-code", "--format", "json", "task"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("resume mode", func(t *testing.T) {
		cfg := &Config{Mode: "resume", SessionID: "ses_123", Model: "opencode/grok-code"}
		got := backend.BuildArgs(cfg, "follow-up")
		want := []string{"run", "-m", "opencode/grok-code", "-s", "ses_123", "--format", "json", "follow-up"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("resume without session", func(t *testing.T) {
		cfg := &Config{Mode: "resume"}
		got := backend.BuildArgs(cfg, "task")
		want := []string{"run", "--format", "json", "task"}
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

func TestBackendRegistry_IncludesOpencode(t *testing.T) {
	if _, ok := backendRegistry["opencode"]; !ok {
		t.Error("backendRegistry should include opencode")
	}
}
