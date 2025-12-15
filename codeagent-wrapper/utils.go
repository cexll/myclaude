package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

func resolveTimeout() int {
	raw := os.Getenv("CODEX_TIMEOUT")
	if raw == "" {
		return defaultTimeout
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		logWarn(fmt.Sprintf("Invalid CODEX_TIMEOUT '%s', falling back to %ds", raw, defaultTimeout))
		return defaultTimeout
	}

	if parsed > 10000 {
		return parsed / 1000
	}
	return parsed
}

func readPipedTask() (string, error) {
	if isTerminal() {
		logInfo("Stdin is tty, skipping pipe read")
		return "", nil
	}
	logInfo("Reading from stdin pipe...")
	data, err := io.ReadAll(stdinReader)
	if err != nil {
		return "", fmt.Errorf("read stdin: %w", err)
	}
	if len(data) == 0 {
		logInfo("Stdin pipe returned empty data")
		return "", nil
	}
	logInfo(fmt.Sprintf("Read %d bytes from stdin pipe", len(data)))
	return string(data), nil
}

func shouldUseStdin(taskText string, piped bool) bool {
	if piped {
		return true
	}
	if len(taskText) > 800 {
		return true
	}
	return strings.IndexAny(taskText, stdinSpecialChars) >= 0
}

func defaultIsTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return true
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func isTerminal() bool {
	return isTerminalFn()
}

func getEnv(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

type logWriter struct {
	prefix string
	maxLen int
	buf    bytes.Buffer
	dropped bool
}

func newLogWriter(prefix string, maxLen int) *logWriter {
	if maxLen <= 0 {
		maxLen = codexLogLineLimit
	}
	return &logWriter{prefix: prefix, maxLen: maxLen}
}

func (lw *logWriter) Write(p []byte) (int, error) {
	if lw == nil {
		return len(p), nil
	}
	total := len(p)
	for len(p) > 0 {
		if idx := bytes.IndexByte(p, '\n'); idx >= 0 {
			lw.writeLimited(p[:idx])
			lw.logLine(true)
			p = p[idx+1:]
			continue
		}
		lw.writeLimited(p)
		break
	}
	return total, nil
}

func (lw *logWriter) Flush() {
	if lw == nil || lw.buf.Len() == 0 {
		return
	}
	lw.logLine(false)
}

func (lw *logWriter) logLine(force bool) {
	if lw == nil {
		return
	}
	line := lw.buf.String()
	dropped := lw.dropped
	lw.dropped = false
	lw.buf.Reset()
	if line == "" && !force {
		return
	}
	if lw.maxLen > 0 {
		if dropped {
			if lw.maxLen > 3 {
				line = line[:min(len(line), lw.maxLen-3)] + "..."
			} else {
				line = line[:min(len(line), lw.maxLen)]
			}
		} else if len(line) > lw.maxLen {
			cutoff := lw.maxLen
			if cutoff > 3 {
				line = line[:cutoff-3] + "..."
			} else {
				line = line[:cutoff]
			}
		}
	}
	logInfo(lw.prefix + line)
}

func (lw *logWriter) writeLimited(p []byte) {
	if lw == nil || len(p) == 0 {
		return
	}
	if lw.maxLen <= 0 {
		lw.buf.Write(p)
		return
	}

	remaining := lw.maxLen - lw.buf.Len()
	if remaining <= 0 {
		lw.dropped = true
		return
	}
	if len(p) <= remaining {
		lw.buf.Write(p)
		return
	}
	lw.buf.Write(p[:remaining])
	lw.dropped = true
}

type tailBuffer struct {
	limit int
	data  []byte
}

func (b *tailBuffer) Write(p []byte) (int, error) {
	if b.limit <= 0 {
		return len(p), nil
	}

	if len(p) >= b.limit {
		b.data = append(b.data[:0], p[len(p)-b.limit:]...)
		return len(p), nil
	}

	total := len(b.data) + len(p)
	if total <= b.limit {
		b.data = append(b.data, p...)
		return len(p), nil
	}

	overflow := total - b.limit
	b.data = append(b.data[overflow:], p...)
	return len(p), nil
}

func (b *tailBuffer) String() string {
	return string(b.data)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen < 0 {
		return ""
	}
	return s[:maxLen] + "..."
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func hello() string {
	return "hello world"
}

func greet(name string) string {
	return "hello " + name
}

func farewell(name string) string {
	return "goodbye " + name
}
