//go:build windows
// +build windows

package main

import (
	"os"
	"testing"
	"time"
)

func TestIsProcessRunning(t *testing.T) {
	t.Run("boundary values", func(t *testing.T) {
		if isProcessRunning(0) {
			t.Fatalf("expected pid 0 to be reported as not running")
		}
		if isProcessRunning(-1) {
			t.Fatalf("expected pid -1 to be reported as not running")
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
}

func TestGetBootTimeParsesBtime(t *testing.T) {
	t.Skip("getBootTime is only implemented on Unix-like systems")
}

func TestGetBootTimeInvalidData(t *testing.T) {
	t.Skip("getBootTime is only implemented on Unix-like systems")
}
