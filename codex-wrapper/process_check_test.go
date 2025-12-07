package main

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"
)

func TestIsProcessRunning(t *testing.T) {
	t.Run("current process", func(t *testing.T) {
		if !isProcessRunning(os.Getpid()) {
			t.Fatalf("expected current process (pid=%d) to be running", os.Getpid())
		}
	})

	t.Run("fake pid", func(t *testing.T) {
		const nonexistentPID = 1 << 30
		if isProcessRunning(nonexistentPID) {
			t.Fatalf("expected pid %d to be reported as not running", nonexistentPID)
		}
	})

	t.Run("terminated process", func(t *testing.T) {
		pid := exitedProcessPID(t)
		if isProcessRunning(pid) {
			t.Fatalf("expected exited child process (pid=%d) to be reported as not running", pid)
		}
	})

	t.Run("boundary values", func(t *testing.T) {
		if isProcessRunning(0) {
			t.Fatalf("pid 0 should never be treated as running")
		}
		if isProcessRunning(-42) {
			t.Fatalf("negative pid should never be treated as running")
		}
	})

	t.Run("find process error", func(t *testing.T) {
		original := findProcess
		defer func() { findProcess = original }()

		mockErr := errors.New("findProcess failure")
		findProcess = func(pid int) (*os.Process, error) {
			return nil, mockErr
		}

		if isProcessRunning(1234) {
			t.Fatalf("expected false when os.FindProcess fails")
		}
	})
}

func exitedProcessPID(t *testing.T) int {
	t.Helper()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "exit 0")
	} else {
		cmd = exec.Command("sh", "-c", "exit 0")
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start helper process: %v", err)
	}
	pid := cmd.Process.Pid

	if err := cmd.Wait(); err != nil {
		t.Fatalf("helper process did not exit cleanly: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	return pid
}

func TestRunProcessCheckSmoke(t *testing.T) {
	t.Run("current process", func(t *testing.T) {
		if !isProcessRunning(os.Getpid()) {
			t.Fatalf("expected current process (pid=%d) to be running", os.Getpid())
		}
	})

	t.Run("fake pid", func(t *testing.T) {
		const nonexistentPID = 1 << 30
		if isProcessRunning(nonexistentPID) {
			t.Fatalf("expected pid %d to be reported as not running", nonexistentPID)
		}
	})

	t.Run("boundary values", func(t *testing.T) {
		if isProcessRunning(0) {
			t.Fatalf("pid 0 should never be treated as running")
		}
		if isProcessRunning(-42) {
			t.Fatalf("negative pid should never be treated as running")
		}
	})

	t.Run("find process error", func(t *testing.T) {
		original := findProcess
		defer func() { findProcess = original }()

		mockErr := errors.New("findProcess failure")
		findProcess = func(pid int) (*os.Process, error) {
			return nil, mockErr
		}

		if isProcessRunning(1234) {
			t.Fatalf("expected false when os.FindProcess fails")
		}
	})
}
