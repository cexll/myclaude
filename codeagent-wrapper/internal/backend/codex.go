package backend

import (
	"strings"

	config "codeagent-wrapper/internal/config"
)

type CodexBackend struct{}

func (CodexBackend) Name() string    { return "codex" }
func (CodexBackend) Command() string { return "codex" }
func (CodexBackend) Env(baseURL, apiKey string) map[string]string {
	baseURL = strings.TrimSpace(baseURL)
	apiKey = strings.TrimSpace(apiKey)
	if baseURL == "" && apiKey == "" {
		return nil
	}
	env := make(map[string]string, 2)
	if baseURL != "" {
		env["OPENAI_BASE_URL"] = baseURL
	}
	if apiKey != "" {
		env["OPENAI_API_KEY"] = apiKey
	}
	return env
}
func (CodexBackend) BuildArgs(cfg *config.Config, targetArg string) []string {
	return BuildCodexArgs(cfg, targetArg)
}

func BuildCodexArgs(cfg *config.Config, targetArg string) []string {
	if cfg == nil {
		panic("buildCodexArgs: nil config")
	}

	var resumeSessionID string
	isResume := cfg.Mode == "resume"
	if isResume {
		resumeSessionID = strings.TrimSpace(cfg.SessionID)
		if resumeSessionID == "" {
			logErrorFn("invalid config: resume mode requires non-empty session_id")
			isResume = false
		}
	}

	args := []string{"e"}

	// Default to bypass sandbox unless CODEX_BYPASS_SANDBOX=false
	if cfg.Yolo || config.EnvFlagDefaultTrue("CODEX_BYPASS_SANDBOX") {
		logWarnFn("YOLO mode or CODEX_BYPASS_SANDBOX enabled: running without approval/sandbox protection")
		args = append(args, "--dangerously-bypass-approvals-and-sandbox")
	}

	if model := strings.TrimSpace(cfg.Model); model != "" {
		args = append(args, "--model", model)
	}

	if reasoningEffort := strings.TrimSpace(cfg.ReasoningEffort); reasoningEffort != "" {
		args = append(args, "-c", "model_reasoning_effort="+reasoningEffort)
	}

	args = append(args, "--skip-git-repo-check")

	if isResume {
		return append(args,
			"--json",
			"resume",
			resumeSessionID,
			targetArg,
		)
	}

	return append(args,
		"-C", cfg.WorkDir,
		"--json",
		targetArg,
	)
}
