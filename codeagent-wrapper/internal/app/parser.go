package wrapper

import (
	"bufio"
	"io"

	parser "codeagent-wrapper/internal/parser"

	"github.com/goccy/go-json"
)

func parseJSONStream(r io.Reader) (message, threadID string) {
	return parseJSONStreamWithLog(r, logWarn, logInfo)
}

func parseJSONStreamWithWarn(r io.Reader, warnFn func(string)) (message, threadID string) {
	return parseJSONStreamWithLog(r, warnFn, logInfo)
}

func parseJSONStreamWithLog(r io.Reader, warnFn func(string), infoFn func(string)) (message, threadID string) {
	return parseJSONStreamInternal(r, warnFn, infoFn, nil, nil)
}

func parseJSONStreamInternal(r io.Reader, warnFn func(string), infoFn func(string), onMessage func(), onComplete func()) (message, threadID string) {
	return parser.ParseJSONStreamInternal(r, warnFn, infoFn, onMessage, onComplete)
}

func hasKey(m map[string]json.RawMessage, key string) bool { return parser.HasKey(m, key) }

func discardInvalidJSON(decoder *json.Decoder, reader *bufio.Reader) (*bufio.Reader, error) {
	return parser.DiscardInvalidJSON(decoder, reader)
}

func normalizeText(text interface{}) string { return parser.NormalizeText(text) }
