package main

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultWrapperName = "codeagent-wrapper"
	legacyWrapperName  = "codex-wrapper"
)

// currentWrapperName resolves the wrapper name based on the invoked binary.
// Only known names are honored to avoid leaking build/test binary names into logs.
func currentWrapperName() string {
	if len(os.Args) == 0 {
		return defaultWrapperName
	}

	base := filepath.Base(os.Args[0])
	base = strings.TrimSuffix(base, ".exe") // tolerate Windows executables

	switch base {
	case defaultWrapperName, legacyWrapperName:
		return base
	default:
		return defaultWrapperName
	}
}

// logPrefixes returns the set of accepted log name prefixes, including the
// current wrapper name and legacy aliases.
func logPrefixes() []string {
	prefixes := []string{currentWrapperName(), defaultWrapperName, legacyWrapperName}
	seen := make(map[string]struct{}, len(prefixes))
	var unique []string
	for _, prefix := range prefixes {
		if prefix == "" {
			continue
		}
		if _, ok := seen[prefix]; ok {
			continue
		}
		seen[prefix] = struct{}{}
		unique = append(unique, prefix)
	}
	return unique
}

// primaryLogPrefix returns the preferred filename prefix for log files.
// Defaults to the current wrapper name when available, otherwise falls back
// to the canonical default name.
func primaryLogPrefix() string {
	prefixes := logPrefixes()
	if len(prefixes) == 0 {
		return defaultWrapperName
	}
	return prefixes[0]
}
