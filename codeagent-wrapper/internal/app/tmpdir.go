package wrapper

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const tmpDirEnvOverrideKey = "CODEAGENT_TMPDIR"

var tmpDirExecutableCheckFn = canExecuteInDir

func ensureExecutableTempDir() {
	// Windows doesn't execute scripts via shebang, and os.TempDir semantics differ.
	if runtime.GOOS == "windows" {
		return
	}

	if override := strings.TrimSpace(os.Getenv(tmpDirEnvOverrideKey)); override != "" {
		if resolved, err := resolvePathWithTilde(override); err == nil {
			if err := os.MkdirAll(resolved, 0o700); err == nil {
				if ok, _ := tmpDirExecutableCheckFn(resolved); ok {
					setTempEnv(resolved)
					return
				}
			}
		}
		// Invalid override should not block execution; fall back to default behavior.
	}

	current := currentTempDirFromEnv()
	if current == "" {
		current = "/tmp"
	}

	ok, _ := tmpDirExecutableCheckFn(current)
	if ok {
		return
	}

	fallback := defaultFallbackTempDir()
	if fallback == "" {
		return
	}
	if err := os.MkdirAll(fallback, 0o700); err != nil {
		return
	}
	if ok, _ := tmpDirExecutableCheckFn(fallback); !ok {
		return
	}

	setTempEnv(fallback)
	fmt.Fprintf(os.Stderr, "INFO: temp dir is not executable; set TMPDIR=%s\n", fallback)
}

func setTempEnv(dir string) {
	_ = os.Setenv("TMPDIR", dir)
	_ = os.Setenv("TMP", dir)
	_ = os.Setenv("TEMP", dir)
}

func defaultFallbackTempDir() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return ""
	}
	return filepath.Clean(filepath.Join(home, ".codeagent", "tmp"))
}

func currentTempDirFromEnv() string {
	for _, k := range []string{"TMPDIR", "TMP", "TEMP"} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

func resolvePathWithTilde(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", errors.New("empty path")
	}

	if p == "~" || strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~\\") {
		home, err := os.UserHomeDir()
		if err != nil || strings.TrimSpace(home) == "" {
			if err == nil {
				err = errors.New("empty home directory")
			}
			return "", fmt.Errorf("resolve ~: %w", err)
		}
		if p == "~" {
			return home, nil
		}
		return filepath.Clean(home + p[1:]), nil
	}

	return filepath.Clean(p), nil
}

func canExecuteInDir(dir string) (bool, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return false, errors.New("empty dir")
	}

	f, err := os.CreateTemp(dir, "codeagent-tmp-exec-*")
	if err != nil {
		return false, err
	}
	path := f.Name()
	defer func() { _ = os.Remove(path) }()

	if _, err := f.WriteString("#!/bin/sh\nexit 0\n"); err != nil {
		_ = f.Close()
		return false, err
	}
	if err := f.Close(); err != nil {
		return false, err
	}
	if err := os.Chmod(path, 0o700); err != nil {
		return false, err
	}

	if err := exec.Command(path).Run(); err != nil {
		return false, err
	}
	return true, nil
}
