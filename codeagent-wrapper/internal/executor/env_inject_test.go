package executor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	backend "codeagent-wrapper/internal/backend"
	config "codeagent-wrapper/internal/config"
)

// TestEnvInjectionWithAgent tests the full flow of env injection with agent config
func TestEnvInjectionWithAgent(t *testing.T) {
	// Setup temp config
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".codeagent")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write test config with agent that has base_url and api_key
	configContent := `{
		"default_backend": "codex",
		"agents": {
			"test-agent": {
				"backend": "claude",
				"model": "test-model",
				"base_url": "https://test.api.com",
				"api_key": "test-api-key-12345678"
			}
		}
	}`
	configPath := filepath.Join(configDir, "models.json")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	// Reset config cache
	config.ResetModelsConfigCacheForTest()
	defer config.ResetModelsConfigCacheForTest()

	// Test ResolveAgentConfig
	agentBackend, model, _, _, baseURL, apiKey, _, _, _, err := config.ResolveAgentConfig("test-agent")
	if err != nil {
		t.Fatalf("ResolveAgentConfig: %v", err)
	}
	t.Logf("ResolveAgentConfig: backend=%q, model=%q, baseURL=%q, apiKey=%q",
		agentBackend, model, baseURL, apiKey)

	if agentBackend != "claude" {
		t.Errorf("expected backend 'claude', got %q", agentBackend)
	}
	if baseURL != "https://test.api.com" {
		t.Errorf("expected baseURL 'https://test.api.com', got %q", baseURL)
	}
	if apiKey != "test-api-key-12345678" {
		t.Errorf("expected apiKey 'test-api-key-12345678', got %q", apiKey)
	}

	// Test Backend.Env
	b := backend.ClaudeBackend{}
	env := b.Env(baseURL, apiKey)
	t.Logf("Backend.Env: %v", env)

	if env == nil {
		t.Fatal("expected non-nil env from Backend.Env")
	}
	if env["ANTHROPIC_BASE_URL"] != baseURL {
		t.Errorf("expected ANTHROPIC_BASE_URL=%q, got %q", baseURL, env["ANTHROPIC_BASE_URL"])
	}
	if env["ANTHROPIC_API_KEY"] != apiKey {
		t.Errorf("expected ANTHROPIC_API_KEY=%q, got %q", apiKey, env["ANTHROPIC_API_KEY"])
	}
}

// TestEnvInjectionLogic tests the exact logic used in executor
func TestEnvInjectionLogic(t *testing.T) {
	// Setup temp config
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".codeagent")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	configContent := `{
		"default_backend": "codex",
		"agents": {
			"explore": {
				"backend": "claude",
				"model": "MiniMax-M2.1",
				"base_url": "https://api.minimaxi.com/anthropic",
				"api_key": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.test"
			}
		}
	}`
	configPath := filepath.Join(configDir, "models.json")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	config.ResetModelsConfigCacheForTest()
	defer config.ResetModelsConfigCacheForTest()

	// Simulate the executor logic
	cfgBackend := "claude" // This should come from taskSpec.Backend
	agentName := "explore"

	// Step 1: Get backend config (usually empty for claude without global config)
	baseURL, apiKey := config.ResolveBackendConfig(cfgBackend)
	t.Logf("Step 1 - ResolveBackendConfig(%q): baseURL=%q, apiKey=%q", cfgBackend, baseURL, apiKey)

	// Step 2: If agent specified, get agent config
	if agentName != "" {
		agentBackend, _, _, _, agentBaseURL, agentAPIKey, _, _, _, err := config.ResolveAgentConfig(agentName)
		if err != nil {
			t.Fatalf("ResolveAgentConfig(%q): %v", agentName, err)
		}
		t.Logf("Step 2 - ResolveAgentConfig(%q): backend=%q, baseURL=%q, apiKey=%q",
			agentName, agentBackend, agentBaseURL, agentAPIKey)

		// Step 3: Check if agent backend matches cfg backend
		if strings.EqualFold(strings.TrimSpace(agentBackend), strings.TrimSpace(cfgBackend)) {
			baseURL, apiKey = agentBaseURL, agentAPIKey
			t.Logf("Step 3 - Backend match! Using agent config: baseURL=%q, apiKey=%q", baseURL, apiKey)
		} else {
			t.Logf("Step 3 - Backend mismatch: agent=%q, cfg=%q", agentBackend, cfgBackend)
		}
	}

	// Step 4: Get env vars from backend
	b := backend.ClaudeBackend{}
	injected := b.Env(baseURL, apiKey)
	t.Logf("Step 4 - Backend.Env: %v", injected)

	// Verify
	if len(injected) == 0 {
		t.Fatal("Expected env vars to be injected, got none")
	}

	expectedURL := "https://api.minimaxi.com/anthropic"
	if injected["ANTHROPIC_BASE_URL"] != expectedURL {
		t.Errorf("ANTHROPIC_BASE_URL: expected %q, got %q", expectedURL, injected["ANTHROPIC_BASE_URL"])
	}

	if _, ok := injected["ANTHROPIC_API_KEY"]; !ok {
		t.Error("ANTHROPIC_API_KEY not set")
	}

	// Step 5: Test masking
	for k, v := range injected {
		masked := maskSensitiveValue(k, v)
		t.Logf("Step 5 - Env log: %s=%s", k, masked)
	}
}

// TestTaskSpecBackendPropagation tests that taskSpec.Backend is properly used
func TestTaskSpecBackendPropagation(t *testing.T) {
	// Simulate what happens in RunCodexTaskWithContext
	taskSpec := TaskSpec{
		ID:      "test",
		Task:    "hello",
		Backend: "claude",
		Agent:   "explore",
	}

	// This is the logic from executor.go lines 889-916
	cfg := &config.Config{
		Mode:    "new",
		Task:    taskSpec.Task,
		Backend: "codex", // default
	}

	var backend Backend = nil // nil in single mode
	commandName := "codex"    // default

	if backend != nil {
		cfg.Backend = backend.Name()
	} else if taskSpec.Backend != "" {
		cfg.Backend = taskSpec.Backend
	} else if commandName != "" {
		cfg.Backend = commandName
	}

	t.Logf("taskSpec.Backend=%q, cfg.Backend=%q", taskSpec.Backend, cfg.Backend)

	if cfg.Backend != "claude" {
		t.Errorf("expected cfg.Backend='claude', got %q", cfg.Backend)
	}
}
