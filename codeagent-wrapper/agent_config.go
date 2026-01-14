package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type AgentModelConfig struct {
	Backend     string `json:"backend"`
	Model       string `json:"model"`
	PromptFile  string `json:"prompt_file,omitempty"`
	Description string `json:"description,omitempty"`
	Yolo        bool   `json:"yolo,omitempty"`
	Reasoning   string `json:"reasoning,omitempty"`
}

type ModelsConfig struct {
	DefaultBackend string                      `json:"default_backend"`
	DefaultModel   string                      `json:"default_model"`
	Agents         map[string]AgentModelConfig `json:"agents"`
}

var defaultModelsConfig = ModelsConfig{
	DefaultBackend: "opencode",
	DefaultModel:   "opencode/grok-code",
	Agents: map[string]AgentModelConfig{
			"oracle":                  {Backend: "claude", Model: "claude-sonnet-4-20250514", PromptFile: "~/.claude/skills/omo/references/oracle.md", Description: "Technical advisor"},
			"librarian":               {Backend: "claude", Model: "claude-sonnet-4-5-20250514", PromptFile: "~/.claude/skills/omo/references/librarian.md", Description: "Researcher"},
			"explore":                 {Backend: "opencode", Model: "opencode/grok-code", PromptFile: "~/.claude/skills/omo/references/explore.md", Description: "Code search"},
			"develop":                 {Backend: "codex", Model: "", PromptFile: "~/.claude/skills/omo/references/develop.md", Description: "Code development"},
			"frontend-ui-ux-engineer": {Backend: "gemini", Model: "", PromptFile: "~/.claude/skills/omo/references/frontend-ui-ux-engineer.md", Description: "Frontend engineer"},
			"document-writer":         {Backend: "gemini", Model: "", PromptFile: "~/.claude/skills/omo/references/document-writer.md", Description: "Documentation"},
		},
	}

func loadModelsConfig() *ModelsConfig {
	home, err := os.UserHomeDir()
	if err != nil {
		logWarn(fmt.Sprintf("Failed to resolve home directory for models config: %v; using defaults", err))
		return &defaultModelsConfig
	}

	configPath := filepath.Join(home, ".codeagent", "models.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			logWarn(fmt.Sprintf("Failed to read models config %s: %v; using defaults", configPath, err))
		}
		return &defaultModelsConfig
	}

	var cfg ModelsConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		logWarn(fmt.Sprintf("Failed to parse models config %s: %v; using defaults", configPath, err))
		return &defaultModelsConfig
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

	return &cfg
}

func resolveAgentConfig(agentName string) (backend, model, promptFile, reasoning string, yolo bool) {
	cfg := loadModelsConfig()
	if agent, ok := cfg.Agents[agentName]; ok {
		return agent.Backend, agent.Model, agent.PromptFile, agent.Reasoning, agent.Yolo
	}
	return cfg.DefaultBackend, cfg.DefaultModel, "", "", false
}
