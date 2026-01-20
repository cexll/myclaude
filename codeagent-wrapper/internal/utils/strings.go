package utils

import "strings"

func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen < 0 {
		return ""
	}
	return s[:maxLen] + "..."
}

// SafeTruncate safely truncates string to maxLen, avoiding panic and UTF-8 corruption.
func SafeTruncate(s string, maxLen int) string {
	if maxLen <= 0 || s == "" {
		return ""
	}

	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}

	if maxLen < 4 {
		return string(runes[:1])
	}

	cutoff := maxLen - 3
	if cutoff <= 0 {
		return string(runes[:1])
	}
	if len(runes) <= cutoff {
		return s
	}
	return string(runes[:cutoff]) + "..."
}

// SanitizeOutput removes ANSI escape sequences and control characters.
func SanitizeOutput(s string) string {
	var result strings.Builder
	inEscape := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			inEscape = true
			i++ // skip '['
			continue
		}
		if inEscape {
			if (s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= 'a' && s[i] <= 'z') {
				inEscape = false
			}
			continue
		}
		// Keep printable chars and common whitespace.
		if s[i] >= 32 || s[i] == '\n' || s[i] == '\t' {
			result.WriteByte(s[i])
		}
	}
	return result.String()
}
