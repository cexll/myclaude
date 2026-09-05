package backend

import (
	"os"
	"path/filepath"
	"strings"

	config "codeagent-wrapper/internal/config"
)

type GeminiBackend struct{}

func (GeminiBackend) Name() string    { return "gemini" }
func (GeminiBackend) Command() string { return "gemini" }
func (GeminiBackend) Env(baseURL, apiKey string) map[string]string {
	baseURL = strings.TrimSpace(baseURL)
	apiKey = strings.TrimSpace(apiKey)
	if baseURL == "" && apiKey == "" {
		return nil
	}
	env := make(map[string]string, 2)
	if baseURL != "" {
		env["GOOGLE_GEMINI_BASE_URL"] = baseURL
	}
	if apiKey != "" {
		env["GEMINI_API_KEY"] = apiKey
	}
	return env
}
func (GeminiBackend) BuildArgs(cfg *config.Config, targetArg string) []string {
	return buildGeminiArgs(cfg, targetArg)
}

// LoadGeminiEnv loads environment variables from ~/.gemini/.env
// Supports GEMINI_API_KEY, GEMINI_MODEL, GOOGLE_GEMINI_BASE_URL
// Also sets GEMINI_API_KEY_AUTH_MECHANISM=bearer for third-party API compatibility
func LoadGeminiEnv() map[string]string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return nil
	}

	envDir := filepath.Clean(filepath.Join(home, ".gemini"))
	envPath := filepath.Clean(filepath.Join(envDir, ".env"))
	rel, err := filepath.Rel(envDir, envPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return nil
	}

	data, err := os.ReadFile(envPath) // #nosec G304 -- path is fixed under user home and validated to stay within envDir
	if err != nil {
		return nil
	}

	env := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		if key != "" && value != "" {
			env[key] = value
		}
	}

	// Set bearer auth mechanism for third-party API compatibility
	if _, ok := env["GEMINI_API_KEY"]; ok {
		if _, hasAuth := env["GEMINI_API_KEY_AUTH_MECHANISM"]; !hasAuth {
			env["GEMINI_API_KEY_AUTH_MECHANISM"] = "bearer"
		}
	}

	if len(env) == 0 {
		return nil
	}
	return env
}

func buildGeminiArgs(cfg *config.Config, targetArg string) []string {
	if cfg == nil {
		return nil
	}
	args := []string{"-o", "stream-json", "-y"}

	if model := strings.TrimSpace(cfg.Model); model != "" {
		args = append(args, "-m", model)
	}

	if cfg.Mode == "resume" {
		if cfg.SessionID != "" {
			args = append(args, "-r", cfg.SessionID)
		}
	}

	// Use positional argument instead of deprecated -p flag.
	// For stdin mode ("-"), use -p to read from stdin.
	if targetArg == "-" {
		args = append(args, "-p", targetArg)
	} else {
		args = append(args, targetArg)
	}

	return args
}
