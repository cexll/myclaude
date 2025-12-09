package main

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
	if cfg == nil {
		return nil
	}
	// claude -p --dangerously-skip-permissions --output-format stream-json --verbose <prompt>
	args := []string{
		"-p",
		"--dangerously-skip-permissions",
		"--output-format", "stream-json",
		"--verbose",
	}
	return append(args, targetArg)
}

type GeminiBackend struct{}

func (GeminiBackend) Name() string { return "gemini" }
func (GeminiBackend) Command() string {
	return "gemini"
}
func (GeminiBackend) BuildArgs(cfg *Config, targetArg string) []string {
	if cfg == nil {
		return nil
	}
	// gemini -o stream-json -y -p <prompt>
	return []string{"-o", "stream-json", "-y", "-p", targetArg}
}
