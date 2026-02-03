package worktree

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Paths contains worktree information
type Paths struct {
	Dir    string // .worktrees/do-{task_id}/
	Branch string // do/{task_id}
	TaskID string // auto-generated task_id
}

// Hook points for testing
var (
	randReader  io.Reader = rand.Reader
	timeNowFunc           = time.Now
	execCommand           = exec.Command
)

// generateTaskID creates a unique task ID in format: YYYYMMDD-{6 hex chars}
func generateTaskID() (string, error) {
	bytes := make([]byte, 3)
	if _, err := io.ReadFull(randReader, bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	date := timeNowFunc().Format("20060102")
	return fmt.Sprintf("%s-%s", date, hex.EncodeToString(bytes)), nil
}

// isGitRepo checks if the given directory is inside a git repository
func isGitRepo(dir string) bool {
	cmd := execCommand("git", "-C", dir, "rev-parse", "--is-inside-work-tree")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "true"
}

// getGitRoot returns the root directory of the git repository
func getGitRoot(dir string) (string, error) {
	cmd := execCommand("git", "-C", dir, "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git root: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// CreateWorktree creates a new git worktree with auto-generated task_id
// Returns Paths containing the worktree directory, branch name, and task_id
func CreateWorktree(projectDir string) (*Paths, error) {
	if projectDir == "" {
		projectDir = "."
	}

	// Verify it's a git repository
	if !isGitRepo(projectDir) {
		return nil, fmt.Errorf("not a git repository: %s", projectDir)
	}

	// Get git root for consistent path calculation
	gitRoot, err := getGitRoot(projectDir)
	if err != nil {
		return nil, err
	}

	// Generate task ID
	taskID, err := generateTaskID()
	if err != nil {
		return nil, err
	}

	// Calculate paths
	worktreeDir := filepath.Join(gitRoot, ".worktrees", fmt.Sprintf("do-%s", taskID))
	branchName := fmt.Sprintf("do/%s", taskID)

	// Create worktree with new branch
	cmd := execCommand("git", "-C", gitRoot, "worktree", "add", "-b", branchName, worktreeDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("failed to create worktree: %w\noutput: %s", err, string(output))
	}

	return &Paths{
		Dir:    worktreeDir,
		Branch: branchName,
		TaskID: taskID,
	}, nil
}
