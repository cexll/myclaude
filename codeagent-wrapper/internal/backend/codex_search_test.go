package backend

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"codeagent-wrapper/internal/config"
)

func freshCodexSearch(t *testing.T) (string, string) {
	t.Helper()
	for _, entry := range os.Environ() {
		key := strings.SplitN(entry, "=", 2)[0]
		if strings.HasPrefix(key, "CODEX_") {
			t.Setenv(key, "")
			if err := os.Unsetenv(key); err != nil {
				t.Fatal(err)
			}
		}
	}
	root := t.TempDir()
	home := filepath.Join(root, "home")
	workdir := filepath.Join(root, "project", "nested")
	for _, dir := range []string{home, workdir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("CODEX_HOME", "")
	t.Setenv("CODEAGENT_PARALLEL_SEARCH", "true")
	return home, workdir
}

func TestCodexParallelSearchPreservesConfiguration(t *testing.T) {
	for _, name := range []string{"config.toml", "config.yaml", "config.yml", "config.json", "managed_config.toml", "requirements.toml", "auth.json"} {
		t.Run(name, func(t *testing.T) {
			home, workdir := freshCodexSearch(t)
			path := filepath.Join(home, ".codex", name)
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				t.Fatal(err)
			}
			const saved = "web_search = \"disabled\"\n"
			if err := os.WriteFile(path, []byte(saved), 0o600); err != nil {
				t.Fatal(err)
			}
			if codexSearchUnconfigured(workdir, filepath.Join(home, "absent-system")) {
				t.Fatal("saved host configuration must suppress the new default")
			}
			if got, err := os.ReadFile(path); err != nil || string(got) != saved {
				t.Fatal("saved host settings changed")
			}
		})
	}
	for _, location := range []string{"project", "ancestor", "system", "custom-home"} {
		t.Run(location, func(t *testing.T) {
			home, workdir := freshCodexSearch(t)
			systemDir := filepath.Join(home, "absent-system")
			var path string
			switch location {
			case "project":
				path = filepath.Join(workdir, ".codex", "config.toml")
			case "ancestor":
				path = filepath.Join(filepath.Dir(workdir), ".codex", "config.toml")
			case "system":
				path = filepath.Join(systemDir, "requirements.toml")
			case "custom-home":
				custom := filepath.Join(home, "custom")
				t.Setenv("CODEX_HOME", custom)
				path = filepath.Join(custom, "config.toml")
			}
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("[mcp_servers.saved]\nurl=\"https://example.com/mcp\"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if codexSearchUnconfigured(workdir, systemDir) {
				t.Fatal("configured installation must keep native behavior")
			}
		})
	}
}

func TestCodexParallelSearchFreshAndUncertainPaths(t *testing.T) {
	home, workdir := freshCodexSearch(t)
	systemDir := filepath.Join(home, "absent-system")
	if !codexSearchUnconfigured(workdir, systemDir) {
		t.Fatal("fresh installation should receive the default")
	}
	// Host-generated session/cache directories do not create a search choice.
	if err := os.MkdirAll(filepath.Join(home, ".codex", "sessions"), 0o700); err != nil {
		t.Fatal(err)
	}
	if !codexSearchUnconfigured(workdir, systemDir) {
		t.Fatal("session persistence must not remove the fresh default on resume")
	}
	t.Setenv("CODEX_UNKNOWN_HOST_CONTROL", "configured")
	if codexSearchUnconfigured(workdir, systemDir) {
		t.Fatal("unknown host configuration must preserve native behavior")
	}
	if err := os.Unsetenv("CODEX_UNKNOWN_HOST_CONTROL"); err != nil {
		t.Fatal(err)
	}
	if codexSearchUnconfigured(filepath.Join(workdir, "missing"), systemDir) {
		t.Fatal("unknown effective directory must preserve native behavior")
	}
	t.Setenv("CODEX_HOME", "relative-home")
	if codexSearchUnconfigured(workdir, systemDir) {
		t.Fatal("ambiguous custom home must preserve native behavior")
	}
	t.Setenv("CODEX_HOME", "")
	if runtime.GOOS != "windows" {
		link := filepath.Join(home, "linked-project")
		if err := os.Symlink(workdir, link); err != nil {
			t.Fatal(err)
		}
		configDir := filepath.Join(filepath.Dir(workdir), ".codex")
		if err := os.Mkdir(configDir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte("web_search = \"disabled\""), 0o600); err != nil {
			t.Fatal(err)
		}
		if codexSearchUnconfigured(link, systemDir) {
			t.Fatal("physical project ancestor settings must be preserved")
		}
	}
}

func TestSupportedCodexVersion(t *testing.T) {
	for _, test := range []struct {
		version string
		want    bool
	}{
		{"codex-cli 0.163.0", true}, {"codex-cli 0.164.1", true}, {"codex-cli 1.0.0", true},
		{"codex-cli 0.162.9", false}, {"codex-cli 0.163.0-alpha.1", false},
		{"codex-cli -1.163.0", false}, {"other 0.163.0", false}, {"codex-cli 0.163", false},
	} {
		if got := supportedCodexVersion(test.version); got != test.want {
			t.Errorf("supportedCodexVersion(%q)=%v, want %v", test.version, got, test.want)
		}
	}
}

func TestCodexParallelSearchLaunchAndOptOut(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("automatic default is limited to Linux")
	}
	if !pathAbsent("/etc/codex") {
		t.Skip("machine has managed Codex settings")
	}
	home, workdir := freshCodexSearch(t)
	bin := filepath.Join(home, "bin")
	if err := os.Mkdir(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "codex"), []byte("#!/bin/sh\nprintf 'codex-cli 0.163.0\\n'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("CODEX_BYPASS_SANDBOX", "false")
	for _, mode := range []string{"new", "resume"} {
		cfg := &config.Config{Mode: mode, SessionID: "existing-conversation", WorkDir: workdir, Model: "saved-model"}
		got := BuildCodexArgs(cfg, "task")
		joined := strings.Join(got, " ")
		for _, value := range []string{parallelSearchURL, `"User-Agent"="myclaude-codeagent-wrapper"`, `web_search="disabled"`, "--model saved-model"} {
			if !strings.Contains(joined, value) {
				t.Fatalf("%s launch missing %q: %v", mode, value, got)
			}
		}
		if strings.Contains(joined, "session_id=") || strings.Contains(joined, "dangerously-bypass") {
			t.Fatal("MCP conversation identity and permissions must remain host-owned")
		}
		t.Setenv("CODEAGENT_PARALLEL_SEARCH", "false")
		native := BuildCodexArgs(cfg, "task")
		if !reflect.DeepEqual(got[9:], native[1:]) {
			t.Fatalf("integration changed other flags: parallel=%v native=%v", got, native)
		}
		t.Setenv("CODEAGENT_PARALLEL_SEARCH", "true")
	}
}
