//go:build unix || darwin || linux
// +build unix darwin linux

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
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

func TestGetProcessStartTimeReadsProcStat(t *testing.T) {
	pid := 4321
	boot := time.Unix(1_710_000_000, 0)
	startTicks := uint64(4500)

	statFields := make([]string, 25)
	for i := range statFields {
		statFields[i] = strconv.Itoa(i + 1)
	}
	statFields[19] = strconv.FormatUint(startTicks, 10)
	statContent := fmt.Sprintf("%d (%s) %s", pid, "cmd with space", strings.Join(statFields, " "))

	stubReadFile(t, func(path string) ([]byte, error) {
		switch path {
		case fmt.Sprintf("/proc/%d/stat", pid):
			return []byte(statContent), nil
		case "/proc/stat":
			return []byte(fmt.Sprintf("cpu 0 0 0 0\nbtime %d\n", boot.Unix())), nil
		default:
			return nil, os.ErrNotExist
		}
	})

	got := getProcessStartTime(pid)
	want := boot.Add(time.Duration(startTicks/100) * time.Second)
	if !got.Equal(want) {
		t.Fatalf("getProcessStartTime() = %v, want %v", got, want)
	}
}

func TestGetProcessStartTimeInvalidData(t *testing.T) {
	pid := 99
	stubReadFile(t, func(path string) ([]byte, error) {
		switch path {
		case fmt.Sprintf("/proc/%d/stat", pid):
			return []byte("garbage"), nil
		case "/proc/stat":
			return []byte("btime not-a-number\n"), nil
		default:
			return nil, os.ErrNotExist
		}
	})

	if got := getProcessStartTime(pid); !got.IsZero() {
		t.Fatalf("invalid /proc data should return zero time, got %v", got)
	}
}

func TestGetBootTimeParsesBtime(t *testing.T) {
	const bootSec = 1_711_111_111
	stubReadFile(t, func(path string) ([]byte, error) {
		if path != "/proc/stat" {
			return nil, os.ErrNotExist
		}
		content := fmt.Sprintf("intr 0\nbtime %d\n", bootSec)
		return []byte(content), nil
	})

	got := getBootTime()
	want := time.Unix(bootSec, 0)
	if !got.Equal(want) {
		t.Fatalf("getBootTime() = %v, want %v", got, want)
	}
}

func TestGetBootTimeInvalidData(t *testing.T) {
	cases := []struct {
		name    string
		content string
	}{
		{"missing", "cpu 0 0 0 0"},
		{"malformed", "btime abc"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			stubReadFile(t, func(string) ([]byte, error) {
				return []byte(tt.content), nil
			})
			if got := getBootTime(); !got.IsZero() {
				t.Fatalf("getBootTime() unexpected value for %s: %v", tt.name, got)
			}
		})
	}
}

func stubReadFile(t *testing.T, fn func(string) ([]byte, error)) {
	t.Helper()
	original := readFileFn
	readFileFn = fn
	t.Cleanup(func() {
		readFileFn = original
	})
}
