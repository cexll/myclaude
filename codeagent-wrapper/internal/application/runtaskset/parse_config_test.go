package runtaskset

import "testing"

func TestParseConfig_SkillsField(t *testing.T) {
	cfg, err := ParseConfig([]byte(`---TASK---
id: t1
workdir: .
skills: golang-base-practices, vercel-react-best-practices
---CONTENT---
Do something.
`))
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if len(cfg.Tasks) != 1 {
		t.Fatalf("tasks = %d, want 1", len(cfg.Tasks))
	}
	got := cfg.Tasks[0].Skills
	if len(got) != 2 || got[0] != "golang-base-practices" || got[1] != "vercel-react-best-practices" {
		t.Fatalf("skills = %#v, want parsed skills", got)
	}
}

func TestParseConfig_InvalidWorkdirDashRejected(t *testing.T) {
	_, err := ParseConfig([]byte(`---TASK---
id: t1
workdir: -
---CONTENT---
Do something.
`))
	if err == nil {
		t.Fatal("ParseConfig() expected error for invalid workdir")
	}
}
