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
