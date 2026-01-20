package wrapper

import backend "codeagent-wrapper/internal/backend"

func init() {
	backend.SetLogFuncs(logWarn, logError)
}
