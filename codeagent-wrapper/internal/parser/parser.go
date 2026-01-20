package parser

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/goccy/go-json"
)

const (
	jsonLineReaderSize   = 64 * 1024
	jsonLineMaxBytes     = 10 * 1024 * 1024
	jsonLinePreviewBytes = 256
)

type lineScratch struct {
	buf     []byte
	preview []byte
}

const maxPooledLineScratchCap = 1 << 20 // 1 MiB

var lineScratchPool = sync.Pool{
	New: func() any {
		return &lineScratch{
			buf:     make([]byte, 0, jsonLineReaderSize),
			preview: make([]byte, 0, jsonLinePreviewBytes),
		}
	},
}

func ParseJSONStreamInternal(r io.Reader, warnFn func(string), infoFn func(string), onMessage func(), onComplete func()) (message, threadID string) {
	reader := bufio.NewReaderSize(r, jsonLineReaderSize)
	scratch := lineScratchPool.Get().(*lineScratch)
	if scratch.buf == nil {
		scratch.buf = make([]byte, 0, jsonLineReaderSize)
	} else {
		scratch.buf = scratch.buf[:0]
	}
	if scratch.preview == nil {
		scratch.preview = make([]byte, 0, jsonLinePreviewBytes)
	} else {
		scratch.preview = scratch.preview[:0]
	}
	defer func() {
		if cap(scratch.buf) > maxPooledLineScratchCap {
			scratch.buf = nil
		} else if scratch.buf != nil {
			scratch.buf = scratch.buf[:0]
		}
		if cap(scratch.preview) > jsonLinePreviewBytes*4 {
			scratch.preview = nil
		} else if scratch.preview != nil {
			scratch.preview = scratch.preview[:0]
		}
		lineScratchPool.Put(scratch)
	}()

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

	notifyComplete := func() {
		if onComplete != nil {
			onComplete()
		}
	}

	totalEvents := 0

	var (
		codexMessage    string
		claudeMessage   string
		geminiBuffer    strings.Builder
		opencodeMessage strings.Builder
	)

	for {
		line, tooLong, err := readLineWithLimit(reader, jsonLineMaxBytes, jsonLinePreviewBytes, scratch)
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
			warnFn(fmt.Sprintf("Skipped overlong JSON line (> %d bytes): %s", jsonLineMaxBytes, TruncateBytes(line, 100)))
			continue
		}

		// Single unmarshal for all backend types
		var event UnifiedEvent
		if err := json.Unmarshal(line, &event); err != nil {
			warnFn(fmt.Sprintf("Failed to parse event: %s", TruncateBytes(line, 100)))
			continue
		}

		// Detect backend type by field presence
		isCodex := event.ThreadID != ""
		if !isCodex && len(event.Item) > 0 {
			var itemHeader struct {
				Type string `json:"type"`
			}
			if json.Unmarshal(event.Item, &itemHeader) == nil && itemHeader.Type != "" {
				isCodex = true
			}
		}
		// Codex-specific event types without thread_id or item
		if !isCodex && (event.Type == "turn.started" || event.Type == "turn.completed") {
			isCodex = true
		}
		isClaude := event.Subtype != "" || event.Result != ""
		if !isClaude && event.Type == "result" && event.SessionID != "" && event.Status == "" {
			isClaude = true
		}
		isGemini := (event.Type == "init" && event.SessionID != "") || event.Role != "" || event.Delta != nil || event.Status != ""
		isOpencode := event.OpencodeSessionID != "" && len(event.Part) > 0

		// Handle Opencode events first (most specific detection)
		if isOpencode {
			if threadID == "" {
				threadID = event.OpencodeSessionID
			}

			var part OpencodePart
			if err := json.Unmarshal(event.Part, &part); err != nil {
				warnFn(fmt.Sprintf("Failed to parse opencode part: %s", err.Error()))
				continue
			}

			// Extract sessionID from part if available
			if part.SessionID != "" && threadID == "" {
				threadID = part.SessionID
			}

			infoFn(fmt.Sprintf("Parsed Opencode event #%d type=%s part_type=%s", totalEvents, event.Type, part.Type))

			if event.Type == "text" && part.Text != "" {
				opencodeMessage.WriteString(part.Text)
				notifyMessage()
			}

			if part.Type == "step-finish" && part.Reason == "stop" {
				notifyComplete()
			}
			continue
		}

		// Handle Codex events
		if isCodex {
			var details []string
			if event.ThreadID != "" {
				details = append(details, fmt.Sprintf("thread_id=%s", event.ThreadID))
			}

			if len(details) > 0 {
				infoFn(fmt.Sprintf("Parsed event #%d type=%s (%s)", totalEvents, event.Type, strings.Join(details, ", ")))
			} else {
				infoFn(fmt.Sprintf("Parsed event #%d type=%s", totalEvents, event.Type))
			}

			switch event.Type {
			case "thread.started":
				threadID = event.ThreadID
				infoFn(fmt.Sprintf("thread.started event thread_id=%s", threadID))

			case "thread.completed":
				if event.ThreadID != "" && threadID == "" {
					threadID = event.ThreadID
				}
				infoFn(fmt.Sprintf("thread.completed event thread_id=%s", event.ThreadID))
				notifyComplete()

			case "turn.completed":
				infoFn("turn.completed event")
				notifyComplete()

			case "item.completed":
				var itemType string
				if len(event.Item) > 0 {
					var itemHeader struct {
						Type string `json:"type"`
					}
					if err := json.Unmarshal(event.Item, &itemHeader); err == nil {
						itemType = itemHeader.Type
					}
				}

				if itemType == "agent_message" && len(event.Item) > 0 {
					// Lazy parse: only parse item content when needed
					var item ItemContent
					if err := json.Unmarshal(event.Item, &item); err == nil {
						normalized := NormalizeText(item.Text)
						infoFn(fmt.Sprintf("item.completed event item_type=%s message_len=%d", itemType, len(normalized)))
						if normalized != "" {
							codexMessage = normalized
							notifyMessage()
						}
					} else {
						warnFn(fmt.Sprintf("Failed to parse item content: %s", err.Error()))
					}
				} else {
					infoFn(fmt.Sprintf("item.completed event item_type=%s", itemType))
				}
			}
			continue
		}

		// Handle Claude events
		if isClaude {
			if event.SessionID != "" && threadID == "" {
				threadID = event.SessionID
			}

			infoFn(fmt.Sprintf("Parsed Claude event #%d type=%s subtype=%s result_len=%d", totalEvents, event.Type, event.Subtype, len(event.Result)))

			if event.Result != "" {
				claudeMessage = event.Result
				notifyMessage()
			}

			if event.Type == "result" {
				notifyComplete()
			}
			continue
		}

		// Handle Gemini events
		if isGemini {
			if event.SessionID != "" && threadID == "" {
				threadID = event.SessionID
			}

			if event.Content != "" {
				geminiBuffer.WriteString(event.Content)
			}

			if event.Status != "" {
				notifyMessage()

				if event.Type == "result" && (event.Status == "success" || event.Status == "error" || event.Status == "complete" || event.Status == "failed") {
					notifyComplete()
				}
			}

			delta := false
			if event.Delta != nil {
				delta = *event.Delta
			}

			infoFn(fmt.Sprintf("Parsed Gemini event #%d type=%s role=%s delta=%t status=%s content_len=%d", totalEvents, event.Type, event.Role, delta, event.Status, len(event.Content)))
			continue
		}

		// Unknown event format from other backends (turn.started/assistant/user); ignore.
		continue
	}

	switch {
	case opencodeMessage.Len() > 0:
		message = opencodeMessage.String()
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

func HasKey(m map[string]json.RawMessage, key string) bool {
	_, ok := m[key]
	return ok
}

func DiscardInvalidJSON(decoder *json.Decoder, reader *bufio.Reader) (*bufio.Reader, error) {
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

func readLineWithLimit(r *bufio.Reader, maxBytes int, previewBytes int, scratch *lineScratch) (line []byte, tooLong bool, err error) {
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

	if scratch == nil {
		scratch = &lineScratch{}
	}
	if scratch.preview == nil {
		scratch.preview = make([]byte, 0, min(previewBytes, len(part)))
	}
	if scratch.buf == nil {
		scratch.buf = make([]byte, 0, min(maxBytes, len(part)*2))
	}

	preview := scratch.preview[:0]
	if previewBytes > 0 {
		preview = append(preview, part[:min(previewBytes, len(part))]...)
	}

	buf := scratch.buf[:0]
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
		scratch.preview = preview
		scratch.buf = buf
		return preview, true, nil
	}
	scratch.preview = preview
	scratch.buf = buf
	return buf, false, nil
}

func TruncateBytes(b []byte, maxLen int) string {
	if len(b) <= maxLen {
		return string(b)
	}
	if maxLen < 0 {
		return ""
	}
	return string(b[:maxLen]) + "..."
}

func NormalizeText(text interface{}) string {
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
