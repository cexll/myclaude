package wrapper

import (
	"os"
	"testing"
)

func TestParseArgs_Workdir_OSPaths(t *testing.T) {
	oldArgv := os.Args
	t.Cleanup(func() { os.Args = oldArgv })

	workdirs := []struct {
		name string
		path string
	}{
		{name: "windows drive forward slashes", path: "D:/repo/path"},
		{name: "windows drive backslashes", path: `C:\repo\path`},
		{name: "windows UNC", path: `\\server\share\repo`},
		{name: "unix absolute", path: "/home/user/repo"},
		{name: "relative", path: "./relative/repo"},
	}

	for _, wd := range workdirs {
		t.Run("new mode: "+wd.name, func(t *testing.T) {
			os.Args = []string{"codeagent-wrapper", "task", wd.path}
			cfg, err := parseArgs()
			if err != nil {
				t.Fatalf("parseArgs() error: %v", err)
			}
			if cfg.Mode != "new" || cfg.Task != "task" || cfg.WorkDir != wd.path {
				t.Fatalf("cfg mismatch: got mode=%q task=%q workdir=%q, want mode=%q task=%q workdir=%q", cfg.Mode, cfg.Task, cfg.WorkDir, "new", "task", wd.path)
			}
		})

		t.Run("resume mode: "+wd.name, func(t *testing.T) {
			os.Args = []string{"codeagent-wrapper", "resume", "sid-1", "task", wd.path}
			cfg, err := parseArgs()
			if err != nil {
				t.Fatalf("parseArgs() error: %v", err)
			}
			if cfg.Mode != "resume" || cfg.SessionID != "sid-1" || cfg.Task != "task" || cfg.WorkDir != wd.path {
				t.Fatalf("cfg mismatch: got mode=%q sid=%q task=%q workdir=%q, want mode=%q sid=%q task=%q workdir=%q", cfg.Mode, cfg.SessionID, cfg.Task, cfg.WorkDir, "resume", "sid-1", "task", wd.path)
			}
		})
	}
}
