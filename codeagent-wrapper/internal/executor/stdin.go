package executor

import "strings"

const stdinSpecialChars = "\n\\\"'`$"

func ShouldUseStdin(taskText string, piped bool) bool {
	if piped {
		return true
	}
	if len(taskText) > 800 {
		return true
	}
	return strings.ContainsAny(taskText, stdinSpecialChars)
}
