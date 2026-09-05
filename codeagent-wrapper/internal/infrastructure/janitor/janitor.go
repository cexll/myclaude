package janitor

import legacy "codeagent-wrapper/internal/logger"

type CleanupStats = legacy.CleanupStats

func CleanupOldLogs() (CleanupStats, error) { return legacy.CleanupOldLogs() }

func IsUnsafeFile(path string, tempDir string) (bool, string) {
	return legacy.IsUnsafeFile(path, tempDir)
}

func IsPIDReused(logPath string, pid int) bool { return legacy.IsPIDReused(logPath, pid) }

func ParsePIDFromLog(path string) (int, bool) { return legacy.ParsePIDFromLog(path) }
