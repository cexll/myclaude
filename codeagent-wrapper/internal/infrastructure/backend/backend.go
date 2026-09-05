package backend

import (
	legacy "codeagent-wrapper/internal/backend"
	config "codeagent-wrapper/internal/config"
)

type Backend = legacy.Backend
type CodexBackend = legacy.CodexBackend
type ClaudeBackend = legacy.ClaudeBackend
type GeminiBackend = legacy.GeminiBackend
type OpencodeBackend = legacy.OpencodeBackend
type MinimalClaudeSettings = legacy.MinimalClaudeSettings

func Registry() map[string]Backend { return legacy.Registry() }

func Select(name string) (Backend, error) { return legacy.Select(name) }

func SetLogFuncs(warnFn, errorFn func(string)) { legacy.SetLogFuncs(warnFn, errorFn) }

func BuildCodexArgs(cfg *config.Config, targetArg string) []string {
	return legacy.BuildCodexArgs(cfg, targetArg)
}

func LoadMinimalClaudeSettings() MinimalClaudeSettings { return legacy.LoadMinimalClaudeSettings() }

func LoadMinimalEnvSettings() map[string]string { return legacy.LoadMinimalEnvSettings() }

func LoadGeminiEnv() map[string]string { return legacy.LoadGeminiEnv() }
