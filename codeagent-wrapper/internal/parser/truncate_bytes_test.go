package parser

import "testing"

func TestTruncateBytes(t *testing.T) {
	if got := TruncateBytes([]byte("abc"), 3); got != "abc" {
		t.Fatalf("TruncateBytes() = %q, want %q", got, "abc")
	}
	if got := TruncateBytes([]byte("abcd"), 3); got != "abc..." {
		t.Fatalf("TruncateBytes() = %q, want %q", got, "abc...")
	}
	if got := TruncateBytes([]byte("abcd"), -1); got != "" {
		t.Fatalf("TruncateBytes() = %q, want empty string", got)
	}
}
