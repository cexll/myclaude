package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// JSONEvent represents a Codex JSON output event
type JSONEvent struct {
	Type     string     `json:"type"`
	ThreadID string     `json:"thread_id,omitempty"`
	Item     *EventItem `json:"item,omitempty"`
}

// EventItem represents the item field in a JSON event
type EventItem struct {
	Type string      `json:"type"`
	Text interface{} `json:"text"`
}

// ClaudeEvent for Claude stream-json format
type ClaudeEvent struct {
	Type      string `json:"type"`
	Subtype   string `json:"subtype,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	Result    string `json:"result,omitempty"`
}

// ClaudeMessageEvent 兼容 Claude Code 新版 stream-json 输出（2.x 观察到的格式之一）。
// 典型形态：{"type":"assistant","message":{"id":"...","role":"assistant","content":[{"type":"text","text":"..."}]}, "session_id":"..."}
// 注意：字段可能随版本演进，因此这里尽量宽松解析，仅提取可用的文本内容。
type ClaudeMessageEvent struct {
	Type      string      `json:"type"`
	SessionID string      `json:"session_id,omitempty"`
	Message   interface{} `json:"message,omitempty"`
}

// GeminiEvent for Gemini stream-json format
type GeminiEvent struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id,omitempty"`
	Role      string `json:"role,omitempty"`
	Content   string `json:"content,omitempty"`
	Delta     bool   `json:"delta,omitempty"`
	Status    string `json:"status,omitempty"`
}

func parseJSONStream(r io.Reader) (message, threadID string) {
	return parseJSONStreamWithLog(r, logWarn, logInfo)
}

func parseJSONStreamWithWarn(r io.Reader, warnFn func(string)) (message, threadID string) {
	return parseJSONStreamWithLog(r, warnFn, logInfo)
}

func parseJSONStreamWithLog(r io.Reader, warnFn func(string), infoFn func(string)) (message, threadID string) {
	return parseJSONStreamInternal(r, warnFn, infoFn, nil)
}

const (
	jsonLineReaderSize   = 64 * 1024
	jsonLineMaxBytes     = 10 * 1024 * 1024
	jsonLinePreviewBytes = 256
)

type codexHeader struct {
	Type     string `json:"type"`
	ThreadID string `json:"thread_id,omitempty"`
	Item     *struct {
		Type string `json:"type"`
	} `json:"item,omitempty"`
}

