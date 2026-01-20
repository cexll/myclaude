package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveAgentConfig_Defaults(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Cleanup(ResetModelsConfigCacheForTest)
	ResetModelsConfigCacheForTest()

	// Test that default agents resolve correctly without config file
	tests := []struct {
		agent          string
		wantBackend    string
		wantModel      string
		wantPromptFile string
	}{
		{"oracle", "claude", "claude-opus-4-5-20251101", "~/.claude/skills/omo/references/oracle.md"},
		{"librarian", "claude", "claude-sonnet-4-5-20250929", "~/.claude/skills/omo/references/librarian.md"},
		{"explore", "opencode", "opencode/grok-code", "~/.claude/skills/omo/references/explore.md"},
		{"frontend-ui-ux-engineer", "gemini", "", "~/.claude/skills/omo/references/frontend-ui-ux-engineer.md"},
		{"document-writer", "gemini", "", "~/.claude/skills/omo/references/document-writer.md"},
	}

	for _, tt := range tests {
		t.Run(tt.agent, func(t *testing.T) {
			backend, model, promptFile, _, _, _, _ := resolveAgentConfig(tt.agent)
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
	t.Cleanup(ResetModelsConfigCacheForTest)
	ResetModelsConfigCacheForTest()

	backend, model, promptFile, _, _, _, _ := resolveAgentConfig("unknown-agent")
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
	t.Cleanup(ResetModelsConfigCacheForTest)
	ResetModelsConfigCacheForTest()

	cfg := loadModelsConfig()
	if cfg.DefaultBackend != "opencode" {
		t.Errorf("DefaultBackend = %q, want %q", cfg.DefaultBackend, "opencode")
	}
	if len(cfg.Agents) != 6 {
		t.Errorf("len(Agents) = %d, want 6", len(cfg.Agents))
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
		"backends": {
			"Claude": {
				"base_url": "https://backend.example",
				"api_key": "backend-key"
			},
			"codex": {
				"base_url": "https://openai.example",
				"api_key": "openai-key"
			}
		},
		"agents": {
			"custom-agent": {
				"backend": "codex",
				"model": "gpt-4o",
				"description": "Custom agent",
				"base_url": "https://agent.example",
				"api_key": "agent-key"
			}
		}
	}`
	configPath := filepath.Join(configDir, "models.json")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)
	t.Cleanup(ResetModelsConfigCacheForTest)
	ResetModelsConfigCacheForTest()

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
	if _, ok := cfg.Agents["oracle"]; !ok {
		t.Error("default agent oracle should be merged")
	}

	baseURL, apiKey := ResolveBackendConfig("claude")
	if baseURL != "https://backend.example" {
		t.Errorf("ResolveBackendConfig(baseURL) = %q, want %q", baseURL, "https://backend.example")
	}
	if apiKey != "backend-key" {
		t.Errorf("ResolveBackendConfig(apiKey) = %q, want %q", apiKey, "backend-key")
	}

	backend, model, _, _, agentBaseURL, agentAPIKey, _ := ResolveAgentConfig("custom-agent")
	if backend != "codex" {
		t.Errorf("ResolveAgentConfig(backend) = %q, want %q", backend, "codex")
	}
	if model != "gpt-4o" {
		t.Errorf("ResolveAgentConfig(model) = %q, want %q", model, "gpt-4o")
	}
	if agentBaseURL != "https://agent.example" {
		t.Errorf("ResolveAgentConfig(baseURL) = %q, want %q", agentBaseURL, "https://agent.example")
	}
	if agentAPIKey != "agent-key" {
		t.Errorf("ResolveAgentConfig(apiKey) = %q, want %q", agentAPIKey, "agent-key")
	}
}

func TestResolveAgentConfig_DynamicAgent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Cleanup(ResetModelsConfigCacheForTest)
	ResetModelsConfigCacheForTest()

	agentDir := filepath.Join(home, ".codeagent", "agents")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "sarsh.md"), []byte("prompt\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	backend, model, promptFile, _, _, _, _ := resolveAgentConfig("sarsh")
	if backend != "opencode" {
		t.Errorf("backend = %q, want %q", backend, "opencode")
	}
	if model != "opencode/grok-code" {
		t.Errorf("model = %q, want %q", model, "opencode/grok-code")
	}
	if promptFile != "~/.codeagent/agents/sarsh.md" {
		t.Errorf("promptFile = %q, want %q", promptFile, "~/.codeagent/agents/sarsh.md")
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
	t.Cleanup(ResetModelsConfigCacheForTest)
	ResetModelsConfigCacheForTest()

	cfg := loadModelsConfig()
	// Should fall back to defaults
	if cfg.DefaultBackend != "opencode" {
		t.Errorf("invalid JSON should fallback, got DefaultBackend = %q", cfg.DefaultBackend)
	}
}
