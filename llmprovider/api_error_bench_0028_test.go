package llmprovider

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// 0028-MADR D-A10's benchmarks: classifying an error whose message says
// "tokens", as every overflow message does, so redaction's token passes run.

func benchClassify(b *testing.B, body string) {
	b.Helper()
	b.SetBytes(int64(len(body)))
	b.ReportAllocs()
	for b.Loop() {
		_ = ClassifyHTTPError("together", &http.Response{StatusCode: http.StatusBadRequest, Status: "400 Bad Request",
			Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))})
	}
}

func tokensBody(words int) string {
	msg := "This request used too many tokens. " + strings.Repeat("detail ", words)
	raw, err := json.Marshal(map[string]any{"error": map[string]any{"message": msg, "type": "invalid_request_error"}})
	if err != nil {
		panic(err)
	}
	return string(raw)
}

// BenchmarkClassify_16KiBTokens is a message at the redaction limit before
// 0028 (16 KiB) that says "tokens".
func BenchmarkClassify_16KiBTokens(b *testing.B) { benchClassify(b, tokensBody(2300)) }

// BenchmarkClassify_2KiBOverflow is a 2 KiB OpenAI-style overflow error.
func BenchmarkClassify_2KiBOverflow(b *testing.B) {
	msg := "This model's maximum context length is 128000 tokens. However, your messages resulted in 131072 tokens. " +
		strings.Repeat("detail ", 260) + "Please reduce the length of the messages."
	raw, err := json.Marshal(map[string]any{"error": map[string]any{"message": msg, "type": "invalid_request_error",
		"code": "context_length_exceeded"}})
	if err != nil {
		b.Fatal(err)
	}
	benchClassify(b, string(raw))
}
