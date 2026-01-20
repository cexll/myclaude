package logger

import "sync/atomic"

var loggerPtr atomic.Pointer[Logger]

func setLogger(l *Logger) {
	loggerPtr.Store(l)
}

func closeLogger() error {
	logger := loggerPtr.Swap(nil)
	if logger == nil {
		return nil
	}
	return logger.Close()
}

func activeLogger() *Logger {
	return loggerPtr.Load()
}

func logDebug(msg string) {
	if logger := activeLogger(); logger != nil {
		logger.Debug(msg)
	}
}

func logInfo(msg string) {
	if logger := activeLogger(); logger != nil {
		logger.Info(msg)
	}
}

func logWarn(msg string) {
	if logger := activeLogger(); logger != nil {
		logger.Warn(msg)
	}
}

func logError(msg string) {
	if logger := activeLogger(); logger != nil {
		logger.Error(msg)
	}
}

func SetLogger(l *Logger) { setLogger(l) }

func CloseLogger() error { return closeLogger() }

func ActiveLogger() *Logger { return activeLogger() }

func LogInfo(msg string) { logInfo(msg) }

func LogDebug(msg string) { logDebug(msg) }

func LogWarn(msg string) { logWarn(msg) }

func LogError(msg string) { logError(msg) }
