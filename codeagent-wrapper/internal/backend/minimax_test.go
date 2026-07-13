package backend

import (
	"reflect"
	"testing"

	config "codeagent-wrapper/internal/config"
)

func TestMiniMaxBackendMetadata(t *testing.T) {
	tests := []struct {
		backend Backend
		name    string
		command string
	}{
		{backend: MiniMaxAnthropicGlobalBackend{}, name: "minimax-anthropic-global", command: "claude"},
		{backend: MiniMaxAnthropicCNBackend{}, name: "minimax-anthropic-cn", command: "claude"},
		{backend: MiniMaxOpenAIGlobalBackend{}, name: "minimax-openai-global", command: "codex"},
		{backend: MiniMaxOpenAICNBackend{}, name: "minimax-openai-cn", command: "codex"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.backend.Name(); got != tt.name {
				t.Fatalf("Name() = %q, want %q", got, tt.name)
			}
			if got := tt.backend.Command(); got != tt.command {
				t.Fatalf("Command() = %q, want %q", got, tt.command)
			}
		})
	}
}

func TestMiniMaxBackendEnv(t *testing.T) {
	tests := []struct {
		name     string
		backend  Backend
		baseKey  string
		keyName  string
		wantBase string
	}{
		{
			name:     "anthropic global",
			backend:  MiniMaxAnthropicGlobalBackend{},
			baseKey:  "ANTHROPIC_BASE_URL",
			keyName:  "ANTHROPIC_API_KEY",
			wantBase: miniMaxGlobalAnthropicBaseURL,
		},
		{
			name:     "anthropic cn",
			backend:  MiniMaxAnthropicCNBackend{},
			baseKey:  "ANTHROPIC_BASE_URL",
			keyName:  "ANTHROPIC_API_KEY",
			wantBase: miniMaxCNAnthropicBaseURL,
		},
		{
			name:    "openai global",
			backend: MiniMaxOpenAIGlobalBackend{},
			keyName: "OPENAI_API_KEY",
		},
		{
			name:    "openai cn",
			backend: MiniMaxOpenAICNBackend{},
			keyName: "OPENAI_API_KEY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := tt.backend.Env(" ", "test-key")
			if tt.baseKey != "" {
				if got := env[tt.baseKey]; got != tt.wantBase {
					t.Fatalf("%s = %q, want %q", tt.baseKey, got, tt.wantBase)
				}
			} else if _, ok := env["OPENAI_BASE_URL"]; ok {
				t.Fatal("OpenAI-compatible backends must configure the provider through Codex arguments")
			}
			if got := env[tt.keyName]; got != "test-key" {
				t.Fatalf("%s = %q, want test-key", tt.keyName, got)
			}

			if tt.baseKey != "" {
				const override = "https://gateway.example/api"
				overridden := tt.backend.Env(override, "test-key")
				if got := overridden[tt.baseKey]; got != override {
					t.Fatalf("override %s = %q, want %q", tt.baseKey, got, override)
				}
			}
		})
	}
}

func TestMiniMaxBackendBuildArgs(t *testing.T) {
	t.Run("anthropic forwards model", func(t *testing.T) {
		t.Setenv("CODEAGENT_SKIP_PERMISSIONS", "false")
		got := MiniMaxAnthropicGlobalBackend{}.BuildArgs(&config.Config{
			Mode:  "new",
			Model: "MiniMax-M3",
		}, "task")
		want := []string{"-p", "--model", "MiniMax-M3", "--output-format", "stream-json", "--verbose", "task"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("BuildArgs() = %v, want %v", got, want)
		}
	})

	openAITests := []struct {
		name       string
		backend    Backend
		providerID string
		baseURL    string
	}{
		{
			name:       "openai global forwards model and provider",
			backend:    MiniMaxOpenAIGlobalBackend{},
			providerID: "minimax_openai_global",
			baseURL:    miniMaxGlobalOpenAIBaseURL,
		},
		{
			name:       "openai cn forwards model and provider",
			backend:    MiniMaxOpenAICNBackend{},
			providerID: "minimax_openai_cn",
			baseURL:    miniMaxCNOpenAIBaseURL,
		},
	}
	for _, tt := range openAITests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("CODEX_BYPASS_SANDBOX", "false")
			got := tt.backend.BuildArgs(&config.Config{
				Mode:    "new",
				Model:   "MiniMax-M2.7",
				WorkDir: "/workspace",
			}, "task")
			want := []string{
				"e",
				"-c", "model_provider=\"" + tt.providerID + "\"",
				"-c", "model_providers." + tt.providerID + ".name=\"MiniMax\"",
				"-c", "model_providers." + tt.providerID + ".base_url=\"" + tt.baseURL + "\"",
				"-c", "model_providers." + tt.providerID + ".env_key=\"OPENAI_API_KEY\"",
				"-c", "model_providers." + tt.providerID + ".wire_api=\"responses\"",
				"-c", "model_providers." + tt.providerID + ".supports_websockets=false",
				"--model", "MiniMax-M2.7", "--skip-git-repo-check", "-C", "/workspace", "--json", "task",
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("BuildArgs() = %v, want %v", got, want)
			}
		})
	}
}

func TestMiniMaxRegistryAliases(t *testing.T) {
	tests := map[string]string{
		"minimax":                  "minimax-anthropic-global",
		"minimax-anthropic":        "minimax-anthropic-global",
		"minimax-anthropic-global": "minimax-anthropic-global",
		"minimax-anthropic-cn":     "minimax-anthropic-cn",
		"minimax-openai":           "minimax-openai-global",
		"minimax-openai-global":    "minimax-openai-global",
		"minimax-openai-cn":        "minimax-openai-cn",
	}

	for alias, wantName := range tests {
		t.Run(alias, func(t *testing.T) {
			selected, err := Select(alias)
			if err != nil {
				t.Fatalf("Select(%q): %v", alias, err)
			}
			if got := selected.Name(); got != wantName {
				t.Fatalf("Select(%q).Name() = %q, want %q", alias, got, wantName)
			}
		})
	}
}