func parseJSONStreamInternal(r io.Reader, warnFn func(string), infoFn func(string), onMessage func()) (message, threadID string) {
	reader := bufio.NewReaderSize(r, jsonLineReaderSize)

	if warnFn == nil {
		warnFn = func(string) {}
	}
	if infoFn == nil {
		infoFn = func(string) {}
	}

	notifyMessage := func() {
		if onMessage != nil {
			onMessage()
		}
	}

	totalEvents := 0

	var (
		codexMessage  string
		claudeMessage string
		geminiBuffer  strings.Builder
	)

	for {
		line, tooLong, err := readLineWithLimit(reader, jsonLineMaxBytes, jsonLinePreviewBytes)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			warnFn("Read stdout error: " + err.Error())
			break
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		totalEvents++

		if tooLong {
			warnFn(fmt.Sprintf("Skipped overlong JSON line (> %d bytes): %s", jsonLineMaxBytes, truncateBytes(line, 100)))
			continue
		}

		var codex codexHeader
		if err := json.Unmarshal(line, &codex); err == nil {
			isCodex := codex.ThreadID != "" || (codex.Item != nil && codex.Item.Type != "")
			if isCodex {
				var details []string
				if codex.ThreadID != "" {
					details = append(details, fmt.Sprintf("thread_id=%s", codex.ThreadID))
				}
				if codex.Item != nil && codex.Item.Type != "" {
					details = append(details, fmt.Sprintf("item_type=%s", codex.Item.Type))
				}
				if len(details) > 0 {
					infoFn(fmt.Sprintf("Parsed event #%d type=%s (%s)", totalEvents, codex.Type, strings.Join(details, ", ")))
				} else {
					infoFn(fmt.Sprintf("Parsed event #%d type=%s", totalEvents, codex.Type))
				}

				switch codex.Type {
				case "thread.started":
					threadID = codex.ThreadID
					infoFn(fmt.Sprintf("thread.started event thread_id=%s", threadID))
				case "item.completed":
					itemType := ""
					if codex.Item != nil {
						itemType = codex.Item.Type
					}

					if itemType == "agent_message" {
						var event JSONEvent
						if err := json.Unmarshal(line, &event); err != nil {
							warnFn(fmt.Sprintf("Failed to parse Codex event: %s", truncateBytes(line, 100)))
							continue
						}

						normalized := ""
						if event.Item != nil {
							normalized = normalizeText(event.Item.Text)
						}
						infoFn(fmt.Sprintf("item.completed event item_type=%s message_len=%d", itemType, len(normalized)))
						if normalized != "" {
							codexMessage = normalized
							notifyMessage()
						}
					} else {
						infoFn(fmt.Sprintf("item.completed event item_type=%s", itemType))
					}
				}
				continue
			}
		}

		var raw map[string]json.RawMessage
		if err := json.Unmarshal(line, &raw); err != nil {
			warnFn(fmt.Sprintf("Failed to parse line: %s", truncateBytes(line, 100)))
			continue
		}

		switch {
		case hasKey(raw, "message"):
			var event ClaudeMessageEvent
			if err := json.Unmarshal(line, &event); err != nil {
				warnFn(fmt.Sprintf("Failed to parse Claude message event: %s", truncateBytes(line, 100)))
				continue
			}

			if event.SessionID != "" && threadID == "" {
				threadID = event.SessionID
			}

			role := event.Type
			if m, ok := event.Message.(map[string]interface{}); ok {
				if sid, ok := m["session_id"].(string); ok && sid != "" && threadID == "" {
					threadID = sid
				}
				if r, ok := m["role"].(string); ok && r != "" {
					role = r
				}
			}

			text := extractClaudeText(event.Message)
			infoFn(fmt.Sprintf("Parsed Claude message event #%d role=%s text_len=%d", totalEvents, role, len(text)))
			if role == "assistant" && text != "" {
				claudeMessage = text
				notifyMessage()
			}

		case hasKey(raw, "subtype") || hasKey(raw, "result"):
			var event ClaudeEvent
			if err := json.Unmarshal(line, &event); err != nil {
				warnFn(fmt.Sprintf("Failed to parse Claude event: %s", truncateBytes(line, 100)))
				continue
			}

			if event.SessionID != "" && threadID == "" {
				threadID = event.SessionID
			}

			infoFn(fmt.Sprintf("Parsed Claude event #%d type=%s subtype=%s result_len=%d", totalEvents, event.Type, event.Subtype, len(event.Result)))

			if event.Result != "" {
				claudeMessage = event.Result
				notifyMessage()
			}

		case hasKey(raw, "role") || hasKey(raw, "delta"):
			var event GeminiEvent
			if err := json.Unmarshal(line, &event); err != nil {
				warnFn(fmt.Sprintf("Failed to parse Gemini event: %s", truncateBytes(line, 100)))
				continue
			}

			if event.SessionID != "" && threadID == "" {
				threadID = event.SessionID
			}

			if event.Content != "" {
				geminiBuffer.WriteString(event.Content)
				notifyMessage()
			}

			infoFn(fmt.Sprintf("Parsed Gemini event #%d type=%s role=%s delta=%t status=%s content_len=%d", totalEvents, event.Type, event.Role, event.Delta, event.Status, len(event.Content)))

		default:
			warnFn(fmt.Sprintf("Unknown event format: %s", truncateBytes(line, 100)))
		}
	}

	switch {
	case geminiBuffer.Len() > 0:
		message = geminiBuffer.String()
	case claudeMessage != "":
		message = claudeMessage
	default:
		message = codexMessage
	}

	infoFn(fmt.Sprintf("parseJSONStream completed: events=%d, message_len=%d, thread_id_found=%t", totalEvents, len(message), threadID != ""))
	return message, threadID
}

