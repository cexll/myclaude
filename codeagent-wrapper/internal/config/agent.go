package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	ilogger "codeagent-wrapper/internal/logger"

	"github.com/goccy/go-json"
)

type BackendConfig struct {
	BaseURL string `json:"base_url,omitempty"`
	APIKey  string `json:"api_key,omitempty"`
}

type AgentModelConfig struct {
	Backend     string `json:"backend"`
	Model       string `json:"model"`
	PromptFile  string `json:"prompt_file,omitempty"`
	Description string `json:"description,omitempty"`
	Yolo        bool   `json:"yolo,omitempty"`
	Reasoning   string `json:"reasoning,omitempty"`
	BaseURL     string `json:"base_url,omitempty"`
	APIKey      string `json:"api_key,omitempty"`
}

type ModelsConfig struct {
	DefaultBackend string                      `json:"default_backend"`
	DefaultModel   string                      `json:"default_model"`
	Agents         map[string]AgentModelConfig `json:"agents"`
	Backends       map[string]BackendConfig    `json:"backends,omitempty"`
}

var defaultModelsConfig = ModelsConfig{
	DefaultBackend: "opencode",
	DefaultModel:   "opencode/grok-code",
	Agents: map[string]AgentModelConfig{
		"oracle":                  {Backend: "claude", Model: "claude-opus-4-5-20251101", PromptFile: "~/.claude/skills/omo/references/oracle.md", Description: "Technical advisor"},
		"librarian":               {Backend: "claude", Model: "claude-sonnet-4-5-20250929", PromptFile: "~/.claude/skills/omo/references/librarian.md", Description: "Researcher"},
		"explore":                 {Backend: "opencode", Model: "opencode/grok-code", PromptFile: "~/.claude/skills/omo/references/explore.md", Description: "Code search"},
		"develop":                 {Backend: "codex", Model: "", PromptFile: "~/.claude/skills/omo/references/develop.md", Description: "Code development"},
		"frontend-ui-ux-engineer": {Backend: "gemini", Model: "", PromptFile: "~/.claude/skills/omo/references/frontend-ui-ux-engineer.md", Description: "Frontend engineer"},
		"document-writer":         {Backend: "gemini", Model: "", PromptFile: "~/.claude/skills/omo/references/document-writer.md", Description: "Documentation"},
	},
}

var (
	modelsConfigOnce   sync.Once
	modelsConfigCached *ModelsConfig
)

func modelsConfig() *ModelsConfig {
	modelsConfigOnce.Do(func() {
		modelsConfigCached = loadModelsConfig()
	})
	if modelsConfigCached == nil {
		return &defaultModelsConfig
	}
	return modelsConfigCached
}

func loadModelsConfig() *ModelsConfig {
	home, err := os.UserHomeDir()
	if err != nil {
		ilogger.LogWarn(fmt.Sprintf("Failed to resolve home directory for models config: %v; using defaults", err))
		return &defaultModelsConfig
	}

	configDir := filepath.Clean(filepath.Join(home, ".codeagent"))
	configPath := filepath.Clean(filepath.Join(configDir, "models.json"))
	rel, err := filepath.Rel(configDir, configPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return &defaultModelsConfig
	}

	data, err := os.ReadFile(configPath) // #nosec G304 -- path is fixed under user home and validated to stay within configDir
	if err != nil {
		if !os.IsNotExist(err) {
			ilogger.LogWarn(fmt.Sprintf("Failed to read models config %s: %v; using defaults", configPath, err))
		}
		return &defaultModelsConfig
	}

	var cfg ModelsConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		ilogger.LogWarn(fmt.Sprintf("Failed to parse models config %s: %v; using defaults", configPath, err))
		return &defaultModelsConfig
	}

	cfg.DefaultBackend = strings.TrimSpace(cfg.DefaultBackend)
	if cfg.DefaultBackend == "" {
		cfg.DefaultBackend = defaultModelsConfig.DefaultBackend
	}
	cfg.DefaultModel = strings.TrimSpace(cfg.DefaultModel)
	if cfg.DefaultModel == "" {
		cfg.DefaultModel = defaultModelsConfig.DefaultModel
	}

	// Merge with defaults
	for name, agent := range defaultModelsConfig.Agents {
		if _, exists := cfg.Agents[name]; !exists {
			if cfg.Agents == nil {
				cfg.Agents = make(map[string]AgentModelConfig)
			}
			cfg.Agents[name] = agent
		}
	}

	// Normalize backend keys so lookups can be case-insensitive.
	if len(cfg.Backends) > 0 {
		normalized := make(map[string]BackendConfig, len(cfg.Backends))
		for k, v := range cfg.Backends {
			key := strings.ToLower(strings.TrimSpace(k))
			if key == "" {
				continue
			}
			normalized[key] = v
		}
		if len(normalized) > 0 {
			cfg.Backends = normalized
		} else {
			cfg.Backends = nil
		}
	}

	return &cfg
}

