package wrapper

import logstore "codeagent-wrapper/internal/infrastructure/logstore"

const wrapperName = logstore.WrapperName

func currentWrapperName() string { return logstore.CurrentWrapperName() }

func primaryLogPrefix() string { return logstore.PrimaryLogPrefix() }
