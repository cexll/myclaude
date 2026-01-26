package backend

import (
	"reflect"
	"testing"

	config "codeagent-wrapper/internal/config"
)

func TestBuildCodexArgs_Workdir_OSPaths(t *testing.T) {
	t.Setenv("CODEX_BYPASS_SANDBOX", "false")

	tests := []struct {
		name    string
		workdir string
	}{
		{name: "windows drive forward slashes", workdir: "D:/repo/path"},
		{name: "windows drive backslashes", workdir: `C:\repo\path`},
		{name: "windows UNC", workdir: `\\server\share\repo`},
		{name: "unix absolute", workdir: "/home/user/repo"},
		{name: "relative", workdir: "./relative/repo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{Mode: "new", WorkDir: tt.workdir}
			got := BuildCodexArgs(cfg, "task")
			want := []string{"e", "--skip-git-repo-check", "-C", tt.workdir, "--json", "task"}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("BuildCodexArgs() = %v, want %v", got, want)
			}
		})
	}

	t.Run("new mode stdin target uses dash", func(t *testing.T) {
		cfg := &config.Config{Mode: "new", WorkDir: `C:\repo\path`}
		got := BuildCodexArgs(cfg, "-")
		want := []string{"e", "--skip-git-repo-check", "-C", `C:\repo\path`, "--json", "-"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("BuildCodexArgs() = %v, want %v", got, want)
		}
	})
}

func TestBuildCodexArgs_ResumeMode_OmitsWorkdir(t *testing.T) {
	t.Setenv("CODEX_BYPASS_SANDBOX", "false")

	cfg := &config.Config{Mode: "resume", SessionID: "sid-123", WorkDir: `C:\repo\path`}
	got := BuildCodexArgs(cfg, "-")
	want := []string{"e", "--skip-git-repo-check", "--json", "resume", "sid-123", "-"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildCodexArgs() = %v, want %v", got, want)
	}
}
