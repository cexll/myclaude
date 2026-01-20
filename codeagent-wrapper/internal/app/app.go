package wrapper

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	version               = "6.0.0-alpha1"
	defaultWorkdir        = "."
	defaultTimeout        = 7200 // seconds (2 hours)
	defaultCoverageTarget = 90.0
	codexLogLineLimit     = 1000
	stdinSpecialChars     = "\n\\\"'`$"
	stderrCaptureLimit    = 4 * 1024
	defaultBackendName    = "codex"
	defaultCodexCommand   = "codex"

	// stdout close reasons
	stdoutCloseReasonWait  = "wait-done"
	stdoutCloseReasonDrain = "drain-timeout"
	stdoutCloseReasonCtx   = "context-cancel"
	stdoutDrainTimeout     = 100 * time.Millisecond
)

// Test hooks for dependency injection
var (
	stdinReader         io.Reader = os.Stdin
	isTerminalFn                  = defaultIsTerminal
	codexCommand                  = defaultCodexCommand
	cleanupHook         func()
	startupCleanupAsync = true

	buildCodexArgsFn   = buildCodexArgs
	selectBackendFn    = selectBackend
	cleanupLogsFn      = cleanupOldLogs
	defaultBuildArgsFn = buildCodexArgs
	runTaskFn          = runCodexTask
	exitFn             = os.Exit
)

func runStartupCleanup() {
	if cleanupLogsFn == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			logWarn(fmt.Sprintf("cleanupOldLogs panic: %v", r))
		}
	}()
	if _, err := cleanupLogsFn(); err != nil {
		logWarn(fmt.Sprintf("cleanupOldLogs error: %v", err))
	}
}

func scheduleStartupCleanup() {
	if !startupCleanupAsync {
		runStartupCleanup()
		return
	}
	if cleanupLogsFn == nil {
		return
	}
	fn := cleanupLogsFn
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logWarn(fmt.Sprintf("cleanupOldLogs panic: %v", r))
			}
		}()
		if _, err := fn(); err != nil {
			logWarn(fmt.Sprintf("cleanupOldLogs error: %v", err))
		}
	}()
}

func runCleanupMode() int {
	if cleanupLogsFn == nil {
		fmt.Fprintln(os.Stderr, "Cleanup failed: log cleanup function not configured")
		return 1
	}

	stats, err := cleanupLogsFn()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cleanup failed: %v\n", err)
		return 1
	}

	fmt.Println("Cleanup completed")
	fmt.Printf("Files scanned: %d\n", stats.Scanned)
	fmt.Printf("Files deleted: %d\n", stats.Deleted)
	if len(stats.DeletedFiles) > 0 {
		for _, f := range stats.DeletedFiles {
			fmt.Printf("  - %s\n", f)
		}
	}
	fmt.Printf("Files kept: %d\n", stats.Kept)
	if len(stats.KeptFiles) > 0 {
		for _, f := range stats.KeptFiles {
			fmt.Printf("  - %s\n", f)
		}
	}
	if stats.Errors > 0 {
		fmt.Printf("Deletion errors: %d\n", stats.Errors)
	}
	return 0
}

