package backend

import (
	"strings"

	config "codeagent-wrapper/internal/config"
)

type OpencodeBackend struct{}

func (OpencodeBackend) Name() string                                 { return "opencode" }
func (OpencodeBackend) Command() string                              { return "opencode" }
func (OpencodeBackend) Env(baseURL, apiKey string) map[string]string { return nil }
func (OpencodeBackend) BuildArgs(cfg *config.Config, targetArg string) []string {
	args := []string{"run"}
	if cfg != nil {
		if model := strings.TrimSpace(cfg.Model); model != "" {
			args = append(args, "-m", model)
		}
		if cfg.Mode == "resume" && cfg.SessionID != "" {
			args = append(args, "-s", cfg.SessionID)
		}
	}
	args = append(args, "--format", "json")
	if targetArg != "-" {
		args = append(args, targetArg)
	}
	return args
}
