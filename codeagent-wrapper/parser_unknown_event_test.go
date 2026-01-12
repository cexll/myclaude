package main

import (
	"strings"
	"testing"
)

func TestBackendParseJSONStream_UnknownEventsAreSilent(t *testing.T) {
	input := strings.Join([]string{
		`{"type":"turn.started"}`,
		`{"type":"assistant","text":"hi"}`,
		`{"type":"user","text":"yo"}`,
		`{"type":"item.completed","item":{"type":"agent_message","text":"ok"}}`,
	}, "\n")

	var infos []string
	infoFn := func(msg string) { infos = append(infos, msg) }

	message, threadID := parseJSONStreamInternal(strings.NewReader(input), nil, infoFn, nil, nil)
	if message != "ok" {
		t.Fatalf("message=%q, want %q (infos=%v)", message, "ok", infos)
	}
	if threadID != "" {
		t.Fatalf("threadID=%q, want empty (infos=%v)", threadID, infos)
	}

	for _, msg := range infos {
		if strings.Contains(msg, "Agent event:") {
			t.Fatalf("unexpected log for unknown event: %q", msg)
		}
	}
}
