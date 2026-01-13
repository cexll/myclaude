package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds CLI configuration
type Config struct {
	Mode               string // "new" or "resume"
	Task               string
	SessionID          string
	WorkDir            string
	Model              string
	ReasoningEffort    string
	ExplicitStdin      bool
	Timeout            int
	Backend            string
	Agent              string
	PromptFile         string
	PromptFileExplicit bool
	SkipPermissions    bool
	Yolo               bool
	MaxParallelWorkers int
}

// ParallelConfig defines the JSON schema for parallel execution
type ParallelConfig struct {
	Tasks         []TaskSpec `json:"tasks"`
	GlobalBackend string     `json:"backend,omitempty"`
}

// TaskSpec describes an individual task entry in the parallel config
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
	Mode            string          `json:"-"`
	UseStdin        bool            `json:"-"`
	Context         context.Context `json:"-"`
}

// TaskResult captures the execution outcome of a task
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

var backendRegistry = map[string]Backend{
	"codex":    CodexBackend{},
	"claude":   ClaudeBackend{},
	"gemini":   GeminiBackend{},
	"opencode": OpencodeBackend{},
}

func selectBackend(name string) (Backend, error) {
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" {
		key = defaultBackendName
	}
	if backend, ok := backendRegistry[key]; ok {
		return backend, nil
	}
	return nil, fmt.Errorf("unsupported backend %q", name)
}

func envFlagEnabled(key string) bool {
	val, ok := os.LookupEnv(key)
	if !ok {
		return false
	}
	val = strings.TrimSpace(strings.ToLower(val))
	switch val {
	case "", "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

func parseBoolFlag(val string, defaultValue bool) bool {
	val = strings.TrimSpace(strings.ToLower(val))
	switch val {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return defaultValue
	}
}

// envFlagDefaultTrue returns true unless the env var is explicitly set to false/0/no/off.
func envFlagDefaultTrue(key string) bool {
	val, ok := os.LookupEnv(key)
	if !ok {
		return true
	}
	return parseBoolFlag(val, true)
}

func validateAgentName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("agent name is empty")
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-', r == '_':
		default:
			return fmt.Errorf("agent name %q contains invalid character %q", name, r)
		}
	}
	return nil
}

func parseParallelConfig(data []byte) (*ParallelConfig, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("parallel config is empty")
	}

	tasks := strings.Split(string(trimmed), "---TASK---")
	var cfg ParallelConfig
	seen := make(map[string]struct{})

	taskIndex := 0
	for _, taskBlock := range tasks {
		taskBlock = strings.TrimSpace(taskBlock)
		if taskBlock == "" {
			continue
		}
		taskIndex++

		parts := strings.SplitN(taskBlock, "---CONTENT---", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("task block #%d missing ---CONTENT--- separator", taskIndex)
		}

		meta := strings.TrimSpace(parts[0])
		content := strings.TrimSpace(parts[1])

		task := TaskSpec{WorkDir: defaultWorkdir}
		agentSpecified := false
		for _, line := range strings.Split(meta, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			kv := strings.SplitN(line, ":", 2)
			if len(kv) != 2 {
				continue
			}
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])

			switch key {
			case "id":
				task.ID = value
			case "workdir":
				// Validate workdir: "-" is not a valid directory
				if value == "-" {
					return nil, fmt.Errorf("task block #%d has invalid workdir: '-' is not a valid directory path", taskIndex)
				}
				task.WorkDir = value
			case "session_id":
				task.SessionID = value
				task.Mode = "resume"
			case "backend":
				task.Backend = value
			case "model":
				task.Model = value
			case "reasoning_effort":
				task.ReasoningEffort = value
			case "agent":
				agentSpecified = true
				task.Agent = value
			case "dependencies":
				for _, dep := range strings.Split(value, ",") {
					dep = strings.TrimSpace(dep)
					if dep != "" {
						task.Dependencies = append(task.Dependencies, dep)
					}
				}
			}
		}

		if task.Mode == "" {
			task.Mode = "new"
		}

		if agentSpecified {
			if strings.TrimSpace(task.Agent) == "" {
				return nil, fmt.Errorf("task block #%d has empty agent field", taskIndex)
			}
			if err := validateAgentName(task.Agent); err != nil {
				return nil, fmt.Errorf("task block #%d invalid agent name: %w", taskIndex, err)
			}
			backend, model, promptFile, reasoning, _ := resolveAgentConfig(task.Agent)
			if task.Backend == "" {
				task.Backend = backend
			}
			if task.Model == "" {
				task.Model = model
			}
			if task.ReasoningEffort == "" {
				task.ReasoningEffort = reasoning
			}
			task.PromptFile = promptFile
		}

		if task.ID == "" {
			return nil, fmt.Errorf("task block #%d missing id field", taskIndex)
		}
		if content == "" {
			return nil, fmt.Errorf("task block #%d (%q) missing content", taskIndex, task.ID)
		}
		if task.Mode == "resume" && strings.TrimSpace(task.SessionID) == "" {
			return nil, fmt.Errorf("task block #%d (%q) has empty session_id", taskIndex, task.ID)
		}
		if _, exists := seen[task.ID]; exists {
			return nil, fmt.Errorf("task block #%d has duplicate id: %s", taskIndex, task.ID)
		}

		task.Task = content
		cfg.Tasks = append(cfg.Tasks, task)
		seen[task.ID] = struct{}{}
	}

	if len(cfg.Tasks) == 0 {
		return nil, fmt.Errorf("no tasks found")
	}

	return &cfg, nil
}

