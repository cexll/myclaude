package wrapper

import ilogger "codeagent-wrapper/internal/logger"

type Logger = ilogger.Logger
type CleanupStats = ilogger.CleanupStats

func NewLogger() (*Logger, error) { return ilogger.NewLogger() }

func NewLoggerWithSuffix(suffix string) (*Logger, error) { return ilogger.NewLoggerWithSuffix(suffix) }

func setLogger(l *Logger) { ilogger.SetLogger(l) }

func closeLogger() error { return ilogger.CloseLogger() }

func activeLogger() *Logger { return ilogger.ActiveLogger() }

func logInfo(msg string) { ilogger.LogInfo(msg) }

func logWarn(msg string) { ilogger.LogWarn(msg) }

func logError(msg string) { ilogger.LogError(msg) }

func cleanupOldLogs() (CleanupStats, error) { return ilogger.CleanupOldLogs() }

func sanitizeLogSuffix(raw string) string { return ilogger.SanitizeLogSuffix(raw) }
