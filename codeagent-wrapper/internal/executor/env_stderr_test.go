package executor

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	config "codeagent-wrapper/internal/config"
)

type fakeCmd struct {
	env map[string]string
}

func (f *fakeCmd) Start() error { return nil }
func (f *fakeCmd) Wait() error  { return nil }
func (f *fakeCmd) StdoutPipe() (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}
func (f *fakeCmd) StderrPipe() (io.ReadCloser, error) {
	return nil, errors.New("fake stderr pipe error")
}
func (f *fakeCmd) StdinPipe() (io.WriteCloser, error) {
	return nil, errors.New("fake stdin pipe error")
}
func (f *fakeCmd) SetStderr(io.Writer) {}
func (f *fakeCmd) SetDir(string)       {}
func (f *fakeCmd) SetEnv(env map[string]string) {
	if len(env) == 0 {
		return
	}
	if f.env == nil {
		f.env = make(map[string]string, len(env))
	}
	for k, v := range env {
		f.env[k] = v
	}
}
func (f *fakeCmd) Process() processHandle { return nil }
func (f *fakeCmd) UnsetEnv(keys ...string) {
	for _, k := range keys {
		delete(f.env, k)
	}
}

func TestEnvInjection_LogsToStderrAndMasksKey(t *testing.T) {
	// Arrange ~/.codeagent/models.json via HOME override.
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".codeagent")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	const baseURL = "https://api.minimaxi.com/anthropic"
	const apiKey = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.test"
	models := `{
  "agents": {
    "explore": {
      "backend": "claude",
      "model": "MiniMax-M2.1",
      "base_url": "` + baseURL + `",
      "api_key": "` + apiKey + `"
    }
  }
}`
	if err := os.WriteFile(filepath.Join(configDir, "models.json"), []byte(models), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	config.ResetModelsConfigCacheForTest()
	defer config.ResetModelsConfigCacheForTest()

	// Capture stderr (RunCodexTaskWithContext prints env injection lines there).
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldStderr := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	readDone := make(chan string, 1)
	go func() {
		defer r.Close()
		b, _ := io.ReadAll(r)
		readDone <- string(b)
	}()

	var cmd *fakeCmd
	restoreRunner := SetNewCommandRunner(func(ctx context.Context, name string, args ...string) CommandRunner {
		cmd = &fakeCmd{}
		return cmd
	})
	defer restoreRunner()

	// Act: force an early return right after env injection by making StderrPipe fail.
	_ = RunCodexTaskWithContext(
		context.Background(),
		TaskSpec{Task: "hi", WorkDir: ".", Backend: "claude", Agent: "explore"},
		nil,
		"claude",
		nil,
		nil,
		false,
		false,
		1,
	)

	_ = w.Close()
	got := <-readDone

	// Assert: env was injected into the command and logging is present with masking.
	if cmd == nil || cmd.env == nil {
		t.Fatalf("expected cmd env to be set, got cmd=%v env=%v", cmd, nil)
	}
	if cmd.env["ANTHROPIC_BASE_URL"] != baseURL {
		t.Fatalf("ANTHROPIC_BASE_URL=%q, want %q", cmd.env["ANTHROPIC_BASE_URL"], baseURL)
	}
	if cmd.env["ANTHROPIC_API_KEY"] != apiKey {
		t.Fatalf("ANTHROPIC_API_KEY=%q, want %q", cmd.env["ANTHROPIC_API_KEY"], apiKey)
	}

	if !strings.Contains(got, "Env: ANTHROPIC_BASE_URL="+baseURL) {
		t.Fatalf("stderr missing base URL env log; stderr=%q", got)
	}
	if !strings.Contains(got, "Env: ANTHROPIC_API_KEY=eyJh****test") {
		t.Fatalf("stderr missing masked API key log; stderr=%q", got)
	}
}
