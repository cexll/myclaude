package backend

import config "codeagent-wrapper/internal/config"

const (
	MiniMaxGlobalOpenAIBaseURL      = "https://api.minimax.io/v1"
	MiniMaxCNOpenAIBaseURL          = "https://api.minimaxi.com/v1"
	MiniMaxGlobalAnthropicBaseURL   = "https://api.minimax.io/anthropic"
	MiniMaxCNAnthropicBaseURL       = "https://api.minimaxi.com/anthropic"
	MiniMaxM3ModelID                = "MiniMax-M3"
	MiniMaxM27ModelID               = "MiniMax-M2.7"
	MiniMaxM3ContextWindow          = 1000000
	MiniMaxM27ContextWindow         = 204800
	MiniMaxM3InputPricePerMillion   = 0.6
	MiniMaxM3OutputPricePerMillion  = 2.4
	MiniMaxM3CacheReadPerMillion    = 0.12
	MiniMaxM27InputPricePerMillion  = 0.3
	MiniMaxM27OutputPricePerMillion = 1.2
	MiniMaxM27CacheReadPerMillion   = 0.06
	MiniMaxM27CacheWritePerMillion  = 0.375
)

type MiniMaxAnthropicGlobalBackend struct{}

type MiniMaxAnthropicCNBackend struct{}

type MiniMaxOpenAIGlobalBackend struct{}

type MiniMaxOpenAICNBackend struct{}

func (MiniMaxAnthropicGlobalBackend) Name() string    { return "minimax-anthropic-global" }
func (MiniMaxAnthropicGlobalBackend) Command() string { return ClaudeBackend{}.Command() }
func (MiniMaxAnthropicGlobalBackend) Env(baseURL, apiKey string) map[string]string {
	return ClaudeBackend{}.Env(defaultString(baseURL, MiniMaxGlobalAnthropicBaseURL), apiKey)
}
func (MiniMaxAnthropicGlobalBackend) BuildArgs(cfg *config.Config, targetArg string) []string {
	return ClaudeBackend{}.BuildArgs(cfg, targetArg)
}

func (MiniMaxAnthropicCNBackend) Name() string    { return "minimax-anthropic-cn" }
func (MiniMaxAnthropicCNBackend) Command() string { return ClaudeBackend{}.Command() }
func (MiniMaxAnthropicCNBackend) Env(baseURL, apiKey string) map[string]string {
	return ClaudeBackend{}.Env(defaultString(baseURL, MiniMaxCNAnthropicBaseURL), apiKey)
}
func (MiniMaxAnthropicCNBackend) BuildArgs(cfg *config.Config, targetArg string) []string {
	return ClaudeBackend{}.BuildArgs(cfg, targetArg)
}

func (MiniMaxOpenAIGlobalBackend) Name() string    { return "minimax-openai-global" }
func (MiniMaxOpenAIGlobalBackend) Command() string { return CodexBackend{}.Command() }
func (MiniMaxOpenAIGlobalBackend) Env(baseURL, apiKey string) map[string]string {
	return CodexBackend{}.Env(defaultString(baseURL, MiniMaxGlobalOpenAIBaseURL), apiKey)
}
func (MiniMaxOpenAIGlobalBackend) BuildArgs(cfg *config.Config, targetArg string) []string {
	return CodexBackend{}.BuildArgs(cfg, targetArg)
}

func (MiniMaxOpenAICNBackend) Name() string    { return "minimax-openai-cn" }
func (MiniMaxOpenAICNBackend) Command() string { return CodexBackend{}.Command() }
func (MiniMaxOpenAICNBackend) Env(baseURL, apiKey string) map[string]string {
	return CodexBackend{}.Env(defaultString(baseURL, MiniMaxCNOpenAIBaseURL), apiKey)
}
func (MiniMaxOpenAICNBackend) BuildArgs(cfg *config.Config, targetArg string) []string {
	return CodexBackend{}.BuildArgs(cfg, targetArg)
}

func defaultString(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
