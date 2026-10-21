package backend

import (
	"fmt"
	"strings"

	config "codeagent-wrapper/internal/config"
)

const (
	miniMaxGlobalOpenAIBaseURL    = "https://api.minimax.io/v1"
	miniMaxCNOpenAIBaseURL        = "https://api.minimaxi.com/v1"
	miniMaxGlobalAnthropicBaseURL = "https://api.minimax.io/anthropic"
	miniMaxCNAnthropicBaseURL     = "https://api.minimaxi.com/anthropic"
)

type MiniMaxAnthropicGlobalBackend struct{}

type MiniMaxAnthropicCNBackend struct{}

type MiniMaxOpenAIGlobalBackend struct{}

type MiniMaxOpenAICNBackend struct{}

func (MiniMaxAnthropicGlobalBackend) Name() string    { return "minimax-anthropic-global" }
func (MiniMaxAnthropicGlobalBackend) Command() string { return ClaudeBackend{}.Command() }
func (MiniMaxAnthropicGlobalBackend) Env(baseURL, apiKey string) map[string]string {
	return ClaudeBackend{}.Env(defaultBaseURL(baseURL, miniMaxGlobalAnthropicBaseURL), apiKey)
}
func (MiniMaxAnthropicGlobalBackend) BuildArgs(cfg *config.Config, targetArg string) []string {
	return ClaudeBackend{}.BuildArgs(miniMaxDefaultConfig(cfg), targetArg)
}

func (MiniMaxAnthropicCNBackend) Name() string    { return "minimax-anthropic-cn" }
func (MiniMaxAnthropicCNBackend) Command() string { return ClaudeBackend{}.Command() }
func (MiniMaxAnthropicCNBackend) Env(baseURL, apiKey string) map[string]string {
	return ClaudeBackend{}.Env(defaultBaseURL(baseURL, miniMaxCNAnthropicBaseURL), apiKey)
}
func (MiniMaxAnthropicCNBackend) BuildArgs(cfg *config.Config, targetArg string) []string {
	return ClaudeBackend{}.BuildArgs(miniMaxDefaultConfig(cfg), targetArg)
}

func (MiniMaxOpenAIGlobalBackend) Name() string    { return "minimax-openai-global" }
func (MiniMaxOpenAIGlobalBackend) Command() string { return CodexBackend{}.Command() }
func (MiniMaxOpenAIGlobalBackend) Env(_ string, apiKey string) map[string]string {
	return CodexBackend{}.Env("", apiKey)
}
func (MiniMaxOpenAIGlobalBackend) BuildArgs(cfg *config.Config, targetArg string) []string {
	return buildMiniMaxOpenAIArgs(cfg, targetArg, "minimax_openai_global", miniMaxGlobalOpenAIBaseURL)
}

func (MiniMaxOpenAICNBackend) Name() string    { return "minimax-openai-cn" }
func (MiniMaxOpenAICNBackend) Command() string { return CodexBackend{}.Command() }
func (MiniMaxOpenAICNBackend) Env(_ string, apiKey string) map[string]string {
	return CodexBackend{}.Env("", apiKey)
}
func (MiniMaxOpenAICNBackend) BuildArgs(cfg *config.Config, targetArg string) []string {
	return buildMiniMaxOpenAIArgs(cfg, targetArg, "minimax_openai_cn", miniMaxCNOpenAIBaseURL)
}

func defaultBaseURL(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

// Default only new, unconfigured MiniMax sessions. Explicit models (including
// legacy/custom IDs) and resumed sessions retain the caller's selection.
func miniMaxDefaultConfig(cfg *config.Config) *config.Config {
	if cfg == nil || cfg.Mode == "resume" || strings.TrimSpace(cfg.Model) != "" {
		return cfg
	}
	resolved := *cfg
	resolved.Model = "MiniMax-M3.1-Flash"
	return &resolved
}

func buildMiniMaxOpenAIArgs(cfg *config.Config, targetArg, providerID, baseURL string) []string {
	args := CodexBackend{}.BuildArgs(miniMaxDefaultConfig(cfg), targetArg)
	providerArgs := []string{
		"-c", fmt.Sprintf("model_provider=%q", providerID),
		"-c", fmt.Sprintf("model_providers.%s.name=%q", providerID, "MiniMax"),
		"-c", fmt.Sprintf("model_providers.%s.base_url=%q", providerID, baseURL),
		"-c", fmt.Sprintf("model_providers.%s.env_key=%q", providerID, "OPENAI_API_KEY"),
		"-c", fmt.Sprintf("model_providers.%s.wire_api=%q", providerID, "responses"),
		"-c", fmt.Sprintf("model_providers.%s.supports_websockets=false", providerID),
	}

	result := make([]string, 0, len(args)+len(providerArgs))
	result = append(result, args[0])
	result = append(result, providerArgs...)
	return append(result, args[1:]...)
}
