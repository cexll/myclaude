//go:build windows
// +build windows

package main

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

// sendTermSignal on Windows directly kills the process.
// SIGTERM is not supported on Windows.
func sendTermSignal(proc processHandle) error {
	if proc == nil {
		return nil
	}
	pid := proc.Pid()
	if pid > 0 {
		// Kill the whole process tree to avoid leaving inheriting child processes around.
		// This also helps prevent exec.Cmd.Wait() from blocking on stderr/stdout pipes held open by children.
		taskkill := "taskkill"
		if root := os.Getenv("SystemRoot"); root != "" {
			taskkill = filepath.Join(root, "System32", "taskkill.exe")
		}
		cmd := exec.Command(taskkill, "/PID", strconv.Itoa(pid), "/T", "/F")
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		if err := cmd.Run(); err == nil {
			return nil
		}
	}
	return proc.Kill()
}
