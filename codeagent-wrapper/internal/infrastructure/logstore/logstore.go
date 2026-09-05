package logstore

import legacy "codeagent-wrapper/internal/logger"

type Logger = legacy.Logger

const WrapperName = legacy.WrapperName

func NewLogger() (*Logger, error) { return legacy.NewLogger() }

func NewLoggerWithSuffix(suffix string) (*Logger, error) { return legacy.NewLoggerWithSuffix(suffix) }

func SetLogger(l *Logger) { legacy.SetLogger(l) }

func CloseLogger() error { return legacy.CloseLogger() }

func ActiveLogger() *Logger { return legacy.ActiveLogger() }

func LogInfo(msg string) { legacy.LogInfo(msg) }

func LogDebug(msg string) { legacy.LogDebug(msg) }

func LogWarn(msg string) { legacy.LogWarn(msg) }

func LogError(msg string) { legacy.LogError(msg) }

func CurrentWrapperName() string { return legacy.CurrentWrapperName() }

func LogPrefixes() []string { return legacy.LogPrefixes() }

func PrimaryLogPrefix() string { return legacy.PrimaryLogPrefix() }

func LogConcurrencyPlanning(limit, total int) { legacy.LogConcurrencyPlanning(limit, total) }

func LogConcurrencyState(event, taskID string, active, limit int) {
	legacy.LogConcurrencyState(event, taskID, active, limit)
}

func SanitizeLogSuffix(raw string) string { return legacy.SanitizeLogSuffix(raw) }
