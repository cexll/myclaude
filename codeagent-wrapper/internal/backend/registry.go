package backend

import (
	"fmt"
	"strings"
)

var registry = map[string]Backend{
	"codex":    CodexBackend{},
	"claude":   ClaudeBackend{},
	"gemini":   GeminiBackend{},
	"opencode": OpencodeBackend{},

	// MiniMax is exposed through both Anthropic-compatible and OpenAI-compatible
	// endpoints. Keep global_en and cn_zh regions distinct so users can select the
	// exact regional base URL required by their account.
	"minimax":                  MiniMaxAnthropicGlobalBackend{},
	"minimax-anthropic":        MiniMaxAnthropicGlobalBackend{},
	"minimax-anthropic-global": MiniMaxAnthropicGlobalBackend{},
	"minimax-anthropic-cn":     MiniMaxAnthropicCNBackend{},
	"minimax-openai":           MiniMaxOpenAIGlobalBackend{},
	"minimax-openai-global":    MiniMaxOpenAIGlobalBackend{},
	"minimax-openai-cn":        MiniMaxOpenAICNBackend{},
}

// Registry exposes the available backends. Intended for internal inspection/tests.
func Registry() map[string]Backend {
	return registry
}

func Select(name string) (Backend, error) {
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" {
		key = "codex"
	}
	if backend, ok := registry[key]; ok {
		return backend, nil
	}
	return nil, fmt.Errorf("unsupported backend %q", name)
}
