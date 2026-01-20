package logger

import (
	"os"
	"path/filepath"
	"time"
)

func SetProcessRunningCheck(fn func(int) bool) (restore func()) {
	prev := processRunningCheck
	if fn != nil {
		processRunningCheck = fn
	} else {
		processRunningCheck = isProcessRunning
	}
	return func() { processRunningCheck = prev }
}

func SetProcessStartTimeFn(fn func(int) time.Time) (restore func()) {
	prev := processStartTimeFn
	if fn != nil {
		processStartTimeFn = fn
	} else {
		processStartTimeFn = getProcessStartTime
	}
	return func() { processStartTimeFn = prev }
}

func SetRemoveLogFileFn(fn func(string) error) (restore func()) {
	prev := removeLogFileFn
	if fn != nil {
		removeLogFileFn = fn
	} else {
		removeLogFileFn = os.Remove
	}
	return func() { removeLogFileFn = prev }
}

func SetGlobLogFilesFn(fn func(string) ([]string, error)) (restore func()) {
	prev := globLogFiles
	if fn != nil {
		globLogFiles = fn
	} else {
		globLogFiles = filepath.Glob
	}
	return func() { globLogFiles = prev }
}

func SetFileStatFn(fn func(string) (os.FileInfo, error)) (restore func()) {
	prev := fileStatFn
	if fn != nil {
		fileStatFn = fn
	} else {
		fileStatFn = os.Lstat
	}
	return func() { fileStatFn = prev }
}

func SetEvalSymlinksFn(fn func(string) (string, error)) (restore func()) {
	prev := evalSymlinksFn
	if fn != nil {
		evalSymlinksFn = fn
	} else {
		evalSymlinksFn = filepath.EvalSymlinks
	}
	return func() { evalSymlinksFn = prev }
}
