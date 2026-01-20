package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds CLI configuration.
type Config struct {
	Mode               string // "new" or "resume"
	Task               string
	SessionID          string
	WorkDir            string
	Model              string
	ReasoningEffort    string
	ExplicitStdin      bool
	Timeout            int
	Backend            string
	Agent              string
	PromptFile         string
	PromptFileExplicit bool
	SkipPermissions    bool
	Yolo               bool
	MaxParallelWorkers int
}

// EnvFlagEnabled returns true when the environment variable exists and is not
// explicitly set to a falsey value ("0/false/no/off").
func EnvFlagEnabled(key string) bool {
	val, ok := os.LookupEnv(key)
	if !ok {
		return false
	}
	val = strings.TrimSpace(strings.ToLower(val))
	switch val {
	case "", "0", "false", "no", "off":
		return false
	default:
		return true
	}
}

func ParseBoolFlag(val string, defaultValue bool) bool {
	val = strings.TrimSpace(strings.ToLower(val))
	switch val {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return defaultValue
	}
}

// EnvFlagDefaultTrue returns true unless the env var is explicitly set to
// false/0/no/off.
func EnvFlagDefaultTrue(key string) bool {
	val, ok := os.LookupEnv(key)
	if !ok {
		return true
	}
	return ParseBoolFlag(val, true)
}

func ValidateAgentName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("agent name is empty")
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-', r == '_':
		default:
			return fmt.Errorf("agent name %q contains invalid character %q", name, r)
		}
	}
	return nil
}

const maxParallelWorkersLimit = 100

// ResolveMaxParallelWorkers reads CODEAGENT_MAX_PARALLEL_WORKERS. It returns 0
// for "unlimited".
func ResolveMaxParallelWorkers() int {
	raw := strings.TrimSpace(os.Getenv("CODEAGENT_MAX_PARALLEL_WORKERS"))
	if raw == "" {
		return 0
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0
	}
	if value > maxParallelWorkersLimit {
		return maxParallelWorkersLimit
	}
	return value
}
