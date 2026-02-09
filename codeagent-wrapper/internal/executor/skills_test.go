package executor

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// setTestHome overrides the home directory for both Unix (HOME) and Windows (USERPROFILE).
func setTestHome(t *testing.T, home string) {
	t.Helper()
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
	}
}

// --- helper: create a temp skill dir with SKILL.md ---

func createTempSkill(t *testing.T, name, content string) string {
	t.Helper()
	home := t.TempDir()
	skillDir := filepath.Join(home, ".claude", "skills", name)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return home
}

// --- ParseParallelConfig skills parsing tests ---

func TestParseParallelConfig_SkillsField(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		taskIdx        int
		expectedSkills []string
	}{
		{
			name: "single skill",
			input: `---TASK---
id: t1
workdir: .
skills: golang-base-practices
---CONTENT---
Do something.
`,
			taskIdx:        0,
			expectedSkills: []string{"golang-base-practices"},
		},
		{
			name: "multiple comma-separated skills",
			input: `---TASK---
id: t1
workdir: .
skills: golang-base-practices, vercel-react-best-practices
---CONTENT---
Do something.
`,
			taskIdx:        0,
			expectedSkills: []string{"golang-base-practices", "vercel-react-best-practices"},
		},
		{
			name: "no skills field",
			input: `---TASK---
id: t1
workdir: .
---CONTENT---
Do something.
`,
			taskIdx:        0,
			expectedSkills: nil,
		},
		{
			name: "empty skills value",
			input: `---TASK---
id: t1
workdir: .
skills:
---CONTENT---
Do something.
`,
			taskIdx:        0,
			expectedSkills: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ParseParallelConfig([]byte(tt.input))
			if err != nil {
				t.Fatalf("ParseParallelConfig error: %v", err)
			}
			got := cfg.Tasks[tt.taskIdx].Skills
			if len(got) != len(tt.expectedSkills) {
				t.Fatalf("skills: got %v, want %v", got, tt.expectedSkills)
			}
			for i := range got {
				if got[i] != tt.expectedSkills[i] {
					t.Errorf("skills[%d]: got %q, want %q", i, got[i], tt.expectedSkills[i])
				}
			}
		})
	}
}

// --- stripYAMLFrontmatter tests ---

func TestStripYAMLFrontmatter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "with frontmatter",
			input:    "---\nname: test\ndescription: foo\n---\n\n# Body\nContent here.",
			expected: "# Body\nContent here.",
		},
		{
			name:     "no frontmatter",
			input:    "# Just a body\nNo frontmatter.",
			expected: "# Just a body\nNo frontmatter.",
		},
		{
			name:     "empty",
			input:    "",
			expected: "",
		},
		{
			name:     "only frontmatter",
			input:    "---\nname: test\n---",
			expected: "",
		},
		{
			name:     "frontmatter with allowed-tools",
			input:    "---\nname: do\nallowed-tools: [\"Bash\"]\n---\n\n# Skill content",
			expected: "# Skill content",
		},
		{
			name:     "CRLF line endings",
			input:    "---\r\nname: test\r\n---\r\n\r\n# Body\r\nContent.",
			expected: "# Body\nContent.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripYAMLFrontmatter(tt.input)
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

// --- DetectProjectSkills tests ---

func TestDetectProjectSkills_GoProject(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0644)

	skills := DetectProjectSkills(tmpDir)
	// Result depends on whether golang-base-practices is installed locally
	t.Logf("detected skills for Go project: %v", skills)
}

func TestDetectProjectSkills_NoFingerprints(t *testing.T) {
	tmpDir := t.TempDir()
	skills := DetectProjectSkills(tmpDir)
	if len(skills) != 0 {
		t.Errorf("expected no skills for empty dir, got %v", skills)
	}
}

func TestDetectProjectSkills_FullStack(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(`{"name":"test"}`), 0644)

	skills := DetectProjectSkills(tmpDir)
	t.Logf("detected skills for fullstack project: %v", skills)
	seen := make(map[string]bool)
	for _, s := range skills {
		if seen[s] {
			t.Errorf("duplicate skill detected: %s", s)
		}
		seen[s] = true
	}
}

func TestDetectProjectSkills_NonexistentDir(t *testing.T) {
	skills := DetectProjectSkills("/nonexistent/path/xyz")
	if len(skills) != 0 {
		t.Errorf("expected no skills for nonexistent dir, got %v", skills)
	}
}

// --- ResolveSkillContent tests (CI-friendly with temp dirs) ---

