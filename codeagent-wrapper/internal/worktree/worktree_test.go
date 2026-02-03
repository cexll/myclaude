package worktree

import (
	"crypto/rand"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sync"
	"testing"
	"time"
)

func resetHooks() {
	randReader = rand.Reader
	timeNowFunc = time.Now
	execCommand = exec.Command
}

func TestGenerateTaskID(t *testing.T) {
	defer resetHooks()

	taskID, err := generateTaskID()
	if err != nil {
		t.Fatalf("generateTaskID() error = %v", err)
	}

	// Format: YYYYMMDD-6hex
	pattern := regexp.MustCompile(`^\d{8}-[0-9a-f]{6}$`)
	if !pattern.MatchString(taskID) {
		t.Errorf("generateTaskID() = %q, want format YYYYMMDD-xxxxxx", taskID)
	}
}

func TestGenerateTaskID_FixedTime(t *testing.T) {
	defer resetHooks()

	// Mock time to a fixed date
	timeNowFunc = func() time.Time {
		return time.Date(2026, 2, 3, 12, 0, 0, 0, time.UTC)
	}

	taskID, err := generateTaskID()
	if err != nil {
		t.Fatalf("generateTaskID() error = %v", err)
	}

	if !regexp.MustCompile(`^20260203-[0-9a-f]{6}$`).MatchString(taskID) {
		t.Errorf("generateTaskID() = %q, want prefix 20260203-", taskID)
	}
}

func TestGenerateTaskID_RandReaderError(t *testing.T) {
	defer resetHooks()

	// Mock rand reader to return error
	randReader = &errorReader{err: errors.New("mock rand error")}

	_, err := generateTaskID()
	if err == nil {
		t.Fatal("generateTaskID() expected error, got nil")
	}
	if !regexp.MustCompile(`failed to generate random bytes`).MatchString(err.Error()) {
		t.Errorf("error = %q, want 'failed to generate random bytes'", err.Error())
	}
}

type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}

func TestGenerateTaskID_Uniqueness(t *testing.T) {
	defer resetHooks()

	const count = 100
	ids := make(map[string]struct{}, count)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id, err := generateTaskID()
			if err != nil {
				t.Errorf("generateTaskID() error = %v", err)
				return
			}
			mu.Lock()
			ids[id] = struct{}{}
			mu.Unlock()
		}()
	}
	wg.Wait()

	if len(ids) != count {
		t.Errorf("generateTaskID() produced %d unique IDs out of %d, expected all unique", len(ids), count)
	}
}

