package wrapper

import (
	"strings"
	"testing"

	config "codeagent-wrapper/internal/config"
	backend "codeagent-wrapper/internal/infrastructure/backend"
)

func TestConfigureSingleModeBackend_DefaultBackendDoesNotOverwriteInjectedCommand(t *testing.T) {
	defer resetTestHooks()

	cfg := &Config{Backend: defaultBackendName}
	codexCommand = "injected-cmd"
	buildCodexArgsFn = func(cfg *Config, targetArg string) []string { return []string{"injected", targetArg} }

	restore := selectBackendFn
	selectBackendFn = func(name string) (backend.Backend, error) {
		return backend.CodexBackend{}, nil
	}
	t.Cleanup(func() { selectBackendFn = restore })

	if _, err := configureSingleModeBackend(cfg); err != nil {
		t.Fatalf("configureSingleModeBackend() error = %v", err)
	}
	if codexCommand != "injected-cmd" {
		t.Fatalf("codexCommand = %q, want injected command preserved", codexCommand)
	}
	if got := strings.Join(buildCodexArgsFn(cfg, "x"), " "); got != "injected x" {
		t.Fatalf("buildCodexArgsFn changed unexpectedly: %q", got)
	}
}

func TestConfigureSingleModeBackend_NonDefaultBackendOverridesCommand(t *testing.T) {
	defer resetTestHooks()

	cfg := &config.Config{Backend: "claude"}
	codexCommand = "injected-cmd"
	buildCodexArgsFn = func(cfg *Config, targetArg string) []string { return []string{"injected", targetArg} }

	restore := selectBackendFn
	selectBackendFn = func(name string) (backend.Backend, error) {
		return backend.ClaudeBackend{}, nil
	}
	t.Cleanup(func() { selectBackendFn = restore })

	selected, err := configureSingleModeBackend(cfg)
	if err != nil {
		t.Fatalf("configureSingleModeBackend() error = %v", err)
	}
	if selected.Name() != "claude" || cfg.Backend != "claude" {
		t.Fatalf("selected=%q cfg.Backend=%q", selected.Name(), cfg.Backend)
	}
	if codexCommand != selected.Command() {
		t.Fatalf("codexCommand = %q, want %q", codexCommand, selected.Command())
	}
}

func TestRequireSingleModeLogger_ErrorWhenMissing(t *testing.T) {
	defer resetTestHooks()

	if logger := activeLogger(); logger != nil {
		t.Fatalf("expected no active logger")
	}
	if _, err := requireSingleModeLogger(); err == nil || !strings.Contains(err.Error(), "logger is not initialized") {
		t.Fatalf("requireSingleModeLogger() error = %v", err)
	}
}
