package wrapper

import backend "codeagent-wrapper/internal/infrastructure/backend"

func init() {
	backend.SetLogFuncs(logWarn, logError)
}
