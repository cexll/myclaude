package executor

import (
	"strings"
	"testing"

	backend "codeagent-wrapper/internal/backend"
)

func TestMaskSensitiveValue(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		expected string
	}{
		{
			name:     "API_KEY with long value",
			key:      "ANTHROPIC_API_KEY",
			value:    "sk-ant-api03-xxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			expected: "sk-a****xxxx",
		},
		{
			name:     "api_key lowercase",
			key:      "api_key",
			value:    "abcdefghijklmnop",
			expected: "abcd****mnop",
		},
		{
			name:     "AUTH_TOKEN",
			key:      "AUTH_TOKEN",
			value:    "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
			expected: "eyJh****VCJ9",
		},
		{
			name:     "SECRET",
			key:      "MY_SECRET",
			value:    "super-secret-value-12345",
			expected: "supe****2345",
		},
		{
			name:     "short key value (8 chars)",
			key:      "API_KEY",
			value:    "12345678",
			expected: "****",
		},
		{
			name:     "very short key value",
			key:      "API_KEY",
			value:    "abc",
			expected: "****",
		},
		{
			name:     "empty key value",
			key:      "API_KEY",
			value:    "",
			expected: "",
		},
		{
			name:     "non-sensitive BASE_URL",
			key:      "ANTHROPIC_BASE_URL",
			value:    "https://api.anthropic.com",
			expected: "https://api.anthropic.com",
		},
		{
			name:     "non-sensitive MODEL",
			key:      "MODEL",
			value:    "claude-3-opus",
			expected: "claude-3-opus",
		},
		{
			name:     "case insensitive - Key",
			key:      "My_Key",
			value:    "1234567890abcdef",
			expected: "1234****cdef",
		},
		{
			name:     "case insensitive - TOKEN",
			key:      "ACCESS_TOKEN",
			value:    "access123456789",
			expected: "acce****6789",
		},
		{
			name:     "partial match - apikey",
			key:      "MYAPIKEY",
			value:    "1234567890",
			expected: "1234****7890",
		},
		{
			name:     "partial match - secretvalue",
			key:      "SECRETVALUE",
			value:    "abcdefghij",
			expected: "abcd****ghij",
		},
		{
			name:     "9 char value (just above threshold)",
			key:      "API_KEY",
			value:    "123456789",
			expected: "1234****6789",
		},
		{
			name:     "exactly 8 char value (at threshold)",
			key:      "API_KEY",
			value:    "12345678",
			expected: "****",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskSensitiveValue(tt.key, tt.value)
			if result != tt.expected {
				t.Errorf("maskSensitiveValue(%q, %q) = %q, want %q", tt.key, tt.value, result, tt.expected)
			}
		})
	}
}

func TestMaskSensitiveValue_NoLeakage(t *testing.T) {
	// Ensure sensitive values are never fully exposed
	sensitiveKeys := []string{"API_KEY", "api_key", "AUTH_TOKEN", "SECRET", "access_token", "MYAPIKEY"}
	longValue := "this-is-a-very-long-secret-value-that-should-be-masked"

	for _, key := range sensitiveKeys {
		t.Run(key, func(t *testing.T) {
			masked := maskSensitiveValue(key, longValue)
			// Should not contain the full value
			if masked == longValue {
				t.Errorf("key %q: value was not masked", key)
			}
			// Should contain mask marker
			if !strings.Contains(masked, "****") {
				t.Errorf("key %q: masked value %q does not contain ****", key, masked)
			}
			// First 4 chars should be visible
			if !strings.HasPrefix(masked, longValue[:4]) {
				t.Errorf("key %q: masked value should start with first 4 chars", key)
			}
			// Last 4 chars should be visible
			if !strings.HasSuffix(masked, longValue[len(longValue)-4:]) {
				t.Errorf("key %q: masked value should end with last 4 chars", key)
			}
		})
	}
}

func TestMaskSensitiveValue_NonSensitivePassthrough(t *testing.T) {
	// Non-sensitive keys should pass through unchanged
	nonSensitiveKeys := []string{
		"ANTHROPIC_BASE_URL",
		"BASE_URL",
		"MODEL",
		"BACKEND",
		"WORKDIR",
		"HOME",
		"PATH",
	}
	value := "any-value-here-12345"

	for _, key := range nonSensitiveKeys {
		t.Run(key, func(t *testing.T) {
			result := maskSensitiveValue(key, value)
			if result != value {
				t.Errorf("key %q: expected passthrough but got %q", key, result)
			}
		})
	}
}

