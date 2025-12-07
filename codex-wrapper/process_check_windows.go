//go:build windows
// +build windows

package main

import (
	"errors"
	"os"
	"syscall"
)

const (
	processQueryLimitedInformation = 0x1000
	stillActive                    = 259 // STILL_ACTIVE exit code
)

var findProcess = os.FindProcess

// isProcessRunning returns true if a process with the given pid is running on Windows.
func isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	if _, err := findProcess(pid); err != nil {
		return false
	}

	handle, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		if errors.Is(err, syscall.ERROR_ACCESS_DENIED) {
			return true
		}
		return false
	}
	defer syscall.CloseHandle(handle)

	var exitCode uint32
	if err := syscall.GetExitCodeProcess(handle, &exitCode); err != nil {
		return true
	}

	return exitCode == stillActive
}
