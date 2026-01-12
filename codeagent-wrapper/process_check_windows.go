//go:build windows
// +build windows

package main

import (
	"errors"
	"os"
	"syscall"
	"time"
	"unsafe"
)

const (
	processQueryLimitedInformation = 0x1000
	stillActive                    = 259 // STILL_ACTIVE exit code
)

var (
	findProcess      = os.FindProcess
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	getProcessTimes  = kernel32.NewProc("GetProcessTimes")
	fileTimeToUnixFn = fileTimeToUnix
)

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

// getProcessStartTime returns the start time of a process on Windows.
// Returns zero time if the start time cannot be determined.
func getProcessStartTime(pid int) time.Time {
	if pid <= 0 {
		return time.Time{}
	}

	handle, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		return time.Time{}
	}
	defer syscall.CloseHandle(handle)

	var creationTime, exitTime, kernelTime, userTime syscall.Filetime
	ret, _, _ := getProcessTimes.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&creationTime)),
		uintptr(unsafe.Pointer(&exitTime)),
		uintptr(unsafe.Pointer(&kernelTime)),
		uintptr(unsafe.Pointer(&userTime)),
	)

	if ret == 0 {
		return time.Time{}
	}

	return fileTimeToUnixFn(creationTime)
}

// fileTimeToUnix converts Windows FILETIME to Unix time.
func fileTimeToUnix(ft syscall.Filetime) time.Time {
	// FILETIME is 100-nanosecond intervals since January 1, 1601 UTC
	nsec := ft.Nanoseconds()
	return time.Unix(0, nsec)
}