func hasKey(m map[string]json.RawMessage, key string) bool {
	_, ok := m[key]
	return ok
}

func discardInvalidJSON(decoder *json.Decoder, reader *bufio.Reader) (*bufio.Reader, error) {
	var buffered bytes.Buffer

	if decoder != nil {
		if buf := decoder.Buffered(); buf != nil {
			_, _ = buffered.ReadFrom(buf)
		}
	}

	line, err := reader.ReadBytes('\n')
	buffered.Write(line)

	data := buffered.Bytes()
	newline := bytes.IndexByte(data, '\n')
	if newline == -1 {
		return reader, err
	}

	remaining := data[newline+1:]
	if len(remaining) == 0 {
		return reader, err
	}

	return bufio.NewReader(io.MultiReader(bytes.NewReader(remaining), reader)), err
}

func readLineWithLimit(r *bufio.Reader, maxBytes int, previewBytes int) (line []byte, tooLong bool, err error) {
	if r == nil {
		return nil, false, errors.New("reader is nil")
	}
	if maxBytes <= 0 {
		return nil, false, errors.New("maxBytes must be > 0")
	}
	if previewBytes < 0 {
		previewBytes = 0
	}

	part, isPrefix, err := r.ReadLine()
	if err != nil {
		return nil, false, err
	}

	if !isPrefix {
		if len(part) > maxBytes {
			return part[:min(len(part), previewBytes)], true, nil
		}
		return part, false, nil
	}

	preview := make([]byte, 0, min(previewBytes, len(part)))
	if previewBytes > 0 {
		preview = append(preview, part[:min(previewBytes, len(part))]...)
	}

	buf := make([]byte, 0, min(maxBytes, len(part)*2))
	total := 0
	if len(part) > maxBytes {
		tooLong = true
	} else {
		buf = append(buf, part...)
		total = len(part)
	}

	for isPrefix {
		part, isPrefix, err = r.ReadLine()
		if err != nil {
			return nil, tooLong, err
		}

		if previewBytes > 0 && len(preview) < previewBytes {
			preview = append(preview, part[:min(previewBytes-len(preview), len(part))]...)
		}

		if !tooLong {
			if total+len(part) > maxBytes {
				tooLong = true
				continue
			}
			buf = append(buf, part...)
			total += len(part)
		}
	}

	if tooLong {
		return preview, true, nil
	}
	return buf, false, nil
}

func truncateBytes(b []byte, maxLen int) string {
	if len(b) <= maxLen {
		return string(b)
	}
	if maxLen < 0 {
		return ""
	}
	return string(b[:maxLen]) + "..."
}

func normalizeText(text interface{}) string {
	switch v := text.(type) {
	case string:
		return v
	case []interface{}:
		var sb strings.Builder
		for _, item := range v {
			if s, ok := item.(string); ok {
				sb.WriteString(s)
			}
		}
		return sb.String()
	default:
		return ""
	}
}

func extractClaudeText(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case []interface{}:
		var sb strings.Builder
		for _, item := range t {
			sb.WriteString(extractClaudeText(item))
		}
		return sb.String()
	case map[string]interface{}:
		if text, ok := t["text"].(string); ok && text != "" {
			return text
		}
		// 常见结构：{"content":[{"type":"text","text":"..."}]}
		if content, ok := t["content"]; ok {
			return extractClaudeText(content)
		}
		// 兜底：把 message 本身继续递归（某些实现可能嵌套）
		if msg, ok := t["message"]; ok {
			return extractClaudeText(msg)
		}
		return ""
	default:
		return ""
	}
}