// TestClaudeBackendEnv tests that ClaudeBackend.Env returns correct env vars
func TestClaudeBackendEnv(t *testing.T) {
	tests := []struct {
		name       string
		baseURL    string
		apiKey     string
		expectKeys []string
		expectNil  bool
	}{
		{
			name:       "both base_url and api_key",
			baseURL:    "https://api.custom.com",
			apiKey:     "sk-test-key-12345",
			expectKeys: []string{"ANTHROPIC_BASE_URL", "ANTHROPIC_API_KEY"},
		},
		{
			name:       "only base_url",
			baseURL:    "https://api.custom.com",
			apiKey:     "",
			expectKeys: []string{"ANTHROPIC_BASE_URL"},
		},
		{
			name:       "only api_key",
			baseURL:    "",
			apiKey:     "sk-test-key-12345",
			expectKeys: []string{"ANTHROPIC_API_KEY"},
		},
		{
			name:      "both empty",
			baseURL:   "",
			apiKey:    "",
			expectNil: true,
		},
		{
			name:      "whitespace only",
			baseURL:   "   ",
			apiKey:    "  ",
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := backend.ClaudeBackend{}
			env := b.Env(tt.baseURL, tt.apiKey)

			if tt.expectNil {
				if env != nil {
					t.Errorf("expected nil env, got %v", env)
				}
				return
			}

			if env == nil {
				t.Fatal("expected non-nil env")
			}

			for _, key := range tt.expectKeys {
				if _, ok := env[key]; !ok {
					t.Errorf("expected key %q in env", key)
				}
			}

			// Verify values are correct
			if tt.baseURL != "" && strings.TrimSpace(tt.baseURL) != "" {
				if env["ANTHROPIC_BASE_URL"] != strings.TrimSpace(tt.baseURL) {
					t.Errorf("ANTHROPIC_BASE_URL = %q, want %q", env["ANTHROPIC_BASE_URL"], strings.TrimSpace(tt.baseURL))
				}
			}
			if tt.apiKey != "" && strings.TrimSpace(tt.apiKey) != "" {
				if env["ANTHROPIC_API_KEY"] != strings.TrimSpace(tt.apiKey) {
					t.Errorf("ANTHROPIC_API_KEY = %q, want %q", env["ANTHROPIC_API_KEY"], strings.TrimSpace(tt.apiKey))
				}
			}
		})
	}
}

// TestEnvLoggingIntegration tests that env vars are properly masked in logs
func TestEnvLoggingIntegration(t *testing.T) {
	b := backend.ClaudeBackend{}
	baseURL := "https://api.minimaxi.com/anthropic"
	apiKey := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.longjwttoken"

	env := b.Env(baseURL, apiKey)
	if env == nil {
		t.Fatal("expected non-nil env")
	}

	// Verify that when we log these values, sensitive ones are masked
	for k, v := range env {
		masked := maskSensitiveValue(k, v)

		if k == "ANTHROPIC_BASE_URL" {
			// URL should not be masked
			if masked != v {
				t.Errorf("BASE_URL should not be masked: got %q, want %q", masked, v)
			}
		}

		if k == "ANTHROPIC_API_KEY" {
			// API key should be masked
			if masked == v {
				t.Errorf("API_KEY should be masked, but got original value")
			}
			if !strings.Contains(masked, "****") {
				t.Errorf("masked API_KEY should contain ****: got %q", masked)
			}
			// Should still show first 4 and last 4 chars
			if !strings.HasPrefix(masked, v[:4]) {
				t.Errorf("masked value should start with first 4 chars of original")
			}
			if !strings.HasSuffix(masked, v[len(v)-4:]) {
				t.Errorf("masked value should end with last 4 chars of original")
			}
		}
	}
}

// TestGeminiBackendEnv tests GeminiBackend.Env for comparison
func TestGeminiBackendEnv(t *testing.T) {
	b := backend.GeminiBackend{}
	env := b.Env("https://custom.api", "gemini-api-key-12345")

	if env == nil {
		t.Fatal("expected non-nil env")
	}

	// Check that GEMINI env vars are set
	if _, ok := env["GOOGLE_GEMINI_BASE_URL"]; !ok {
		t.Error("expected GOOGLE_GEMINI_BASE_URL in env")
	}
	if _, ok := env["GEMINI_API_KEY"]; !ok {
		t.Error("expected GEMINI_API_KEY in env")
	}

	// Verify masking works for Gemini keys too
	for k, v := range env {
		masked := maskSensitiveValue(k, v)
		if strings.Contains(strings.ToLower(k), "key") {
			if masked == v && len(v) > 0 {
				t.Errorf("key %q should be masked", k)
			}
		}
	}
}

// TestCodexBackendEnv tests CodexBackend.Env
func TestCodexBackendEnv(t *testing.T) {
	b := backend.CodexBackend{}
	env := b.Env("https://custom.api", "codex-api-key-12345")

	if env == nil {
		t.Fatal("expected non-nil env for codex")
	}

	// Check for OPENAI env vars
	if _, ok := env["OPENAI_BASE_URL"]; !ok {
		t.Error("expected OPENAI_BASE_URL in env")
	}
	if _, ok := env["OPENAI_API_KEY"]; !ok {
		t.Error("expected OPENAI_API_KEY in env")
	}
}
