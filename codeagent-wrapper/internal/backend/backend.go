package backend

import config "codeagent-wrapper/internal/config"

// Backend defines the contract for invoking different AI CLI backends.
// Each backend is responsible for supplying the executable command and
// building the argument list based on the wrapper config.
type Backend interface {
	Name() string
	BuildArgs(cfg *config.Config, targetArg string) []string
	Command() string
	Env(baseURL, apiKey string) map[string]string
}

var (
	logWarnFn  = func(string) {}
	logErrorFn = func(string) {}
)

// SetLogFuncs configures optional logging hooks used by some backends.
// Callers can safely pass nil to disable the hook.
func SetLogFuncs(warnFn, errorFn func(string)) {
	if warnFn != nil {
		logWarnFn = warnFn
	} else {
		logWarnFn = func(string) {}
	}
	if errorFn != nil {
		logErrorFn = errorFn
	} else {
		logErrorFn = func(string) {}
	}
}
