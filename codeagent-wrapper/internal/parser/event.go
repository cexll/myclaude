package parser

import "github.com/goccy/go-json"

// JSONEvent represents a Codex JSON output event.
type JSONEvent struct {
	Type     string     `json:"type"`
	ThreadID string     `json:"thread_id,omitempty"`
	Item     *EventItem `json:"item,omitempty"`
}

// EventItem represents the item field in a JSON event.
type EventItem struct {
	Type string      `json:"type"`
	Text interface{} `json:"text"`
}

// ClaudeEvent for Claude stream-json format.
type ClaudeEvent struct {
	Type      string `json:"type"`
	Subtype   string `json:"subtype,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	Result    string `json:"result,omitempty"`
}

// GeminiEvent for Gemini stream-json format.
type GeminiEvent struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id,omitempty"`
	Role      string `json:"role,omitempty"`
	Content   string `json:"content,omitempty"`
	Delta     bool   `json:"delta,omitempty"`
	Status    string `json:"status,omitempty"`
}

// UnifiedEvent combines all backend event formats into a single structure
// to avoid multiple JSON unmarshal operations per event.
type UnifiedEvent struct {
	// Common fields
	Type string `json:"type"`

	// Codex-specific fields
	ThreadID string          `json:"thread_id,omitempty"`
	Item     json.RawMessage `json:"item,omitempty"` // Lazy parse

	// Claude-specific fields
	Subtype   string `json:"subtype,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	Result    string `json:"result,omitempty"`

	// Gemini-specific fields
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
	Delta   *bool  `json:"delta,omitempty"`
	Status  string `json:"status,omitempty"`

	// Opencode-specific fields (camelCase sessionID)
	OpencodeSessionID string          `json:"sessionID,omitempty"`
	Part              json.RawMessage `json:"part,omitempty"`
}

// OpencodePart represents the part field in opencode events.
type OpencodePart struct {
	Type      string `json:"type"`
	Text      string `json:"text,omitempty"`
	Reason    string `json:"reason,omitempty"`
	SessionID string `json:"sessionID,omitempty"`
}

// ItemContent represents the parsed item.text field for Codex events.
type ItemContent struct {
	Type string      `json:"type"`
	Text interface{} `json:"text"`
}
