package llmprovider_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
)

// windowRecord is what 0088-PLAN's OQ10 measurement keeps of Anthropic's
// reply (0088-MADR ST65): the request's shape, the status, the error's type
// and its message, redacted. Nothing else of the reply is read into it.
type windowRecord struct {
	Model      string `json:"model"`
	MaxTokens  int    `json:"max_tokens"`
	InputWords int    `json:"input_words"`
	Status     int    `json:"status"`
	Type       string `json:"error_type"`
	Message    string `json:"message"`
	StopReason string `json:"stop_reason,omitempty"`
}

// windowWords is the input's length in words, each a common English word
// that is one token: about 184,000 tokens, inside the band where the input
// alone fits a 200,000-token window and the input plus 32,000 does not.
const (
	windowWords     = 184000
	windowMaxTokens = 32000
)

var windowVocabulary = strings.Fields("the of and to in is it you that he was for on are with as his they be at one have this from or had by")

// windowBody is the one request's body: windowWords words and an instruction
// to answer in one word, so that an accepted request bills little output.
func windowBody(model string) ([]byte, error) {
	var b strings.Builder
	b.WriteString("Reply with the single word ok. Ignore the text that follows.\n")
	for i := range windowWords {
		b.WriteString(windowVocabulary[i%len(windowVocabulary)])
		b.WriteByte(' ')
	}
	return json.Marshal(map[string]any{
		"model":      model,
		"max_tokens": windowMaxTokens,
		"messages":   []map[string]any{{"role": "user", "content": b.String()}},
	})
}

var windowRedactions = []struct {
	re   *regexp.Regexp
	with string
}{
	{regexp.MustCompile(`req_[A-Za-z0-9]+`), "req_REDACTED"},
	{regexp.MustCompile(`(?i)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`), "00000000-0000-0000-0000-000000000000"},
	{regexp.MustCompile(`wrkspc_[A-Za-z0-9]+`), "wrkspc_REDACTED"},
	{regexp.MustCompile(`(?:user|org|account)-[A-Za-z0-9]+`), "user-REDACTED"},
	{regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`), "user@example.com"},
}

func redactWindowMessage(s string) string {
	for _, r := range windowRedactions {
		s = r.re.ReplaceAllString(s, r.with)
	}
	return s
}

// recordWindow sends the one request to base's /v1/messages with key, and
// returns the record. It sends exactly one request: no retry, and the
// client's redirects are not followed.
func recordWindow(ctx context.Context, client *http.Client, base, key, model string) (windowRecord, error) {
	body, err := windowBody(model)
	if err != nil {
		return windowRecord{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return windowRecord{}, err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("x-api-key", key)
	c := *client
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := c.Do(req)
	if err != nil {
		return windowRecord{}, fmt.Errorf("window measurement: the request failed in transport")
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return windowRecord{}, fmt.Errorf("window measurement: the reply could not be read")
	}
	var reply struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
		StopReason string `json:"stop_reason"`
	}
	_ = json.Unmarshal(raw, &reply) // a reply that is not JSON keeps only its status
	return windowRecord{
		Model: model, MaxTokens: windowMaxTokens, InputWords: windowWords,
		Status: resp.StatusCode, Type: reply.Error.Type,
		Message: redactWindowMessage(reply.Error.Message), StopReason: reply.StopReason,
	}, nil
}

func writeWindowRecord(path string, r windowRecord) error {
	raw, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o600)
}

// TestWindowRecord_Twin_0088 is the measurement's offline twin (0046-MADR
// D3): a server answering as Anthropic does with a request id, a workspace
// and a UUID in its message gets one request, carrying the version header,
// the key and max_tokens 32,000, and the record holds the status, the type
// and the message with each identifier redacted, and no other reply field.
func TestWindowRecord_Twin_0088(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		var body struct {
			MaxTokens int `json:"max_tokens"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if r.URL.Path != "/v1/messages" || r.Header.Get("x-api-key") != "twin-key" ||
			r.Header.Get("anthropic-version") != "2023-06-01" || body.MaxTokens != windowMaxTokens {
			w.WriteHeader(http.StatusTeapot)
			return
		}
		w.Header().Set("request-id", "req_abc123")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"type":"error","error":{"type":"invalid_request_error","message":"input length and `+
			"`max_tokens`"+` exceed context limit: 184000 + 32000 > 200000 (req_abc123, wrkspc_01xyz, 123e4567-e89b-12d3-a456-426614174000)"},"request_id":"req_abc123"}`)
	}))
	defer srv.Close()
	r, err := recordWindow(context.Background(), srv.Client(), srv.URL, "twin-key", "claude-twin")
	if err != nil {
		t.Fatal(err)
	}
	if got := n.Load(); got != 1 {
		t.Fatalf("requests = %d; want 1", got)
	}
	want := windowRecord{
		Model: "claude-twin", MaxTokens: windowMaxTokens, InputWords: windowWords,
		Status: http.StatusBadRequest, Type: "invalid_request_error",
		Message: "input length and `max_tokens` exceed context limit: 184000 + 32000 > 200000 (req_REDACTED, wrkspc_REDACTED, 00000000-0000-0000-0000-000000000000)",
	}
	if r != want {
		t.Fatalf("record = %+v; want %+v", r, want)
	}
	path := filepath.Join(t.TempDir(), "window-0088", "refusal.json")
	if err := writeWindowRecord(path, r); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "abc123") || strings.Contains(string(raw), "twin-key") {
		t.Fatalf("record holds an identifier or the key: %s", raw)
	}
}
