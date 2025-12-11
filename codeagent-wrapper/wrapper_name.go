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

var executablePathFn = os.Executable

func normalizeWrapperName(path string) string {
	if path == "" {
		return ""
	}

	base := filepath.Base(path)
	base = strings.TrimSuffix(base, ".exe") // tolerate Windows executables

	switch base {
	case defaultWrapperName, legacyWrapperName:
		return base
	default:
		return ""
	}
}

// currentWrapperName resolves the wrapper name based on the invoked binary.
// Only known names are honored to avoid leaking build/test binary names into logs.
func currentWrapperName() string {
	if len(os.Args) == 0 {
		return defaultWrapperName
	}

	if name := normalizeWrapperName(os.Args[0]); name != "" {
		return name
	}

	execPath, err := executablePathFn()
	if err == nil {
		if name := normalizeWrapperName(execPath); name != "" {
			return name
		}

		if resolved, err := filepath.EvalSymlinks(execPath); err == nil {
			if name := normalizeWrapperName(resolved); name != "" {
				return name
			}
			if alias := resolveAlias(execPath, resolved); alias != "" {
				return alias
			}
		}

		if alias := resolveAlias(execPath, ""); alias != "" {
			return alias
		}
	}

	return defaultWrapperName
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

func resolveAlias(execPath string, target string) string {
	if execPath == "" {
		return ""
	}

	dir := filepath.Dir(execPath)
	for _, candidate := range []string{defaultWrapperName, legacyWrapperName} {
		aliasPath := filepath.Join(dir, candidate)
		info, err := os.Lstat(aliasPath)
		if err != nil {
			continue
		}
		if info.Mode()&os.ModeSymlink == 0 {
			continue
		}

		resolved, err := filepath.EvalSymlinks(aliasPath)
		if err != nil {
			continue
		}
		if target != "" && resolved != target {
			continue
		}

		if name := normalizeWrapperName(aliasPath); name != "" {
			return name
		}
	}

	return ""
}