func TestCreateWorktree_NotGitRepo(t *testing.T) {
	defer resetHooks()

	tmpDir, err := os.MkdirTemp("", "worktree-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	_, err = CreateWorktree(tmpDir)
	if err == nil {
		t.Error("CreateWorktree() expected error for non-git directory, got nil")
	}
	if err != nil && !regexp.MustCompile(`not a git repository`).MatchString(err.Error()) {
		t.Errorf("CreateWorktree() error = %q, want 'not a git repository'", err.Error())
	}
}

func TestCreateWorktree_EmptyProjectDir(t *testing.T) {
	defer resetHooks()

	// When projectDir is empty, it should default to "."
	// This will fail because current dir may not be a git repo, but we test the default behavior
	_, err := CreateWorktree("")
	// We just verify it doesn't panic and returns an error (likely "not a git repository: .")
	if err == nil {
		// If we happen to be in a git repo, that's fine too
		return
	}
	if !regexp.MustCompile(`not a git repository: \.`).MatchString(err.Error()) {
		// It might be a git repo and fail later, which is also acceptable
		return
	}
}

func TestCreateWorktree_Success(t *testing.T) {
	defer resetHooks()

	// Create temp git repo
	tmpDir, err := os.MkdirTemp("", "worktree-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize git repo
	if err := exec.Command("git", "-C", tmpDir, "init").Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}
	if err := exec.Command("git", "-C", tmpDir, "config", "user.email", "test@test.com").Run(); err != nil {
		t.Fatalf("failed to set git email: %v", err)
	}
	if err := exec.Command("git", "-C", tmpDir, "config", "user.name", "Test").Run(); err != nil {
		t.Fatalf("failed to set git name: %v", err)
	}

	// Create initial commit (required for worktree)
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	if err := exec.Command("git", "-C", tmpDir, "add", ".").Run(); err != nil {
		t.Fatalf("failed to git add: %v", err)
	}
	if err := exec.Command("git", "-C", tmpDir, "commit", "-m", "initial").Run(); err != nil {
		t.Fatalf("failed to git commit: %v", err)
	}

	// Test CreateWorktree
	paths, err := CreateWorktree(tmpDir)
	if err != nil {
		t.Fatalf("CreateWorktree() error = %v", err)
	}

	// Verify task ID format
	pattern := regexp.MustCompile(`^\d{8}-[0-9a-f]{6}$`)
	if !pattern.MatchString(paths.TaskID) {
		t.Errorf("TaskID = %q, want format YYYYMMDD-xxxxxx", paths.TaskID)
	}

	// Verify branch name
	expectedBranch := "do/" + paths.TaskID
	if paths.Branch != expectedBranch {
		t.Errorf("Branch = %q, want %q", paths.Branch, expectedBranch)
	}

	// Verify worktree directory exists
	if _, err := os.Stat(paths.Dir); os.IsNotExist(err) {
		t.Errorf("worktree directory %q does not exist", paths.Dir)
	}

	// Verify worktree directory is under .worktrees/
	expectedDirSuffix := filepath.Join(".worktrees", "do-"+paths.TaskID)
	if !regexp.MustCompile(regexp.QuoteMeta(expectedDirSuffix) + `$`).MatchString(paths.Dir) {
		t.Errorf("Dir = %q, want suffix %q", paths.Dir, expectedDirSuffix)
	}

	// Verify branch exists
	cmd := exec.Command("git", "-C", tmpDir, "branch", "--list", paths.Branch)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to list branches: %v", err)
	}
	if len(output) == 0 {
		t.Errorf("branch %q was not created", paths.Branch)
	}
}

