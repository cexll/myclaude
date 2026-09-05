package parser

import (
	"strings"
	"testing"
)

func TestParseJSONStream_Opencode(t *testing.T) {
	input := `{"type":"step_start","timestamp":1768187730683,"sessionID":"ses_44fced3c7ffe83sZpzY1rlQka3","part":{"id":"prt_bb0339afa001NTqoJ2NS8x91zP","sessionID":"ses_44fced3c7ffe83sZpzY1rlQka3","messageID":"msg_bb033866f0011oZxTqvfy0TKtS","type":"step-start","snapshot":"904f0fd58c125b79e60f0993e38f9d9f6200bf47"}}
{"type":"text","timestamp":1768187744432,"sessionID":"ses_44fced3c7ffe83sZpzY1rlQka3","part":{"id":"prt_bb0339cb5001QDd0Lh0PzFZpa3","sessionID":"ses_44fced3c7ffe83sZpzY1rlQka3","messageID":"msg_bb033866f0011oZxTqvfy0TKtS","type":"text","text":"Hello from opencode"}}
{"type":"step_finish","timestamp":1768187744471,"sessionID":"ses_44fced3c7ffe83sZpzY1rlQka3","part":{"id":"prt_bb033d0af0019VRZzpO2OVW1na","sessionID":"ses_44fced3c7ffe83sZpzY1rlQka3","messageID":"msg_bb033866f0011oZxTqvfy0TKtS","type":"step-finish","reason":"stop","snapshot":"904f0fd58c125b79e60f0993e38f9d9f6200bf47","cost":0}}`

	message, threadID := ParseJSONStreamInternal(strings.NewReader(input), nil, nil, nil, nil)

	if threadID != "ses_44fced3c7ffe83sZpzY1rlQka3" {
		t.Errorf("threadID = %q, want %q", threadID, "ses_44fced3c7ffe83sZpzY1rlQka3")
	}
	if message != "Hello from opencode" {
		t.Errorf("message = %q, want %q", message, "Hello from opencode")
	}
}

func TestParseJSONStream_Opencode_MultipleTextEvents(t *testing.T) {
	input := `{"type":"text","sessionID":"ses_123","part":{"type":"text","text":"Part 1"}}
{"type":"text","sessionID":"ses_123","part":{"type":"text","text":" Part 2"}}
{"type":"step_finish","sessionID":"ses_123","part":{"type":"step-finish","reason":"stop"}}`

	message, threadID := ParseJSONStreamInternal(strings.NewReader(input), nil, nil, nil, nil)

	if threadID != "ses_123" {
		t.Errorf("threadID = %q, want %q", threadID, "ses_123")
	}
	if message != "Part 1 Part 2" {
		t.Errorf("message = %q, want %q", message, "Part 1 Part 2")
	}
}

func TestParseJSONStream_Opencode_NoStopReason(t *testing.T) {
	input := `{"type":"text","sessionID":"ses_456","part":{"type":"text","text":"Content"}}
{"type":"step_finish","sessionID":"ses_456","part":{"type":"step-finish","reason":"tool-calls"}}`

	message, threadID := ParseJSONStreamInternal(strings.NewReader(input), nil, nil, nil, nil)

	if threadID != "ses_456" {
		t.Errorf("threadID = %q, want %q", threadID, "ses_456")
	}
	if message != "Content" {
		t.Errorf("message = %q, want %q", message, "Content")
	}
}
