package backend

import (
	"os"
	"path/filepath"
	"strings"

	config "codeagent-wrapper/internal/config"

	"github.com/goccy/go-json"
)

type ClaudeBackend struct{}

func (ClaudeBackend) Name() string    { return "claude" }
func (ClaudeBackend) Command() string { return "claude" }
func (ClaudeBackend) Env(baseURL, apiKey string) map[string]string {
	baseURL = strings.TrimSpace(baseURL)
	apiKey = strings.TrimSpace(apiKey)
	if baseURL == "" && apiKey == "" {
		return nil
	}
	env := make(map[string]string, 2)
	if baseURL != "" {
		env["ANTHROPIC_BASE_URL"] = baseURL
	}
	if apiKey != "" {
		env["ANTHROPIC_API_KEY"] = apiKey
	}
	return env
}
func (ClaudeBackend) BuildArgs(cfg *config.Config, targetArg string) []string {
	return buildClaudeArgs(cfg, targetArg)
}

const MaxClaudeSettingsBytes = 1 << 20 // 1MB

type MinimalClaudeSettings struct {
	Env   map[string]string
	Model string
}

// LoadMinimalClaudeSettings 从 ~/.claude/settings.json 只提取安全的最小子集：
// - env: 只接受字符串类型的值
// - model: 只接受字符串类型的值
// 文件缺失/解析失败/超限都返回空。
func LoadMinimalClaudeSettings() MinimalClaudeSettings {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return MinimalClaudeSettings{}
	}

	claudeDir := filepath.Clean(filepath.Join(home, ".claude"))
	settingPath := filepath.Clean(filepath.Join(claudeDir, "settings.json"))
	rel, err := filepath.Rel(claudeDir, settingPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return MinimalClaudeSettings{}
	}

	info, err := os.Stat(settingPath)
	if err != nil || info.Size() > MaxClaudeSettingsBytes {
		return MinimalClaudeSettings{}
	}

	data, err := os.ReadFile(settingPath) // #nosec G304 -- path is fixed under user home and validated to stay within claudeDir
	if err != nil {
		return MinimalClaudeSettings{}
	}

	var cfg struct {
		Env   map[string]any `json:"env"`
		Model any            `json:"model"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return MinimalClaudeSettings{}
	}

	out := MinimalClaudeSettings{}

	if model, ok := cfg.Model.(string); ok {
		out.Model = strings.TrimSpace(model)
	}

	if len(cfg.Env) == 0 {
		return out
	}

	env := make(map[string]string, len(cfg.Env))
	for k, v := range cfg.Env {
		s, ok := v.(string)
		if !ok {
			continue
		}
		env[k] = s
	}
	if len(env) == 0 {
		return out
	}
	out.Env = env
	return out
}

func LoadMinimalEnvSettings() map[string]string {
	settings := LoadMinimalClaudeSettings()
	if len(settings.Env) == 0 {
		return nil
	}
	return settings.Env
}

func buildClaudeArgs(cfg *config.Config, targetArg string) []string {
	if cfg == nil {
		return nil
	}
	args := []string{"-p"}
	// Default to skip permissions unless CODEAGENT_SKIP_PERMISSIONS=false
	if cfg.SkipPermissions || cfg.Yolo || config.EnvFlagDefaultTrue("CODEAGENT_SKIP_PERMISSIONS") {
		args = append(args, "--dangerously-skip-permissions")
	}

	// Prevent infinite recursion: disable all setting sources (user, project, local)
	// This ensures a clean execution environment without CLAUDE.md or skills that would trigger codeagent
	args = append(args, "--setting-sources", "")

	if model := strings.TrimSpace(cfg.Model); model != "" {
		args = append(args, "--model", model)
	}

	if cfg.Mode == "resume" {
		if cfg.SessionID != "" {
			// Claude CLI uses -r <session_id> for resume.
			args = append(args, "-r", cfg.SessionID)
		}
	}

	args = append(args, "--output-format", "stream-json", "--verbose", targetArg)

	return args
}
