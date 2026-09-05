package runtask

import config "codeagent-wrapper/internal/config"

func PrepareConfigPlan(cfg *config.Config, deps PrepareDeps) (Plan, error) {
	return PreparePlan(Command{
		Task:               cfg.Task,
		WorkDir:            cfg.WorkDir,
		Mode:               cfg.Mode,
		SessionID:          cfg.SessionID,
		Backend:            cfg.Backend,
		Model:              cfg.Model,
		ReasoningEffort:    cfg.ReasoningEffort,
		Agent:              cfg.Agent,
		SkipPermissions:    cfg.SkipPermissions,
		Worktree:           cfg.Worktree,
		AllowedTools:       cfg.AllowedTools,
		DisallowedTools:    cfg.DisallowedTools,
		ExplicitStdin:      cfg.ExplicitStdin,
		PromptFile:         cfg.PromptFile,
		PromptFileExplicit: cfg.PromptFileExplicit,
		Skills:             cfg.Skills,
	}, deps)
}
