package runtask

import (
	"context"
)

// Spec carries application/runtime execution details for a task.
type Spec struct {
	ID              string
	Task            string
	Dependencies    []string
	WorkDir         string
	SessionID       string
	Backend         string
	Model           string
	ReasoningEffort string
	Agent           string
	PromptFile      string
	SkipPermissions bool
	Worktree        bool
	AllowedTools    []string
	DisallowedTools []string
	Skills          []string
	Mode            string
	UseStdin        bool
	Context         context.Context
}
