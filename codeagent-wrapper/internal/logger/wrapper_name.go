package logger

// WrapperName is the fixed name for this tool.
const WrapperName = "codeagent-wrapper"

// CurrentWrapperName returns the wrapper name (always "codeagent-wrapper").
func CurrentWrapperName() string { return WrapperName }

// LogPrefixes returns the log file name prefixes to look for.
func LogPrefixes() []string { return []string{WrapperName} }

// PrimaryLogPrefix returns the preferred filename prefix for log files.
func PrimaryLogPrefix() string { return WrapperName }
