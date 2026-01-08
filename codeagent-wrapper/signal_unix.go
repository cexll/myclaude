//go:build unix || darwin || linux
// +build unix darwin linux

package main

import (
	"syscall"
)

// sendTermSignal sends SIGTERM for graceful shutdown on Unix.
func sendTermSignal(proc processHandle) error {
	if proc == nil {
		return nil
	}
	return proc.Signal(syscall.SIGTERM)
}
