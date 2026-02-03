package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveAgentConfig_NoConfig_ReturnsHelpfulError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Cleanup(ResetModelsConfigCacheForTest)
	ResetModelsConfigCacheForTest()

	_, _, _, _, _, _, _, _, _, err := ResolveAgentConfig("develop")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, modelsConfigTildePath) {
		t.Fatalf("error should mention %s, got: %s", modelsConfigTildePath, msg)
	}
	if !strings.Contains(msg, filepath.Join(home, ".codeagent", "models.json")) {
		t.Fatalf("error should mention resolved config path, got: %s", msg)
	}
	if !strings.Contains(msg, "\"agents\"") {
		t.Fatalf("error should include example config, got: %s", msg)
	}
}

func TestLoadModelsConfig_NoFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Cleanup(ResetModelsConfigCacheForTest)
	ResetModelsConfigCacheForTest()

	_, err := loadModelsConfig()
	if err == nil {
		t.Fatalf("expected error, got nil")
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

	cfg, err := loadModelsConfig()
	if err != nil {
		t.Fatalf("loadModelsConfig: %v", err)
	}

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

	if _, ok := cfg.Agents["oracle"]; ok {
		t.Error("oracle should not be present without explicit config")
	}

	baseURL, apiKey := ResolveBackendConfig("claude")
	if baseURL != "https://backend.example" {
		t.Errorf("ResolveBackendConfig(baseURL) = %q, want %q", baseURL, "https://backend.example")
	}
	if apiKey != "backend-key" {
		t.Errorf("ResolveBackendConfig(apiKey) = %q, want %q", apiKey, "backend-key")
	}

	backend, model, _, _, agentBaseURL, agentAPIKey, _, _, _, err := ResolveAgentConfig("custom-agent")
	if err != nil {
		t.Fatalf("ResolveAgentConfig(custom-agent): %v", err)
	}
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

	configDir := filepath.Join(home, ".codeagent")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "models.json"), []byte(`{
  "default_backend": "codex",
  "default_model": "gpt-test"
}`), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	backend, model, promptFile, _, _, _, _, _, _, err := ResolveAgentConfig("sarsh")
	if err != nil {
		t.Fatalf("ResolveAgentConfig(sarsh): %v", err)
	}
	if backend != "codex" {
		t.Errorf("backend = %q, want %q", backend, "codex")
	}
	if model != "gpt-test" {
		t.Errorf("model = %q, want %q", model, "gpt-test")
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

	_, err := loadModelsConfig()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestResolveAgentConfig_UnknownAgent_ReturnsError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Cleanup(ResetModelsConfigCacheForTest)
	ResetModelsConfigCacheForTest()

	configDir := filepath.Join(home, ".codeagent")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "models.json"), []byte(`{
  "default_backend": "codex",
  "default_model": "gpt-test",
  "agents": {
    "develop": { "backend": "codex", "model": "gpt-test" }
  }
}`), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, _, _, _, _, _, _, _, _, err := ResolveAgentConfig("unknown-agent")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unknown-agent") {
		t.Fatalf("error should mention agent name, got: %s", err.Error())
	}
}

func TestResolveAgentConfig_EmptyModel_ReturnsError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Cleanup(ResetModelsConfigCacheForTest)
	ResetModelsConfigCacheForTest()

	configDir := filepath.Join(home, ".codeagent")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "models.json"), []byte(`{
  "agents": {
    "bad-agent": { "backend": "codex", "model": " " }
  }
}`), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, _, _, _, _, _, _, _, _, err := ResolveAgentConfig("bad-agent")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "empty model") {
		t.Fatalf("error should mention empty model, got: %s", err.Error())
	}
}
