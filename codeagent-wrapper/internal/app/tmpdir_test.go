package wrapper

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestEnsureExecutableTempDir_Override(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ensureExecutableTempDir is no-op on Windows")
	}
	restore := captureTempEnv()
	t.Cleanup(restore)

	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", os.Getenv("HOME"))

	orig := tmpDirExecutableCheckFn
	tmpDirExecutableCheckFn = func(string) (bool, error) { return true, nil }
	t.Cleanup(func() { tmpDirExecutableCheckFn = orig })

	override := filepath.Join(t.TempDir(), "mytmp")
	t.Setenv(tmpDirEnvOverrideKey, override)

	ensureExecutableTempDir()

	if got := os.Getenv("TMPDIR"); got != override {
		t.Fatalf("TMPDIR=%q, want %q", got, override)
	}
	if got := os.Getenv("TMP"); got != override {
		t.Fatalf("TMP=%q, want %q", got, override)
	}
	if got := os.Getenv("TEMP"); got != override {
		t.Fatalf("TEMP=%q, want %q", got, override)
	}
	if st, err := os.Stat(override); err != nil || !st.IsDir() {
		t.Fatalf("override dir not created: stat=%v err=%v", st, err)
	}
}

func TestEnsureExecutableTempDir_FallbackWhenCurrentNotExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ensureExecutableTempDir is no-op on Windows")
	}
	restore := captureTempEnv()
	t.Cleanup(restore)

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	cur := filepath.Join(t.TempDir(), "cur-tmp")
	if err := os.MkdirAll(cur, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", cur)

	fallback := filepath.Join(home, ".codeagent", "tmp")

	orig := tmpDirExecutableCheckFn
	tmpDirExecutableCheckFn = func(dir string) (bool, error) {
		if filepath.Clean(dir) == filepath.Clean(cur) {
			return false, nil
		}
		if filepath.Clean(dir) == filepath.Clean(fallback) {
			return true, nil
		}
		return true, nil
	}
	t.Cleanup(func() { tmpDirExecutableCheckFn = orig })

	ensureExecutableTempDir()

	if got := os.Getenv("TMPDIR"); filepath.Clean(got) != filepath.Clean(fallback) {
		t.Fatalf("TMPDIR=%q, want %q", got, fallback)
	}
	if st, err := os.Stat(fallback); err != nil || !st.IsDir() {
		t.Fatalf("fallback dir not created: stat=%v err=%v", st, err)
	}
}

func captureTempEnv() func() {
	type entry struct {
		set bool
		val string
	}
	snapshot := make(map[string]entry, 3)
	for _, k := range []string{"TMPDIR", "TMP", "TEMP"} {
		v, ok := os.LookupEnv(k)
		snapshot[k] = entry{set: ok, val: v}
	}
	return func() {
		for k, e := range snapshot {
			if !e.set {
				_ = os.Unsetenv(k)
				continue
			}
			_ = os.Setenv(k, e.val)
		}
	}
}
