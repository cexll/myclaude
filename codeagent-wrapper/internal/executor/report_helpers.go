package executor

import "strings"

// extractCoverageGap extracts what's missing from coverage reports.
func extractCoverageGap(message string) string {
	if message == "" {
		return ""
	}

	lower := strings.ToLower(message)
	lines := strings.Split(message, "\n")

	for _, line := range lines {
		lineLower := strings.ToLower(line)
		line = strings.TrimSpace(line)

		if strings.Contains(lineLower, "uncovered") ||
			strings.Contains(lineLower, "not covered") ||
			strings.Contains(lineLower, "missing coverage") ||
			strings.Contains(lineLower, "lines not covered") {
			if len(line) > 100 {
				return line[:97] + "..."
			}
			return line
		}

		if strings.Contains(lineLower, "branch") && strings.Contains(lineLower, "not taken") {
			if len(line) > 100 {
				return line[:97] + "..."
			}
			return line
		}
	}

	if strings.Contains(lower, "function") && strings.Contains(lower, "0%") {
		for _, line := range lines {
			if strings.Contains(strings.ToLower(line), "0%") && strings.Contains(line, "function") {
				line = strings.TrimSpace(line)
				if len(line) > 100 {
					return line[:97] + "..."
				}
				return line
			}
		}
	}

	return ""
}

// extractErrorDetail extracts meaningful error context from task output.
func extractErrorDetail(message string, maxLen int) string {
	if message == "" || maxLen <= 0 {
		return ""
	}

	lines := strings.Split(message, "\n")
	var errorLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		lower := strings.ToLower(line)

		if strings.HasPrefix(line, "at ") && strings.Contains(line, "(") {
			if len(errorLines) > 0 && strings.HasPrefix(strings.ToLower(errorLines[len(errorLines)-1]), "at ") {
				continue
			}
		}

		if strings.Contains(lower, "error") ||
			strings.Contains(lower, "fail") ||
			strings.Contains(lower, "exception") ||
			strings.Contains(lower, "assert") ||
			strings.Contains(lower, "expected") ||
			strings.Contains(lower, "timeout") ||
			strings.Contains(lower, "not found") ||
			strings.Contains(lower, "cannot") ||
			strings.Contains(lower, "undefined") ||
			strings.HasPrefix(line, "FAIL") ||
			strings.HasPrefix(line, "●") {
			errorLines = append(errorLines, line)
		}
	}

	if len(errorLines) == 0 {
		start := len(lines) - 5
		if start < 0 {
			start = 0
		}
		for _, line := range lines[start:] {
			line = strings.TrimSpace(line)
			if line != "" {
				errorLines = append(errorLines, line)
			}
		}
	}

	result := strings.Join(errorLines, " | ")
	return safeTruncate(result, maxLen)
}