func parseArgs() (*Config, error) {
	args := os.Args[1:]
	if len(args) == 0 {
		return nil, fmt.Errorf("task required")
	}

	backendName := defaultBackendName
	model := ""
	reasoningEffort := ""
	agentName := ""
	promptFile := ""
	promptFileExplicit := false
	yolo := false
	skipPermissions := envFlagEnabled("CODEAGENT_SKIP_PERMISSIONS")
	filtered := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--agent":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--agent flag requires a value")
			}
			value := strings.TrimSpace(args[i+1])
			if value == "" {
				return nil, fmt.Errorf("--agent flag requires a value")
			}
			if err := validateAgentName(value); err != nil {
				return nil, fmt.Errorf("--agent flag invalid value: %w", err)
			}
			resolvedBackend, resolvedModel, resolvedPromptFile, resolvedReasoning, resolvedYolo := resolveAgentConfig(value)
			backendName = resolvedBackend
			model = resolvedModel
			if !promptFileExplicit {
				promptFile = resolvedPromptFile
			}
			if reasoningEffort == "" {
				reasoningEffort = resolvedReasoning
			}
			yolo = resolvedYolo
			agentName = value
			i++
			continue
		case strings.HasPrefix(arg, "--agent="):
			value := strings.TrimSpace(strings.TrimPrefix(arg, "--agent="))
			if value == "" {
				return nil, fmt.Errorf("--agent flag requires a value")
			}
			if err := validateAgentName(value); err != nil {
				return nil, fmt.Errorf("--agent flag invalid value: %w", err)
			}
			resolvedBackend, resolvedModel, resolvedPromptFile, resolvedReasoning, resolvedYolo := resolveAgentConfig(value)
			backendName = resolvedBackend
			model = resolvedModel
			if !promptFileExplicit {
				promptFile = resolvedPromptFile
			}
			if reasoningEffort == "" {
				reasoningEffort = resolvedReasoning
			}
			yolo = resolvedYolo
			agentName = value
			continue
		case arg == "--prompt-file":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--prompt-file flag requires a value")
			}
			value := strings.TrimSpace(args[i+1])
			if value == "" {
				return nil, fmt.Errorf("--prompt-file flag requires a value")
			}
			promptFile = value
			promptFileExplicit = true
			i++
			continue
		case strings.HasPrefix(arg, "--prompt-file="):
			value := strings.TrimSpace(strings.TrimPrefix(arg, "--prompt-file="))
			if value == "" {
				return nil, fmt.Errorf("--prompt-file flag requires a value")
			}
			promptFile = value
			promptFileExplicit = true
			continue
		case arg == "--backend":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--backend flag requires a value")
			}
			backendName = args[i+1]
			i++
			continue
		case strings.HasPrefix(arg, "--backend="):
			value := strings.TrimPrefix(arg, "--backend=")
			if value == "" {
				return nil, fmt.Errorf("--backend flag requires a value")
			}
			backendName = value
			continue
		case arg == "--skip-permissions", arg == "--dangerously-skip-permissions":
			skipPermissions = true
			continue
		case arg == "--model":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--model flag requires a value")
			}
			model = args[i+1]
			i++
			continue
		case strings.HasPrefix(arg, "--model="):
			value := strings.TrimPrefix(arg, "--model=")
			if value == "" {
				return nil, fmt.Errorf("--model flag requires a value")
			}
			model = value
			continue
		case arg == "--reasoning-effort":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--reasoning-effort flag requires a value")
			}
			value := strings.TrimSpace(args[i+1])
			if value == "" {
				return nil, fmt.Errorf("--reasoning-effort flag requires a value")
			}
			reasoningEffort = value
			i++
			continue
		case strings.HasPrefix(arg, "--reasoning-effort="):
			value := strings.TrimSpace(strings.TrimPrefix(arg, "--reasoning-effort="))
			if value == "" {
				return nil, fmt.Errorf("--reasoning-effort flag requires a value")
			}
			reasoningEffort = value
			continue
		case strings.HasPrefix(arg, "--skip-permissions="):
			skipPermissions = parseBoolFlag(strings.TrimPrefix(arg, "--skip-permissions="), skipPermissions)
			continue
		case strings.HasPrefix(arg, "--dangerously-skip-permissions="):
			skipPermissions = parseBoolFlag(strings.TrimPrefix(arg, "--dangerously-skip-permissions="), skipPermissions)
			continue
		}
		filtered = append(filtered, arg)
	}

	if len(filtered) == 0 {
		return nil, fmt.Errorf("task required")
	}
	args = filtered

	cfg := &Config{WorkDir: defaultWorkdir, Backend: backendName, Agent: agentName, PromptFile: promptFile, PromptFileExplicit: promptFileExplicit, SkipPermissions: skipPermissions, Yolo: yolo, Model: strings.TrimSpace(model), ReasoningEffort: strings.TrimSpace(reasoningEffort)}
	cfg.MaxParallelWorkers = resolveMaxParallelWorkers()

	if args[0] == "resume" {
		if len(args) < 3 {
			return nil, fmt.Errorf("resume mode requires: resume <session_id> <task>")
		}
		cfg.Mode = "resume"
		cfg.SessionID = strings.TrimSpace(args[1])
		if cfg.SessionID == "" {
			return nil, fmt.Errorf("resume mode requires non-empty session_id")
		}
		cfg.Task = args[2]
		cfg.ExplicitStdin = (args[2] == "-")
		if len(args) > 3 {
			// Validate workdir: "-" is not a valid directory
			if args[3] == "-" {
				return nil, fmt.Errorf("invalid workdir: '-' is not a valid directory path")
			}
			cfg.WorkDir = args[3]
		}
	} else {
		cfg.Mode = "new"
		cfg.Task = args[0]
		cfg.ExplicitStdin = (args[0] == "-")
		if len(args) > 1 {
			// Validate workdir: "-" is not a valid directory
			if args[1] == "-" {
				return nil, fmt.Errorf("invalid workdir: '-' is not a valid directory path")
			}
			cfg.WorkDir = args[1]
		}
	}

	return cfg, nil
}

const maxParallelWorkersLimit = 100

func resolveMaxParallelWorkers() int {
	raw := strings.TrimSpace(os.Getenv("CODEAGENT_MAX_PARALLEL_WORKERS"))
	if raw == "" {
		return 0
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		logWarn(fmt.Sprintf("Invalid CODEAGENT_MAX_PARALLEL_WORKERS=%q, falling back to unlimited", raw))
		return 0
	}

	if value > maxParallelWorkersLimit {
		logWarn(fmt.Sprintf("CODEAGENT_MAX_PARALLEL_WORKERS=%d exceeds limit, capping at %d", value, maxParallelWorkersLimit))
		return maxParallelWorkersLimit
	}

	return value
}
