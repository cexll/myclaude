package executor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ReadAgentPromptFile(path string, allowOutsideClaudeDir bool) (string, error) {
	raw := strings.TrimSpace(path)
	if raw == "" {
		return "", nil
	}

	expanded := raw
	if raw == "~" || strings.HasPrefix(raw, "~/") || strings.HasPrefix(raw, "~\\") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if raw == "~" {
			expanded = home
		} else {
			expanded = home + raw[1:]
		}
	}

	absPath, err := filepath.Abs(expanded)
	if err != nil {
		return "", err
	}
	absPath = filepath.Clean(absPath)

	home, err := os.UserHomeDir()
	if err != nil {
		if !allowOutsideClaudeDir {
			return "", err
		}
		logWarn(fmt.Sprintf("Failed to resolve home directory for prompt file validation: %v; proceeding without restriction", err))
	} else {
		allowedDirs := []string{
			filepath.Clean(filepath.Join(home, ".claude")),
			filepath.Clean(filepath.Join(home, ".codeagent", "agents")),
		}
		for i := range allowedDirs {
			allowedAbs, err := filepath.Abs(allowedDirs[i])
			if err == nil {
				allowedDirs[i] = filepath.Clean(allowedAbs)
			}
		}

		isWithinDir := func(path, dir string) bool {
			rel, err := filepath.Rel(dir, path)
			if err != nil {
				return false
			}
			rel = filepath.Clean(rel)
			if rel == "." {
				return true
			}
			if rel == ".." {
				return false
			}
			prefix := ".." + string(os.PathSeparator)
			return !strings.HasPrefix(rel, prefix)
		}

		if !allowOutsideClaudeDir {
			withinAllowed := false
			for _, dir := range allowedDirs {
				if isWithinDir(absPath, dir) {
					withinAllowed = true
					break
				}
			}
			if !withinAllowed {
				logWarn(fmt.Sprintf("Refusing to read prompt file outside allowed dirs (%s): %s", strings.Join(allowedDirs, ", "), absPath))
				return "", fmt.Errorf("prompt file must be under ~/.claude or ~/.codeagent/agents")
			}

			resolvedPath, errPath := filepath.EvalSymlinks(absPath)
			if errPath == nil {
				resolvedPath = filepath.Clean(resolvedPath)
				resolvedAllowed := make([]string, 0, len(allowedDirs))
				for _, dir := range allowedDirs {
					resolvedBase, errBase := filepath.EvalSymlinks(dir)
					if errBase != nil {
						continue
					}
					resolvedAllowed = append(resolvedAllowed, filepath.Clean(resolvedBase))
				}
				if len(resolvedAllowed) > 0 {
					withinResolved := false
					for _, dir := range resolvedAllowed {
						if isWithinDir(resolvedPath, dir) {
							withinResolved = true
							break
						}
					}
					if !withinResolved {
						logWarn(fmt.Sprintf("Refusing to read prompt file outside allowed dirs (%s) (resolved): %s", strings.Join(resolvedAllowed, ", "), resolvedPath))
						return "", fmt.Errorf("prompt file must be under ~/.claude or ~/.codeagent/agents")
					}
				}
			}
		} else {
			withinAllowed := false
			for _, dir := range allowedDirs {
				if isWithinDir(absPath, dir) {
					withinAllowed = true
					break
				}
			}
			if !withinAllowed {
				logWarn(fmt.Sprintf("Reading prompt file outside allowed dirs (%s): %s", strings.Join(allowedDirs, ", "), absPath))
			}
		}
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(data), "\r\n"), nil
}

func WrapTaskWithAgentPrompt(prompt string, task string) string {
	return "<agent-prompt>\n" + prompt + "\n</agent-prompt>\n\n" + task
}
