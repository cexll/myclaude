//go:build unix || darwin || linux
// +build unix darwin linux

package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var findProcess = os.FindProcess
var readFileFn = os.ReadFile

// isProcessRunning returns true if a process with the given pid is running on Unix-like systems.
func isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	proc, err := findProcess(pid)
	if err != nil || proc == nil {
		return false
	}

	err = proc.Signal(syscall.Signal(0))
	if err != nil && (errors.Is(err, syscall.ESRCH) || errors.Is(err, os.ErrProcessDone)) {
		return false
	}
	return true
}

// getProcessStartTime returns the start time of a process on Unix-like systems.
// Returns zero time if the start time cannot be determined.
func getProcessStartTime(pid int) time.Time {
	if pid <= 0 {
		return time.Time{}
	}

	// Read /proc/<pid>/stat to get process start time
	statPath := fmt.Sprintf("/proc/%d/stat", pid)
	data, err := readFileFn(statPath)
	if err != nil {
		return time.Time{}
	}

	// Parse stat file: fields are space-separated, but comm (field 2) can contain spaces
	// Find the last ')' to skip comm field safely
	content := string(data)
	lastParen := strings.LastIndex(content, ")")
	if lastParen == -1 {
		return time.Time{}
	}

	fields := strings.Fields(content[lastParen+1:])
	if len(fields) < 20 {
		return time.Time{}
	}

	// Field 22 (index 19 after comm) is starttime in clock ticks since boot
	startTicks, err := strconv.ParseUint(fields[19], 10, 64)
	if err != nil {
		return time.Time{}
	}

	// Get system boot time
	bootTime := getBootTime()
	if bootTime.IsZero() {
		return time.Time{}
	}

	// Convert ticks to duration (typically 100 ticks/sec on most systems)
	ticksPerSec := uint64(100) // sysconf(_SC_CLK_TCK), typically 100
	startTime := bootTime.Add(time.Duration(startTicks/ticksPerSec) * time.Second)

	return startTime
}

// getBootTime returns the system boot time by reading /proc/stat.
func getBootTime() time.Time {
	data, err := readFileFn("/proc/stat")
	if err != nil {
		return time.Time{}
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "btime ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				bootSec, err := strconv.ParseInt(fields[1], 10, 64)
				if err == nil {
					return time.Unix(bootSec, 0)
				}
			}
		}
	}

	return time.Time{}
}
