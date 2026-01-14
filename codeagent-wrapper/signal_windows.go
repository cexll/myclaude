//go:build windows
// +build windows

package main

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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
		if err := killProcessTree(pid); err == nil {
			return nil
		}
	}
	return proc.Kill()
}

func killProcessTree(pid int) error {
	if pid <= 0 {
		return nil
	}

	wmic := "wmic"
	if root := os.Getenv("SystemRoot"); root != "" {
		wmic = filepath.Join(root, "System32", "wbem", "WMIC.exe")
	}

	queryChildren := "(ParentProcessId=" + strconv.Itoa(pid) + ")"
	listCmd := exec.Command(wmic, "process", "where", queryChildren, "get", "ProcessId", "/VALUE")
	listCmd.Stderr = io.Discard
	out, err := listCmd.Output()
	if err == nil {
		for _, childPID := range parseWMICPIDs(out) {
			_ = killProcessTree(childPID)
		}
	}

	querySelf := "(ProcessId=" + strconv.Itoa(pid) + ")"
	termCmd := exec.Command(wmic, "process", "where", querySelf, "call", "terminate")
	termCmd.Stdout = io.Discard
	termCmd.Stderr = io.Discard
	if termErr := termCmd.Run(); termErr != nil && err == nil {
		err = termErr
	}
	return err
}

func parseWMICPIDs(out []byte) []int {
	const prefix = "ProcessId="
	var pids []int
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, prefix)))
		if err != nil || n <= 0 {
			continue
		}
		pids = append(pids, n)
	}
	return pids
}