func LoadDynamicAgent(name string) (AgentModelConfig, bool) {
	if err := ValidateAgentName(name); err != nil {
		return AgentModelConfig{}, false
	}

	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return AgentModelConfig{}, false
	}

	absPath := filepath.Join(home, ".codeagent", "agents", name+".md")
	info, err := os.Stat(absPath)
	if err != nil || info.IsDir() {
		return AgentModelConfig{}, false
	}

	return AgentModelConfig{PromptFile: "~/.codeagent/agents/" + name + ".md"}, true
}

func ResolveBackendConfig(backendName string) (baseURL, apiKey string) {
	cfg := modelsConfig()
	resolved := resolveBackendConfig(cfg, backendName)
	return strings.TrimSpace(resolved.BaseURL), strings.TrimSpace(resolved.APIKey)
}

func resolveBackendConfig(cfg *ModelsConfig, backendName string) BackendConfig {
	if cfg == nil || len(cfg.Backends) == 0 {
		return BackendConfig{}
	}
	key := strings.ToLower(strings.TrimSpace(backendName))
	if key == "" {
		key = strings.ToLower(strings.TrimSpace(cfg.DefaultBackend))
	}
	if key == "" {
		return BackendConfig{}
	}
	if backend, ok := cfg.Backends[key]; ok {
		return backend
	}
	return BackendConfig{}
}

func resolveAgentConfig(agentName string) (backend, model, promptFile, reasoning, baseURL, apiKey string, yolo bool) {
	cfg := modelsConfig()
	if agent, ok := cfg.Agents[agentName]; ok {
		backend = strings.TrimSpace(agent.Backend)
		if backend == "" {
			backend = cfg.DefaultBackend
		}
		backendCfg := resolveBackendConfig(cfg, backend)

		baseURL = strings.TrimSpace(agent.BaseURL)
		if baseURL == "" {
			baseURL = strings.TrimSpace(backendCfg.BaseURL)
		}
		apiKey = strings.TrimSpace(agent.APIKey)
		if apiKey == "" {
			apiKey = strings.TrimSpace(backendCfg.APIKey)
		}

		return backend, strings.TrimSpace(agent.Model), agent.PromptFile, agent.Reasoning, baseURL, apiKey, agent.Yolo
	}

	if dynamic, ok := LoadDynamicAgent(agentName); ok {
		backend = cfg.DefaultBackend
		model = cfg.DefaultModel
		backendCfg := resolveBackendConfig(cfg, backend)
		baseURL = strings.TrimSpace(backendCfg.BaseURL)
		apiKey = strings.TrimSpace(backendCfg.APIKey)
		return backend, model, dynamic.PromptFile, "", baseURL, apiKey, false
	}

	backend = cfg.DefaultBackend
	model = cfg.DefaultModel
	backendCfg := resolveBackendConfig(cfg, backend)
	baseURL = strings.TrimSpace(backendCfg.BaseURL)
	apiKey = strings.TrimSpace(backendCfg.APIKey)
	return backend, model, "", "", baseURL, apiKey, false
}

func ResolveAgentConfig(agentName string) (backend, model, promptFile, reasoning, baseURL, apiKey string, yolo bool) {
	return resolveAgentConfig(agentName)
}

func ResetModelsConfigCacheForTest() {
	modelsConfigCached = nil
	modelsConfigOnce = sync.Once{}
}
