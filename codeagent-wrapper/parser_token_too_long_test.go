package main

import (
	"strings"
	"testing"
)

func TestParseJSONStream_SkipsOverlongLineAndContinues(t *testing.T) {
	// Exceed the 10MB bufio.Scanner limit in parseJSONStreamInternal.
	tooLong := strings.Repeat("a", 11*1024*1024)

	input := strings.Join([]string{
		`{"type":"item.completed","item":{"type":"other_type","text":"` + tooLong + `"}}`,
		`{"type":"thread.started","thread_id":"t-1"}`,
		`{"type":"item.completed","item":{"type":"agent_message","text":"ok"}}`,
	}, "\n")

	var warns []string
	warnFn := func(msg string) { warns = append(warns, msg) }

	gotMessage, gotThreadID := parseJSONStreamInternal(strings.NewReader(input), warnFn, nil, nil, nil)
	if gotMessage != "ok" {
		t.Fatalf("message=%q, want %q (warns=%v)", gotMessage, "ok", warns)
	}
	if gotThreadID != "t-1" {
		t.Fatalf("threadID=%q, want %q (warns=%v)", gotThreadID, "t-1", warns)
	}
	if len(warns) == 0 || !strings.Contains(warns[0], "Skipped overlong JSON line") {
		t.Fatalf("expected warning about overlong JSON line, got %v", warns)
	}
}