func TestResolveSkillContent_ValidSkill(t *testing.T) {
	home := createTempSkill(t, "test-skill", "---\nname: test\n---\n\n# Test Skill\nBest practices here.")
	setTestHome(t, home)

	result := ResolveSkillContent([]string{"test-skill"}, 0)
	if result == "" {
		t.Fatal("expected non-empty content")
	}
	if !strings.Contains(result, `<skill name="test-skill">`) {
		t.Error("missing opening <skill> tag")
	}
	if !strings.Contains(result, "</skill>") {
		t.Error("missing closing </skill> tag")
	}
	if !strings.Contains(result, "# Test Skill") {
		t.Error("missing skill body content")
	}
	if strings.Contains(result, "name: test") {
		t.Error("frontmatter was not stripped")
	}
}

func TestResolveSkillContent_NonexistentSkill(t *testing.T) {
	home := t.TempDir()
	setTestHome(t, home)

	result := ResolveSkillContent([]string{"nonexistent-skill-xyz"}, 0)
	if result != "" {
		t.Errorf("expected empty for nonexistent skill, got %d bytes", len(result))
	}
}

func TestResolveSkillContent_Empty(t *testing.T) {
	if result := ResolveSkillContent(nil, 0); result != "" {
		t.Errorf("expected empty for nil, got %q", result)
	}
	if result := ResolveSkillContent([]string{}, 0); result != "" {
		t.Errorf("expected empty for empty, got %q", result)
	}
}

func TestResolveSkillContent_Budget(t *testing.T) {
	longBody := strings.Repeat("x", 500)
	home := createTempSkill(t, "big-skill", "---\nname: big\n---\n\n"+longBody)
	setTestHome(t, home)

	result := ResolveSkillContent([]string{"big-skill"}, 200)
	if result == "" {
		t.Fatal("expected non-empty even with small budget")
	}
	if len(result) > 200 {
		t.Errorf("result %d bytes exceeds budget 200", len(result))
	}
	t.Logf("budget=200, result=%d bytes", len(result))
}

func TestResolveSkillContent_MultipleSkills(t *testing.T) {
	home := t.TempDir()
	for _, name := range []string{"skill-a", "skill-b"} {
		skillDir := filepath.Join(home, ".claude", "skills", name)
		os.MkdirAll(skillDir, 0755)
		os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# "+name+"\nContent."), 0644)
	}
	setTestHome(t, home)

	result := ResolveSkillContent([]string{"skill-a", "skill-b"}, 0)
	if result == "" {
		t.Fatal("expected non-empty for multiple skills")
	}
	if !strings.Contains(result, `<skill name="skill-a">`) {
		t.Error("missing skill-a tag")
	}
	if !strings.Contains(result, `<skill name="skill-b">`) {
		t.Error("missing skill-b tag")
	}
}

func TestResolveSkillContent_PathTraversal(t *testing.T) {
	home := t.TempDir()
	setTestHome(t, home)

	result := ResolveSkillContent([]string{"../../../etc/passwd"}, 0)
	if result != "" {
		t.Errorf("expected empty for path traversal name, got %d bytes", len(result))
	}
}

func TestResolveSkillContent_InvalidNames(t *testing.T) {
	home := t.TempDir()
	setTestHome(t, home)

	tests := []string{"../bad", "foo/bar", "skill name", "skill.name", "a b"}
	for _, name := range tests {
		result := ResolveSkillContent([]string{name}, 0)
		if result != "" {
			t.Errorf("expected empty for invalid name %q, got %d bytes", name, len(result))
		}
	}
}

func TestResolveSkillContent_ValidNamePattern(t *testing.T) {
	if !validSkillName.MatchString("golang-base-practices") {
		t.Error("golang-base-practices should be valid")
	}
	if !validSkillName.MatchString("my_skill_v2") {
		t.Error("my_skill_v2 should be valid")
	}
	if validSkillName.MatchString("../bad") {
		t.Error("../bad should be invalid")
	}
	if validSkillName.MatchString("") {
		t.Error("empty should be invalid")
	}
}

// --- Integration: skill injection format test ---

func TestSkillInjectionFormat(t *testing.T) {
	home := createTempSkill(t, "test-go", "---\nname: go\n---\n\n# Go Best Practices\nUse gofmt.")
	setTestHome(t, home)

	taskText := "Implement the feature."
	content := ResolveSkillContent([]string{"test-go"}, 0)
	injected := taskText + "\n\n# Domain Best Practices\n\n" + content

	if !strings.Contains(injected, "Implement the feature.") {
		t.Error("original task text lost")
	}
	if !strings.Contains(injected, "# Domain Best Practices") {
		t.Error("missing section header")
	}
	if !strings.Contains(injected, `<skill name="test-go">`) {
		t.Error("missing <skill> tag")
	}
	if !strings.Contains(injected, "Use gofmt.") {
		t.Error("missing skill body")
	}
}
