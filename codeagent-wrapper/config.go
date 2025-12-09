package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
)

// Config holds CLI configuration
type Config struct {
	Mode          string // "new" or "resume"
	Task          string
	SessionID     string
	WorkDir       string
	ExplicitStdin bool
	Timeout       int
	Backend       string
}

// ParallelConfig defines the JSON schema for parallel execution
type ParallelConfig struct {
	Tasks []TaskSpec `json:"tasks"`
}

// TaskSpec describes an individual task entry in the parallel config
type TaskSpec struct {
	ID           string   `json:"id"`
	Task         string   `json:"task"`
	WorkDir      string   `json:"workdir,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
	SessionID    string   `json:"session_id,omitempty"`
	Mode         string   `json:"-"`
	UseStdin     bool     `json:"-"`
}

// TaskResult captures the execution outcome of a task
type TaskResult struct {
	TaskID    string `json:"task_id"`
	ExitCode  int    `json:"exit_code"`
	Message   string `json:"message"`
	SessionID string `json:"session_id"`
	Error     string `json:"error"`
}

var backendRegistry = map[string]Backend{
	"codex":  CodexBackend{},
	"claude": ClaudeBackend{},
	"gemini": GeminiBackend{},
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

func parseParallelConfig(data []byte) (*ParallelConfig, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("parallel config is empty")
	}

	tasks := strings.Split(string(trimmed), "---TASK---")
	var cfg ParallelConfig
	seen := make(map[string]struct{})

	for _, taskBlock := range tasks {
		taskBlock = strings.TrimSpace(taskBlock)
		if taskBlock == "" {
			continue
		}

		parts := strings.SplitN(taskBlock, "---CONTENT---", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("task block missing ---CONTENT--- separator")
		}

		meta := strings.TrimSpace(parts[0])
		content := strings.TrimSpace(parts[1])

		task := TaskSpec{WorkDir: defaultWorkdir}
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
				task.WorkDir = value
			case "session_id":
				task.SessionID = value
				task.Mode = "resume"
			case "dependencies":
				for _, dep := range strings.Split(value, ",") {
					dep = strings.TrimSpace(dep)
					if dep != "" {
						task.Dependencies = append(task.Dependencies, dep)
					}
				}
			}
		}

		if task.ID == "" {
			return nil, fmt.Errorf("task missing id field")
		}
		if content == "" {
			return nil, fmt.Errorf("task %q missing content", task.ID)
		}
		if _, exists := seen[task.ID]; exists {
			return nil, fmt.Errorf("duplicate task id: %s", task.ID)
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
	filtered := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
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
		}
		filtered = append(filtered, arg)
	}

	if len(filtered) == 0 {
		return nil, fmt.Errorf("task required")
	}
	args = filtered

	cfg := &Config{WorkDir: defaultWorkdir, Backend: backendName}

	if args[0] == "resume" {
		if len(args) < 3 {
			return nil, fmt.Errorf("resume mode requires: resume <session_id> <task>")
		}
		cfg.Mode = "resume"
		cfg.SessionID = args[1]
		cfg.Task = args[2]
		cfg.ExplicitStdin = (args[2] == "-")
		if len(args) > 3 {
			cfg.WorkDir = args[3]
		}
	} else {
		cfg.Mode = "new"
		cfg.Task = args[0]
		cfg.ExplicitStdin = (args[0] == "-")
		if len(args) > 1 {
			cfg.WorkDir = args[1]
		}
	}

	return cfg, nil
}
