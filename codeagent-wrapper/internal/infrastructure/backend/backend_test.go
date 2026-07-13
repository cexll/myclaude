package backend

import (
	"testing"

	config "codeagent-wrapper/internal/config"
)

func TestRegistryAndSelect(t *testing.T) {
	reg := Registry()
	if len(reg) == 0 {
		t.Fatalf("Registry() returned empty map")
	}

	b, err := Select("codex")
	if err != nil {
		t.Fatalf("Select(codex): %v", err)
	}
	if b.Name() != "codex" {
		t.Fatalf("backend name = %q, want %q", b.Name(), "codex")
	}
}

func TestBuildCodexArgsAlias(t *testing.T) {
	args := BuildCodexArgs(&config.Config{
		Mode:    "new",
		WorkDir: ".",
	}, "task")
	if len(args) == 0 {
		t.Fatalf("BuildCodexArgs() returned no args")
	}
}

func TestMiniMaxRegistryDelegation(t *testing.T) {
	aliases := []string{
		"minimax",
		"minimax-anthropic-global",
		"minimax-anthropic-cn",
		"minimax-openai-global",
		"minimax-openai-cn",
	}
	for _, alias := range aliases {
		if _, err := Select(alias); err != nil {
			t.Fatalf("Select(%q): %v", alias, err)
		}
	}
}
