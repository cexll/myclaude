//go:build unix || darwin || linux
// +build unix darwin linux

package main

import (
	"errors"
	"os"
	"syscall"
)

var findProcess = os.FindProcess

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
