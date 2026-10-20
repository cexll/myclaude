package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"codeagent-wrapper/internal/config"
)

const parallelSearchURL = "https://search.parallel.ai/mcp"

// The host owns MCP connections, tool permissions and conversation context. We
// only supply a default for a fresh installation, without writing host settings.
func codexParallelSearchArgs(cfg *config.Config) []string {
	if runtime.GOOS != "linux" || !config.EnvFlagDefaultTrue("CODEAGENT_PARALLEL_SEARCH") {
		return nil
	}
	workdir := cfg.WorkDir
	if cfg.Mode == "resume" {
		// BuildCodexArgs deliberately omits -C for resume, and the executor
		// does not set cmd.Dir. Check the directory the host actually loads.
		var err error
		workdir, err = os.Getwd()
		if err != nil {
			return nil
		}
	}
	if !codexSearchUnconfigured(workdir, "/etc/codex") || !codexSupportsHTTPMCP() || !codexParallelAnonymous(workdir) {
		return nil
	}
	return []string{
		"-c", `mcp_servers.codeagent_parallel.url="` + parallelSearchURL + `"`,
		"-c", `mcp_servers.codeagent_parallel.http_headers={"User-Agent"="myclaude-codeagent-wrapper"}`,
		"-c", `mcp_servers.codeagent_parallel.tool_timeout_sec=60`,
		"-c", `web_search="disabled"`,
	}
}

// Absence is deliberately conservative: even an unrelated host config keeps
// the host's existing behavior. In particular, CLI overrides must not supersede
// saved disabled search, authenticated MCP servers, profiles or managed policy.
func codexSearchUnconfigured(workdir, systemDir string) bool {
	// Unknown host controls may carry a parent session's or enterprise settings.
	for _, entry := range os.Environ() {
		key := strings.SplitN(entry, "=", 2)[0]
		if strings.HasPrefix(key, "CODEX_") && key != "CODEX_HOME" && key != "CODEX_BYPASS_SANDBOX" && key != "CODEX_API_KEY" {
			return false
		}
	}
	if !pathAbsent(systemDir) {
		return false
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return false
	}
	codexHome := strings.TrimSpace(os.Getenv("CODEX_HOME"))
	if codexHome == "" {
		codexHome = filepath.Join(home, ".codex")
	}
	if !filepath.IsAbs(codexHome) {
		return false
	}
	for _, dir := range []string{filepath.Join(home, ".codex"), codexHome} {
		if info, err := os.Lstat(dir); err == nil && (!info.IsDir() || info.Mode()&os.ModeSymlink != 0) || err != nil && !os.IsNotExist(err) {
			return false
		}
		for _, name := range []string{"config.toml", "config.yaml", "config.yml", "config.json", "managed_config.toml", "requirements.toml", "auth.json", ".credentials.json", "cloud-config-bundle-cache.json"} {
			if !pathAbsent(filepath.Join(dir, name)) {
				return false
			}
		}
	}
	if workdir == "" {
		workdir = "."
	}
	workdir, err = filepath.Abs(workdir)
	if err != nil {
		return false
	}
	// Resolve symlinks before walking ancestors, just as the host sees the cwd.
	workdir, err = filepath.EvalSymlinks(workdir)
	if err != nil {
		return false
	}
	for {
		if !pathAbsent(filepath.Join(workdir, ".codex", "config.toml")) {
			return false
		}
		parent := filepath.Dir(workdir)
		if parent == workdir {
			return true
		}
		workdir = parent
	}
}

// MCP credentials can survive removal of config.toml, including in a shared OS
// keyring. Ask the same host for status rather than reading credential values or
// changing its credential store. The anonymous endpoint reports "unsupported";
// every authenticated, failed or unfamiliar result keeps the native behavior.
func codexParallelAnonymous(workdir string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "codex",
		"-c", `mcp_servers.codeagent_parallel.url="`+parallelSearchURL+`"`,
		"-c", `mcp_servers.codeagent_parallel.http_headers={"User-Agent"="myclaude-codeagent-wrapper"}`,
		"mcp", "list", "--json")
	cmd.Dir = workdir
	out, err := cmd.Output()
	if err != nil || len(out) > 64*1024 {
		return false
	}
	var servers []struct {
		Name       string `json:"name"`
		Enabled    bool   `json:"enabled"`
		AuthStatus string `json:"auth_status"`
		Transport  struct {
			Type string `json:"type"`
			URL  string `json:"url"`
		} `json:"transport"`
	}
	if json.Unmarshal(out, &servers) != nil || len(servers) != 1 {
		return false
	}
	server := servers[0]
	return server.Name == "codeagent_parallel" && server.Enabled && server.AuthStatus == "unsupported" &&
		server.Transport.Type == "streamable_http" && server.Transport.URL == parallelSearchURL
}

func pathAbsent(path string) bool {
	_, err := os.Lstat(path)
	return os.IsNotExist(err)
}

type codexVersionCheck struct {
	once      sync.Once
	supported bool
}

var codexVersions sync.Map

func codexSupportsHTTPMCP() bool {
	path, err := exec.LookPath("codex")
	if err != nil {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	// A replaced binary gets a fresh probe. Concurrent tasks share the probe,
	// never conversation state or a Search MCP session_id.
	key := fmt.Sprintf("%s:%d:%d", path, info.Size(), info.ModTime().UnixNano())
	value, _ := codexVersions.LoadOrStore(key, &codexVersionCheck{})
	check := value.(*codexVersionCheck)
	check.once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		out, err := exec.CommandContext(ctx, path, "--version").Output()
		if err == nil {
			check.supported = supportedCodexVersion(strings.TrimSpace(string(out)))
		}
	})
	return check.supported
}

func supportedCodexVersion(version string) bool {
	const prefix = "codex-cli "
	if !strings.HasPrefix(version, prefix) {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(version, prefix), ".")
	if len(parts) != 3 {
		return false
	}
	numbers := [3]int{}
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return false
		}
		numbers[i] = n
	}
	return numbers[0] > 0 || numbers[1] >= 163
}
