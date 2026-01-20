package utils

import (
	"strings"
	"testing"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		maxLen int
		want   string
	}{
		{"empty string", "", 10, ""},
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"needs truncation", "hello world", 5, "hello..."},
		{"zero maxLen", "hello", 0, "..."},
		{"negative maxLen", "hello", -1, ""},
		{"maxLen 1", "hello", 1, "h..."},
		{"unicode bytes truncate", "你好世界", 10, "你好世\xe7..."},  // Truncate works on bytes, not runes
		{"mixed truncate", "hello世界abc", 7, "hello\xe4\xb8..."}, // byte-based truncation
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Truncate(tt.s, tt.maxLen)
			if got != tt.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestSafeTruncate(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		maxLen int
		want   string
	}{
		{"empty string", "", 10, ""},
		{"zero maxLen", "hello", 0, ""},
		{"negative maxLen", "hello", -1, ""},
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"needs truncation", "hello world", 8, "hello..."},
		{"maxLen 1", "hello", 1, "h"},
		{"maxLen 2", "hello", 2, "h"},
		{"maxLen 3", "hello", 3, "h"},
		{"maxLen 4", "hello", 4, "h..."},
		{"unicode preserved", "你好世界", 10, "你好世界"},
		{"unicode exact", "你好世界", 4, "你好世界"},
		{"unicode truncate", "你好世界test", 6, "你好世..."},
		{"mixed unicode", "ab你好cd", 5, "ab..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SafeTruncate(tt.s, tt.maxLen)
			if got != tt.want {
				t.Errorf("SafeTruncate(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestSanitizeOutput(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want string
	}{
		{"empty string", "", ""},
		{"plain text", "hello world", "hello world"},
		{"with newline", "hello\nworld", "hello\nworld"},
		{"with tab", "hello\tworld", "hello\tworld"},
		{"ANSI color red", "\x1b[31mred\x1b[0m", "red"},
		{"ANSI bold", "\x1b[1mbold\x1b[0m", "bold"},
		{"ANSI complex", "\x1b[1;31;40mtext\x1b[0m", "text"},
		{"control chars", "hello\x00\x01\x02world", "helloworld"},
		{"mixed ANSI and control", "\x1b[32m\x00ok\x1b[0m", "ok"},
		{"multiple ANSI sequences", "\x1b[31mred\x1b[0m \x1b[32mgreen\x1b[0m", "red green"},
		{"incomplete escape", "\x1b[", ""},
		{"escape without bracket", "\x1bA", "A"},
		{"cursor movement", "\x1b[2Aup\x1b[2Bdown", "updown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeOutput(tt.s)
			if got != tt.want {
				t.Errorf("SanitizeOutput(%q) = %q, want %q", tt.s, got, tt.want)
			}
		})
	}
}

func BenchmarkTruncate(b *testing.B) {
	s := strings.Repeat("hello world ", 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Truncate(s, 50)
	}
}

func BenchmarkSafeTruncate(b *testing.B) {
	s := strings.Repeat("你好世界", 100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SafeTruncate(s, 50)
	}
}

func BenchmarkSanitizeOutput(b *testing.B) {
	s := strings.Repeat("\x1b[31mred\x1b[0m text ", 50)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SanitizeOutput(s)
	}
}
