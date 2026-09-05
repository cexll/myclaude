package executor

import "bytes"

type logWriter struct {
	prefix  string
	maxLen  int
	buf     bytes.Buffer
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
