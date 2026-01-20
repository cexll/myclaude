package wrapper

import config "codeagent-wrapper/internal/config"

// Keep the existing Config name throughout the codebase, but source the
// implementation from internal/config.
type Config = config.Config
