package main

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestExtractCoverage(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"bare int", "92%", "92%"},
		{"bare float", "92.5%", "92.5%"},
		{"coverage prefix", "coverage: 92%", "92%"},
		{"total prefix", "TOTAL 92%", "92%"},
		{"all files", "All files 92%", "92%"},
		{"empty", "", ""},
		{"no number", "coverage: N/A", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractCoverage(tt.in); got != tt.want {
				t.Fatalf("extractCoverage(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestExtractTestResults(t *testing.T) {
	tests := []struct {
		name       string
		in         string
		wantPassed int
		wantFailed int
	}{
		{"pytest one line", "12 passed, 2 failed", 12, 2},
		{"pytest split lines", "12 passed\n2 failed", 12, 2},
		{"jest format", "Tests: 2 failed, 12 passed, 14 total", 12, 2},
		{"go test style count", "ok\texample.com/foo\t0.12s\t12 tests", 12, 0},
		{"zero counts", "0 passed, 0 failed", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed, failed := extractTestResults(tt.in)
			if passed != tt.wantPassed || failed != tt.wantFailed {
				t.Fatalf("extractTestResults(%q) = (%d, %d), want (%d, %d)", tt.in, passed, failed, tt.wantPassed, tt.wantFailed)
			}
		})
	}
}

func TestExtractFilesChanged(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"root file", "Modified: main.go\n", []string{"main.go"}},
		{"path file", "Created: codeagent-wrapper/utils.go\n", []string{"codeagent-wrapper/utils.go"}},
		{"at prefix", "Updated: @codeagent-wrapper/main.go\n", []string{"codeagent-wrapper/main.go"}},
		{"token scan", "Files: @main.go, @codeagent-wrapper/utils.go\n", []string{"main.go", "codeagent-wrapper/utils.go"}},
		{"space path", "Modified: dir/with space/file.go\n", []string{"dir/with space/file.go"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractFilesChanged(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("extractFilesChanged(%q) = %#v, want %#v", tt.in, got, tt.want)
			}
		})
	}

	t.Run("limits to first 10", func(t *testing.T) {
		var b strings.Builder
		for i := 0; i < 12; i++ {
			fmt.Fprintf(&b, "Modified: file%d.go\n", i)
		}
		got := extractFilesChanged(b.String())
		if len(got) != 10 {
			t.Fatalf("len(files)=%d, want 10: %#v", len(got), got)
		}
		for i := 0; i < 10; i++ {
			want := fmt.Sprintf("file%d.go", i)
			if got[i] != want {
				t.Fatalf("files[%d]=%q, want %q", i, got[i], want)
			}
		}
	})
}

func TestSafeTruncate(t *testing.T) {
	tests := []struct {
		name   string
		in     string
		maxLen int
		want   string
	}{
		{"empty", "", 4, ""},
		{"zero maxLen", "hello", 0, ""},
		{"one rune", "你好", 1, "你"},
		{"two runes no truncate", "你好", 2, "你好"},
		{"three runes no truncate", "你好", 3, "你好"},
		{"two runes truncates long", "你好世界", 2, "你"},
		{"three runes truncates long", "你好世界", 3, "你"},
		{"four with ellipsis", "你好世界啊", 4, "你..."},
		{"emoji", "🙂🙂🙂🙂🙂", 4, "🙂..."},
		{"no truncate", "你好世界", 4, "你好世界"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := safeTruncate(tt.in, tt.maxLen); got != tt.want {
				t.Fatalf("safeTruncate(%q, %d) = %q, want %q", tt.in, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestSanitizeOutput(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"ansi", "\x1b[31mred\x1b[0m", "red"},
		{"control chars", "a\x07b\r\nc\t", "ab\nc\t"},
		{"normal", "hello\nworld\t!", "hello\nworld\t!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeOutput(tt.in); got != tt.want {
				t.Fatalf("sanitizeOutput(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
