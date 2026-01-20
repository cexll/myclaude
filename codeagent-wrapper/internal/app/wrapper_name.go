package wrapper

import ilogger "codeagent-wrapper/internal/logger"

const wrapperName = ilogger.WrapperName

func currentWrapperName() string { return ilogger.CurrentWrapperName() }

func primaryLogPrefix() string { return ilogger.PrimaryLogPrefix() }