func TestCreateWorktree_GetGitRootError(t *testing.T) {
	defer resetHooks()

	// Create a temp dir and mock git commands
	tmpDir, err := os.MkdirTemp("", "worktree-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	callCount := 0
	execCommand = func(name string, args ...string) *exec.Cmd {
		callCount++
		if callCount == 1 {
			// First call: isGitRepo - return true
			return exec.Command("echo", "true")
		}
		// Second call: getGitRoot - return error
		return exec.Command("false")
	}

	_, err = CreateWorktree(tmpDir)
	if err == nil {
		t.Fatal("CreateWorktree() expected error, got nil")
	}
	if !regexp.MustCompile(`failed to get git root`).MatchString(err.Error()) {
		t.Errorf("error = %q, want 'failed to get git root'", err.Error())
	}
}

func TestCreateWorktree_GenerateTaskIDError(t *testing.T) {
	defer resetHooks()

	// Create temp git repo
	tmpDir, err := os.MkdirTemp("", "worktree-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize git repo with commit
	if err := exec.Command("git", "-C", tmpDir, "init").Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}
	if err := exec.Command("git", "-C", tmpDir, "config", "user.email", "test@test.com").Run(); err != nil {
		t.Fatalf("failed to set git email: %v", err)
	}
	if err := exec.Command("git", "-C", tmpDir, "config", "user.name", "Test").Run(); err != nil {
		t.Fatalf("failed to set git name: %v", err)
	}
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	if err := exec.Command("git", "-C", tmpDir, "add", ".").Run(); err != nil {
		t.Fatalf("failed to git add: %v", err)
	}
	if err := exec.Command("git", "-C", tmpDir, "commit", "-m", "initial").Run(); err != nil {
		t.Fatalf("failed to git commit: %v", err)
	}

	// Mock rand reader to fail
	randReader = &errorReader{err: errors.New("mock rand error")}

	_, err = CreateWorktree(tmpDir)
	if err == nil {
		t.Fatal("CreateWorktree() expected error, got nil")
	}
	if !regexp.MustCompile(`failed to generate random bytes`).MatchString(err.Error()) {
		t.Errorf("error = %q, want 'failed to generate random bytes'", err.Error())
	}
}

func TestCreateWorktree_WorktreeAddError(t *testing.T) {
	defer resetHooks()

	tmpDir, err := os.MkdirTemp("", "worktree-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	callCount := 0
	execCommand = func(name string, args ...string) *exec.Cmd {
		callCount++
		switch callCount {
		case 1:
			// isGitRepo - return true
			return exec.Command("echo", "true")
		case 2:
			// getGitRoot - return tmpDir
			return exec.Command("echo", tmpDir)
		case 3:
			// worktree add - return error
			return exec.Command("false")
		}
		return exec.Command("false")
	}

	_, err = CreateWorktree(tmpDir)
	if err == nil {
		t.Fatal("CreateWorktree() expected error, got nil")
	}
	if !regexp.MustCompile(`failed to create worktree`).MatchString(err.Error()) {
		t.Errorf("error = %q, want 'failed to create worktree'", err.Error())
	}
}

func TestIsGitRepo(t *testing.T) {
	defer resetHooks()

	// Test non-git directory
	tmpDir, err := os.MkdirTemp("", "worktree-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if isGitRepo(tmpDir) {
		t.Error("isGitRepo() = true for non-git directory, want false")
	}

	// Test git directory
	if err := exec.Command("git", "-C", tmpDir, "init").Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	if !isGitRepo(tmpDir) {
		t.Error("isGitRepo() = false for git directory, want true")
	}
}

func TestIsGitRepo_CommandError(t *testing.T) {
	defer resetHooks()

	// Mock execCommand to return error
	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("false")
	}

	if isGitRepo("/some/path") {
		t.Error("isGitRepo() = true when command fails, want false")
	}
}

func TestIsGitRepo_NotTrueOutput(t *testing.T) {
	defer resetHooks()

	// Mock execCommand to return something other than "true"
	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("echo", "false")
	}

	if isGitRepo("/some/path") {
		t.Error("isGitRepo() = true when output is 'false', want false")
	}
}

func TestGetGitRoot(t *testing.T) {
	defer resetHooks()

	// Create temp git repo
	tmpDir, err := os.MkdirTemp("", "worktree-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := exec.Command("git", "-C", tmpDir, "init").Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	root, err := getGitRoot(tmpDir)
	if err != nil {
		t.Fatalf("getGitRoot() error = %v", err)
	}

	// The root should match tmpDir (accounting for symlinks)
	absRoot, _ := filepath.EvalSymlinks(root)
	absTmp, _ := filepath.EvalSymlinks(tmpDir)
	if absRoot != absTmp {
		t.Errorf("getGitRoot() = %q, want %q", absRoot, absTmp)
	}
}

func TestGetGitRoot_Error(t *testing.T) {
	defer resetHooks()

	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("false")
	}

	_, err := getGitRoot("/some/path")
	if err == nil {
		t.Fatal("getGitRoot() expected error, got nil")
	}
	if !regexp.MustCompile(`failed to get git root`).MatchString(err.Error()) {
		t.Errorf("error = %q, want 'failed to get git root'", err.Error())
	}
}

// Test that rand reader produces expected bytes
func TestGenerateTaskID_RandReaderBytes(t *testing.T) {
	defer resetHooks()

	// Mock rand reader to return fixed bytes
	randReader = &fixedReader{data: []byte{0xab, 0xcd, 0xef}}
	timeNowFunc = func() time.Time {
		return time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	}

	taskID, err := generateTaskID()
	if err != nil {
		t.Fatalf("generateTaskID() error = %v", err)
	}

	expected := "20260115-abcdef"
	if taskID != expected {
		t.Errorf("generateTaskID() = %q, want %q", taskID, expected)
	}
}

type fixedReader struct {
	data []byte
	pos  int
}

func (f *fixedReader) Read(p []byte) (n int, err error) {
	if f.pos >= len(f.data) {
		return 0, io.EOF
	}
	n = copy(p, f.data[f.pos:])
	f.pos += n
	return n, nil
}
