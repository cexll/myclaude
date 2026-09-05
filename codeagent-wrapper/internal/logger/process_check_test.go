package logger

import (
	"math"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"testing"
	"time"
)

func TestIsProcessRunning(t *testing.T) {
	t.Run("boundary values", func(t *testing.T) {
		if isProcessRunning(0) {
			t.Fatalf("pid 0 should never be treated as running")
		}
		if isProcessRunning(-1) {
			t.Fatalf("negative pid should never be treated as running")
		}
	})

	t.Run("pid out of int32 range", func(t *testing.T) {
		if strconv.IntSize <= 32 {
			t.Skip("int cannot represent values above int32 range")
		}

		pid := int(int64(math.MaxInt32) + 1)
		if isProcessRunning(pid) {
			t.Fatalf("expected pid %d (out of int32 range) to be treated as not running", pid)
		}
	})

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

func TestGetProcessStartTimeReadsProcStat(t *testing.T) {
	start := getProcessStartTime(os.Getpid())
	if start.IsZero() {
		t.Fatalf("expected non-zero start time for current process")
	}
	if start.After(time.Now().Add(5 * time.Second)) {
		t.Fatalf("start time is unexpectedly in the future: %v", start)
	}
}

func TestGetProcessStartTimeInvalidData(t *testing.T) {
	if !getProcessStartTime(0).IsZero() {
		t.Fatalf("expected zero time for pid 0")
	}
	if !getProcessStartTime(-1).IsZero() {
		t.Fatalf("expected zero time for negative pid")
	}
	if !getProcessStartTime(1 << 30).IsZero() {
		t.Fatalf("expected zero time for non-existent pid")
	}
	if strconv.IntSize > 32 {
		pid := int(int64(math.MaxInt32) + 1)
		if !getProcessStartTime(pid).IsZero() {
			t.Fatalf("expected zero time for pid %d (out of int32 range)", pid)
		}
	}
}

func TestGetBootTimeParsesBtime(t *testing.T) {
	t.Skip("legacy boot-time probing removed; start time now uses gopsutil")
}

func TestGetBootTimeInvalidData(t *testing.T) {
	t.Skip("legacy boot-time probing removed; start time now uses gopsutil")
}
