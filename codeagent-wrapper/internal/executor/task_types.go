package executor

import "context"

// ParallelConfig defines the JSON schema for parallel execution.
type ParallelConfig struct {
	Tasks         []TaskSpec `json:"tasks"`
	GlobalBackend string     `json:"backend,omitempty"`
}

// TaskSpec describes an individual task entry in the parallel config.
type TaskSpec struct {
	ID              string          `json:"id"`
	Task            string          `json:"task"`
	WorkDir         string          `json:"workdir,omitempty"`
	Dependencies    []string        `json:"dependencies,omitempty"`
	SessionID       string          `json:"session_id,omitempty"`
	Backend         string          `json:"backend,omitempty"`
	Model           string          `json:"model,omitempty"`
	ReasoningEffort string          `json:"reasoning_effort,omitempty"`
	Agent           string          `json:"agent,omitempty"`
	PromptFile      string          `json:"prompt_file,omitempty"`
	SkipPermissions bool            `json:"skip_permissions,omitempty"`
	Mode            string          `json:"-"`
	UseStdin        bool            `json:"-"`
	Context         context.Context `json:"-"`
}

// TaskResult captures the execution outcome of a task.
type TaskResult struct {
	TaskID    string `json:"task_id"`
	ExitCode  int    `json:"exit_code"`
	Message   string `json:"message"`
	SessionID string `json:"session_id"`
	Error     string `json:"error"`
	LogPath   string `json:"log_path"`
	// Structured report fields
	Coverage       string   `json:"coverage,omitempty"`        // extracted coverage percentage (e.g., "92%")
	CoverageNum    float64  `json:"coverage_num,omitempty"`    // numeric coverage for comparison
	CoverageTarget float64  `json:"coverage_target,omitempty"` // target coverage (default 90)
	FilesChanged   []string `json:"files_changed,omitempty"`   // list of changed files
	KeyOutput      string   `json:"key_output,omitempty"`      // brief summary of what was done
	TestsPassed    int      `json:"tests_passed,omitempty"`    // number of tests passed
	TestsFailed    int      `json:"tests_failed,omitempty"`    // number of tests failed
	sharedLog      bool
}