func readAgentPromptFile(path string, allowOutsideClaudeDir bool) (string, error) {
	raw := strings.TrimSpace(path)
	if raw == "" {
		return "", nil
	}

	expanded := raw
	if raw == "~" || strings.HasPrefix(raw, "~/") || strings.HasPrefix(raw, "~\\") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if raw == "~" {
			expanded = home
		} else {
			expanded = home + raw[1:]
		}
	}

	absPath, err := filepath.Abs(expanded)
	if err != nil {
		return "", err
	}
	absPath = filepath.Clean(absPath)

	home, err := os.UserHomeDir()
	if err != nil {
		if !allowOutsideClaudeDir {
			return "", err
		}
		logWarn(fmt.Sprintf("Failed to resolve home directory for prompt file validation: %v; proceeding without restriction", err))
	} else {
		allowedDirs := []string{
			filepath.Clean(filepath.Join(home, ".claude")),
			filepath.Clean(filepath.Join(home, ".codeagent", "agents")),
		}
		for i := range allowedDirs {
			allowedAbs, err := filepath.Abs(allowedDirs[i])
			if err == nil {
				allowedDirs[i] = filepath.Clean(allowedAbs)
			}
		}

		isWithinDir := func(path, dir string) bool {
			rel, err := filepath.Rel(dir, path)
			if err != nil {
				return false
			}
			rel = filepath.Clean(rel)
			if rel == "." {
				return true
			}
			if rel == ".." {
				return false
			}
			prefix := ".." + string(os.PathSeparator)
			return !strings.HasPrefix(rel, prefix)
		}

		if !allowOutsideClaudeDir {
			withinAllowed := false
			for _, dir := range allowedDirs {
				if isWithinDir(absPath, dir) {
					withinAllowed = true
					break
				}
			}
			if !withinAllowed {
				logWarn(fmt.Sprintf("Refusing to read prompt file outside allowed dirs (%s): %s", strings.Join(allowedDirs, ", "), absPath))
				return "", fmt.Errorf("prompt file must be under ~/.claude or ~/.codeagent/agents")
			}

			resolvedPath, errPath := filepath.EvalSymlinks(absPath)
			if errPath == nil {
				resolvedPath = filepath.Clean(resolvedPath)
				resolvedAllowed := make([]string, 0, len(allowedDirs))
				for _, dir := range allowedDirs {
					resolvedBase, errBase := filepath.EvalSymlinks(dir)
					if errBase != nil {
						continue
					}
					resolvedAllowed = append(resolvedAllowed, filepath.Clean(resolvedBase))
				}
				if len(resolvedAllowed) > 0 {
					withinResolved := false
					for _, dir := range resolvedAllowed {
						if isWithinDir(resolvedPath, dir) {
							withinResolved = true
							break
						}
					}
					if !withinResolved {
						logWarn(fmt.Sprintf("Refusing to read prompt file outside allowed dirs (%s) (resolved): %s", strings.Join(resolvedAllowed, ", "), resolvedPath))
						return "", fmt.Errorf("prompt file must be under ~/.claude or ~/.codeagent/agents")
					}
				}
			}
		} else {
			withinAllowed := false
			for _, dir := range allowedDirs {
				if isWithinDir(absPath, dir) {
					withinAllowed = true
					break
				}
			}
			if !withinAllowed {
				logWarn(fmt.Sprintf("Reading prompt file outside allowed dirs (%s): %s", strings.Join(allowedDirs, ", "), absPath))
			}
		}
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(data), "\r\n"), nil
}

func wrapTaskWithAgentPrompt(prompt string, task string) string {
	return "<agent-prompt>\n" + prompt + "\n</agent-prompt>\n\n" + task
}

func runCleanupHook() {
	if logger := activeLogger(); logger != nil {
		logger.Flush()
	}
	if cleanupHook != nil {
		cleanupHook()
	}
}

func printHelp() {
	name := currentWrapperName()
	help := fmt.Sprintf(`%[1]s - Go wrapper for AI CLI backends

Usage:
    %[1]s "task" [workdir]
    %[1]s --backend claude "task" [workdir]
    %[1]s --prompt-file /path/to/prompt.md "task" [workdir]
    %[1]s - [workdir]              Read task from stdin
    %[1]s resume <session_id> "task" [workdir]
    %[1]s resume <session_id> - [workdir]
    %[1]s --parallel               Run tasks in parallel (config from stdin)
    %[1]s --parallel --full-output Run tasks in parallel with full output (legacy)
    %[1]s --version
    %[1]s --help

Parallel mode examples:
    %[1]s --parallel < tasks.txt
    echo '...' | %[1]s --parallel
    %[1]s --parallel --full-output < tasks.txt
    %[1]s --parallel <<'EOF'

Environment Variables:
    CODEX_TIMEOUT         Timeout in milliseconds (default: 7200000)
    CODEAGENT_ASCII_MODE  Use ASCII symbols instead of Unicode (PASS/WARN/FAIL)

Exit Codes:
    0    Success
    1    General error (missing args, no output)
    124  Timeout
    127  backend command not found
    130  Interrupted (Ctrl+C)
    *    Passthrough from backend process`, name)
	fmt.Println(help)
}
