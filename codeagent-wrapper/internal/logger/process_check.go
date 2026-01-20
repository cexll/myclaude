package logger

import (
	"errors"
	"math"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

func pidToInt32(pid int) (int32, bool) {
	if pid <= 0 || pid > math.MaxInt32 {
		return 0, false
	}
	return int32(pid), true
}

// isProcessRunning reports whether a process with the given pid appears to be running.
// It is intentionally conservative on errors to avoid deleting logs for live processes.
func isProcessRunning(pid int) bool {
	pid32, ok := pidToInt32(pid)
	if !ok {
		return false
	}

	exists, err := process.PidExists(pid32)
	if err == nil {
		return exists
	}

	// If we can positively identify that the process doesn't exist, report false.
	if errors.Is(err, process.ErrorProcessNotRunning) {
		return false
	}

	// Permission/inspection failures: assume it's running to be safe.
	return true
}

// getProcessStartTime returns the start time of a process.
// Returns zero time if the start time cannot be determined.
func getProcessStartTime(pid int) time.Time {
	pid32, ok := pidToInt32(pid)
	if !ok {
		return time.Time{}
	}

	proc, err := process.NewProcess(pid32)
	if err != nil {
		return time.Time{}
	}

	ms, err := proc.CreateTime()
	if err != nil || ms <= 0 {
		return time.Time{}
	}

	return time.UnixMilli(ms)
}

func IsProcessRunning(pid int) bool { return isProcessRunning(pid) }

func GetProcessStartTime(pid int) time.Time { return getProcessStartTime(pid) }
