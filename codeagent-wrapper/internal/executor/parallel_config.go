package executor

import (
	"bytes"
	"fmt"
	"strings"

	config "codeagent-wrapper/internal/config"
)

func ParseParallelConfig(data []byte) (*ParallelConfig, error) {
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
			case "skip_permissions", "skip-permissions":
				if value == "" {
					task.SkipPermissions = true
					continue
				}
				task.SkipPermissions = config.ParseBoolFlag(value, false)
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
			if err := config.ValidateAgentName(task.Agent); err != nil {
				return nil, fmt.Errorf("task block #%d invalid agent name: %w", taskIndex, err)
			}
			backend, model, promptFile, reasoning, _, _, _ := config.ResolveAgentConfig(task.Agent)
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
