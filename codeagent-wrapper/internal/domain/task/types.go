package task

// TaskSpec describes an individual task entry in the parallel config.
type TaskSpec struct {
	ID           string   `json:"id"`
	Task         string   `json:"task"`
	Dependencies []string `json:"dependencies,omitempty"`
}

// TaskResult captures the execution outcome of a task.
type TaskResult struct {
	TaskID         string   `json:"task_id"`
	ExitCode       int      `json:"exit_code"`
	Message        string   `json:"message"`
	SessionID      string   `json:"session_id"`
	Error          string   `json:"error"`
	LogPath        string   `json:"log_path"`
	Coverage       string   `json:"coverage,omitempty"`
	CoverageNum    float64  `json:"coverage_num,omitempty"`
	CoverageTarget float64  `json:"coverage_target,omitempty"`
	FilesChanged   []string `json:"files_changed,omitempty"`
	KeyOutput      string   `json:"key_output,omitempty"`
	TestsPassed    int      `json:"tests_passed,omitempty"`
	TestsFailed    int      `json:"tests_failed,omitempty"`
	SharedLog      bool     `json:"-"`
}
