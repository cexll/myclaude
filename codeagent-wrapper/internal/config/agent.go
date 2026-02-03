package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/goccy/go-json"
)

type BackendConfig struct {
	BaseURL string `json:"base_url,omitempty"`
	APIKey  string `json:"api_key,omitempty"`
}

type AgentModelConfig struct {
	Backend         string   `json:"backend"`
	Model           string   `json:"model"`
	PromptFile      string   `json:"prompt_file,omitempty"`
	Description     string   `json:"description,omitempty"`
	Yolo            bool     `json:"yolo,omitempty"`
	Reasoning       string   `json:"reasoning,omitempty"`
	BaseURL         string   `json:"base_url,omitempty"`
	APIKey          string   `json:"api_key,omitempty"`
	AllowedTools    []string `json:"allowed_tools,omitempty"`
	DisallowedTools []string `json:"disallowed_tools,omitempty"`
}

type ModelsConfig struct {
	DefaultBackend string                      `json:"default_backend"`
	DefaultModel   string                      `json:"default_model"`
	Agents         map[string]AgentModelConfig `json:"agents"`
	Backends       map[string]BackendConfig    `json:"backends,omitempty"`
}

var defaultModelsConfig = ModelsConfig{}

const modelsConfigTildePath = "~/.codeagent/models.json"

const modelsConfigExample = `{
  "default_backend": "codex",
  "default_model": "gpt-4.1",
  "backends": {
    "codex": { "api_key": "..." },
    "claude": { "api_key": "..." }
  },
  "agents": {
    "develop": {
      "backend": "codex",
      "model": "gpt-4.1",
      "prompt_file": "~/.codeagent/prompts/develop.md",
      "reasoning": "high",
      "yolo": true
    }
  }
}`

var (
	modelsConfigOnce   sync.Once
	modelsConfigCached *ModelsConfig
	modelsConfigErr    error
)

func modelsConfig() (*ModelsConfig, error) {
	modelsConfigOnce.Do(func() {
		modelsConfigCached, modelsConfigErr = loadModelsConfig()
	})
	return modelsConfigCached, modelsConfigErr
}

func modelsConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "", fmt.Errorf("failed to resolve user home directory: %w", err)
	}

	configDir := filepath.Clean(filepath.Join(home, ".codeagent"))
	configPath := filepath.Clean(filepath.Join(configDir, "models.json"))
	rel, err := filepath.Rel(configDir, configPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("refusing to read models config outside %s: %s", configDir, configPath)
	}
	return configPath, nil
}

func modelsConfigHint(configPath string) string {
	configPath = strings.TrimSpace(configPath)
	if configPath == "" {
		return fmt.Sprintf("Create %s with e.g.:\n%s", modelsConfigTildePath, modelsConfigExample)
	}
	return fmt.Sprintf("Create %s (resolved to %s) with e.g.:\n%s", modelsConfigTildePath, configPath, modelsConfigExample)
}

func loadModelsConfig() (*ModelsConfig, error) {
	configPath, err := modelsConfigPath()
	if err != nil {
		return nil, fmt.Errorf("%w\n\n%s", err, modelsConfigHint(""))
	}

	data, err := os.ReadFile(configPath) // #nosec G304 -- path is fixed under user home and validated to stay within configDir
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("models config not found: %s\n\n%s", configPath, modelsConfigHint(configPath))
		}
		return nil, fmt.Errorf("failed to read models config %s: %w\n\n%s", configPath, err, modelsConfigHint(configPath))
	}

	var cfg ModelsConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse models config %s: %w\n\n%s", configPath, err, modelsConfigHint(configPath))
	}

	cfg.DefaultBackend = strings.TrimSpace(cfg.DefaultBackend)
	cfg.DefaultModel = strings.TrimSpace(cfg.DefaultModel)

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

	return &cfg, nil
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
	cfg, err := modelsConfig()
	if err != nil || cfg == nil {
		return "", ""
	}
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

