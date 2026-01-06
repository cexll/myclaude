package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Backend defines the contract for invoking different AI CLI backends.
// Each backend is responsible for supplying the executable command and
// building the argument list based on the wrapper config.
type Backend interface {
	Name() string
	BuildArgs(cfg *Config, targetArg string) []string
	Command() string
}

type CodexBackend struct{}

func (CodexBackend) Name() string { return "codex" }
func (CodexBackend) Command() string {
	return "codex"
}
func (CodexBackend) BuildArgs(cfg *Config, targetArg string) []string {
	return buildCodexArgs(cfg, targetArg)
}

type ClaudeBackend struct{}

func (ClaudeBackend) Name() string { return "claude" }
func (ClaudeBackend) Command() string {
	return "claude"
}
func (ClaudeBackend) BuildArgs(cfg *Config, targetArg string) []string {
	return buildClaudeArgs(cfg, targetArg)
}

const maxClaudeSettingsBytes = 1 << 20 // 1MB

type minimalClaudeSettings struct {
	Env   map[string]string
	Model string
}

// loadMinimalClaudeSettings 从 ~/.claude/settings.json 只提取安全的最小子集：
// - env: 只接受字符串类型的值
// - model: 只接受字符串类型的值
// 文件缺失/解析失败/超限都返回空。
func loadMinimalClaudeSettings() minimalClaudeSettings {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return minimalClaudeSettings{}
	}

	settingPath := filepath.Join(home, ".claude", "settings.json")
	info, err := os.Stat(settingPath)
	if err != nil || info.Size() > maxClaudeSettingsBytes {
		return minimalClaudeSettings{}
	}

	data, err := os.ReadFile(settingPath)
	if err != nil {
		return minimalClaudeSettings{}
	}

	var cfg struct {
		Env   map[string]any `json:"env"`
		Model any            `json:"model"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return minimalClaudeSettings{}
	}

	out := minimalClaudeSettings{}

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

// loadMinimalEnvSettings is kept for backwards tests; prefer loadMinimalClaudeSettings.
func loadMinimalEnvSettings() map[string]string {
	settings := loadMinimalClaudeSettings()
	if len(settings.Env) == 0 {
		return nil
	}
	return settings.Env
}

func buildClaudeArgs(cfg *Config, targetArg string) []string {
	if cfg == nil {
		return nil
	}
	args := []string{"-p"}
	if cfg.SkipPermissions {
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
	// Note: claude CLI doesn't support -C flag; workdir set via cmd.Dir

	args = append(args, "--output-format", "stream-json", "--verbose", targetArg)

	return args
}

type GeminiBackend struct{}

func (GeminiBackend) Name() string { return "gemini" }
func (GeminiBackend) Command() string {
	return "gemini"
}
func (GeminiBackend) BuildArgs(cfg *Config, targetArg string) []string {
	return buildGeminiArgs(cfg, targetArg)
}

func buildGeminiArgs(cfg *Config, targetArg string) []string {
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
	// Note: gemini CLI doesn't support -C flag; workdir set via cmd.Dir

	args = append(args, "-p", targetArg)

	return args
}
