package wrapper

import (
	janitor "codeagent-wrapper/internal/infrastructure/janitor"
	logstore "codeagent-wrapper/internal/infrastructure/logstore"
)

type Logger = logstore.Logger
type CleanupStats = janitor.CleanupStats

func NewLogger() (*Logger, error) { return logstore.NewLogger() }

func NewLoggerWithSuffix(suffix string) (*Logger, error) { return logstore.NewLoggerWithSuffix(suffix) }

func setLogger(l *Logger) { logstore.SetLogger(l) }

func closeLogger() error { return logstore.CloseLogger() }

func activeLogger() *Logger { return logstore.ActiveLogger() }

func logInfo(msg string) { logstore.LogInfo(msg) }

func logWarn(msg string) { logstore.LogWarn(msg) }

func logError(msg string) { logstore.LogError(msg) }

func cleanupOldLogs() (CleanupStats, error) { return janitor.CleanupOldLogs() }

func sanitizeLogSuffix(raw string) string { return logstore.SanitizeLogSuffix(raw) }
