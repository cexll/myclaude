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
	args := []string{"-p", "--dangerously-skip-permissions"}

	// Only skip permissions when explicitly requested
	// if cfg.SkipPermissions {
	// 	args = append(args, "--dangerously-skip-permissions")
	// }

	// Prevent infinite recursion: disable all setting sources (user, project, local)
	// 但需要保留 user 设置以读取登录态/凭据；否则会导致 `Invalid API key · Please run /login`。
	// 因此这里仅禁用 project/local，保留 user。
	args = append(args, "--setting-sources", "user")

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
	if cfg == nil {
		return nil
	}
	args := []string{"-o", "stream-json", "-y"}

	if cfg.Mode == "resume" {
		if cfg.SessionID != "" {
			args = append(args, "-r", cfg.SessionID)
		}
	}
	// Note: gemini CLI doesn't support -C flag; workdir set via cmd.Dir

	args = append(args, "-p", targetArg)

	return args
}