func resolveAgentConfig(agentName string) (backend, model, promptFile, reasoning, baseURL, apiKey string, yolo bool, allowedTools, disallowedTools []string, err error) {
	if err := ValidateAgentName(agentName); err != nil {
		return "", "", "", "", "", "", false, nil, nil, err
	}

	cfg, err := modelsConfig()
	if err != nil {
		return "", "", "", "", "", "", false, nil, nil, err
	}
	if cfg == nil {
		return "", "", "", "", "", "", false, nil, nil, fmt.Errorf("models config is nil\n\n%s", modelsConfigHint(""))
	}

	if agent, ok := cfg.Agents[agentName]; ok {
		backend = strings.TrimSpace(agent.Backend)
		if backend == "" {
			backend = strings.TrimSpace(cfg.DefaultBackend)
			if backend == "" {
				configPath, pathErr := modelsConfigPath()
				if pathErr != nil {
					return "", "", "", "", "", "", false, nil, nil, fmt.Errorf("agent %q has empty backend and default_backend is not set\n\n%s", agentName, modelsConfigHint(""))
				}
				return "", "", "", "", "", "", false, nil, nil, fmt.Errorf("agent %q has empty backend and default_backend is not set\n\n%s", agentName, modelsConfigHint(configPath))
			}
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

		model = strings.TrimSpace(agent.Model)
		if model == "" {
			configPath, pathErr := modelsConfigPath()
			if pathErr != nil {
				return "", "", "", "", "", "", false, nil, nil, fmt.Errorf("agent %q has empty model; set agents.%s.model in %s\n\n%s", agentName, agentName, modelsConfigTildePath, modelsConfigHint(""))
			}
			return "", "", "", "", "", "", false, nil, nil, fmt.Errorf("agent %q has empty model; set agents.%s.model in %s\n\n%s", agentName, agentName, modelsConfigTildePath, modelsConfigHint(configPath))
		}
		return backend, model, agent.PromptFile, agent.Reasoning, baseURL, apiKey, agent.Yolo, agent.AllowedTools, agent.DisallowedTools, nil
	}

	if dynamic, ok := LoadDynamicAgent(agentName); ok {
		backend = strings.TrimSpace(cfg.DefaultBackend)
		model = strings.TrimSpace(cfg.DefaultModel)
		configPath, pathErr := modelsConfigPath()
		if backend == "" || model == "" {
			if pathErr != nil {
				return "", "", "", "", "", "", false, nil, nil, fmt.Errorf("dynamic agent %q requires default_backend and default_model to be set in %s\n\n%s", agentName, modelsConfigTildePath, modelsConfigHint(""))
			}
			return "", "", "", "", "", "", false, nil, nil, fmt.Errorf("dynamic agent %q requires default_backend and default_model to be set in %s\n\n%s", agentName, modelsConfigTildePath, modelsConfigHint(configPath))
		}
		backendCfg := resolveBackendConfig(cfg, backend)
		baseURL = strings.TrimSpace(backendCfg.BaseURL)
		apiKey = strings.TrimSpace(backendCfg.APIKey)
		return backend, model, dynamic.PromptFile, "", baseURL, apiKey, false, nil, nil, nil
	}

	configPath, pathErr := modelsConfigPath()
	if pathErr != nil {
		return "", "", "", "", "", "", false, nil, nil, fmt.Errorf("agent %q not found in %s\n\n%s", agentName, modelsConfigTildePath, modelsConfigHint(""))
	}
	return "", "", "", "", "", "", false, nil, nil, fmt.Errorf("agent %q not found in %s\n\n%s", agentName, modelsConfigTildePath, modelsConfigHint(configPath))
}

func ResolveAgentConfig(agentName string) (backend, model, promptFile, reasoning, baseURL, apiKey string, yolo bool, allowedTools, disallowedTools []string, err error) {
	return resolveAgentConfig(agentName)
}

func ResetModelsConfigCacheForTest() {
	modelsConfigCached = nil
	modelsConfigErr = nil
	modelsConfigOnce = sync.Once{}
}
