package executor

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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

// techSkillMap maps file-existence fingerprints to skill names.
var techSkillMap = []struct {
	Files  []string // any of these files → this tech
	Skills []string
}{
	{Files: []string{"go.mod", "go.sum"}, Skills: []string{"golang-base-practices"}},
	{Files: []string{"Cargo.toml"}, Skills: []string{"rust-best-practices"}},
	{Files: []string{"pyproject.toml", "setup.py", "requirements.txt", "Pipfile"}, Skills: []string{"python-best-practices"}},
	{Files: []string{"package.json"}, Skills: []string{"vercel-react-best-practices", "frontend-design"}},
	{Files: []string{"vue.config.js", "vite.config.ts", "nuxt.config.ts"}, Skills: []string{"vue-web-app"}},
}

// DetectProjectSkills scans workDir for tech-stack fingerprints and returns
// skill names that are both detected and installed at ~/.claude/skills/{name}/SKILL.md.
func DetectProjectSkills(workDir string) []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	var detected []string
	seen := make(map[string]bool)
	for _, entry := range techSkillMap {
		for _, f := range entry.Files {
			if _, err := os.Stat(filepath.Join(workDir, f)); err == nil {
				for _, skill := range entry.Skills {
					if seen[skill] {
						continue
					}
					skillPath := filepath.Join(home, ".claude", "skills", skill, "SKILL.md")
					if _, err := os.Stat(skillPath); err == nil {
						detected = append(detected, skill)
						seen[skill] = true
					}
				}
				break // one matching file is enough for this entry
			}
		}
	}
	return detected
}

const defaultSkillBudget = 16000 // chars, ~4K tokens

// validSkillName ensures skill names contain only safe characters to prevent path traversal
var validSkillName = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// ResolveSkillContent reads SKILL.md files for the given skill names,
// strips YAML frontmatter, wraps each in <skill> tags, and enforces a
// character budget to prevent context bloat.
func ResolveSkillContent(skills []string, maxBudget int) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	if maxBudget <= 0 {
		maxBudget = defaultSkillBudget
	}
	var sections []string
	remaining := maxBudget
	for _, name := range skills {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if !validSkillName.MatchString(name) {
			logWarn(fmt.Sprintf("skill %q: invalid name (must contain only [a-zA-Z0-9_-]), skipping", name))
			continue
		}
		path := filepath.Join(home, ".claude", "skills", name, "SKILL.md")
		data, err := os.ReadFile(path)
		if err != nil || len(data) == 0 {
			logWarn(fmt.Sprintf("skill %q: SKILL.md not found or empty, skipping", name))
			continue
		}
		body := stripYAMLFrontmatter(strings.TrimSpace(string(data)))
		tagOverhead := len("<skill name=\"\">") + len(name) + len("\n") + len("\n</skill>")
		bodyBudget := remaining - tagOverhead
		if bodyBudget <= 0 {
			logWarn(fmt.Sprintf("skill %q: skipped, insufficient budget for tags", name))
			break
		}
		if len(body) > bodyBudget {
			logWarn(fmt.Sprintf("skill %q: truncated from %d to %d chars (budget)", name, len(body), bodyBudget))
			body = body[:bodyBudget]
		}
		remaining -= len(body) + tagOverhead
		sections = append(sections, "<skill name=\""+name+"\">\n"+body+"\n</skill>")
		if remaining <= 0 {
			break
		}
	}
	if len(sections) == 0 {
		return ""
	}
	return strings.Join(sections, "\n\n")
}

func stripYAMLFrontmatter(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	if !strings.HasPrefix(s, "---") {
		return s
	}
	idx := strings.Index(s[3:], "\n---")
	if idx < 0 {
		return s
	}
	result := s[3+idx+4:]
	if len(result) > 0 && result[0] == '\n' {
		result = result[1:]
	}
	return strings.TrimSpace(result)
}
