package grok

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/llmtest"
)

// llmtestStream is the text reply as a Responses event stream: its text in two
// deltas, the item, then completed (0031-MADR D7).
const llmtestStream = "data: {\"type\":\"response.output_text.delta\",\"delta\":\"hel\"}\n\n" +
	"data: {\"type\":\"response.output_text.delta\",\"delta\":\"lo\"}\n\n" +
	"data: {\"type\":\"response.output_item.done\",\"item\":{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"hello\"}]}}\n\n" +
	"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_llmtest\",\"model\":\"llmtest-model\"}}\n\n"

// llmtestStreamFailure is a Responses event stream that fails after one
// delta.
const llmtestStreamFailure = "data: {\"type\":\"response.output_text.delta\",\"delta\":\"hel\"}\n\n" +
	"data: {\"type\":\"response.failed\",\"response\":{\"error\":{\"code\":\"server_error\",\"message\":\"llmtest\"}}}\n\n"

// TestConformance runs llmtest (0015-MADR D11) against Grok, with an API key
// and with an OAuth session.
func TestConformance(t *testing.T) {
	for name, src := range map[string]func() llmprovider.TokenSource{
		"api key": func() llmprovider.TokenSource { return llmprovider.NewStaticToken("xai-llmtest") },
		"session": func() llmprovider.TokenSource {
			return &auth.OAuthSession{Provider: llmprovider.ProviderGrok, Access: "a", Expiry: time.Now().Add(time.Hour)}
		},
	} {
		t.Run(name, func(t *testing.T) {
			llmtest.Run(t, llmtest.Harness{
				Model: "llmtest-model",
				New: func(baseURL string, opts ...llmprovider.Option) (llmprovider.Provider, error) {
					return New(append([]llmprovider.Option{llmprovider.WithTokenSource(src()),
						llmprovider.WithModel("grok-4.5"), llmprovider.WithBaseURL(baseURL)}, opts...)...)
				},
				Text: func(w http.ResponseWriter, _ *http.Request) {
					_, _ = io.WriteString(w, `{"id":"resp_llmtest","model":"llmtest-model","output":[{"type":"message","content":[{"type":"output_text","text":"hello"}]}]}`)
				},
				ToolCall: func(w http.ResponseWriter, _ *http.Request, tool string) {
					_ = json.NewEncoder(w).Encode(map[string]any{"id": "resp_llmtest", "output": []any{map[string]any{
						"type": "function_call", "call_id": "call_llmtest", "name": tool, "arguments": "{}"}}})
				},
				Fidelity:      true,
				SendsStrict:   true,
				StrictTools:   true,
				StreamText:    func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, llmtestStream) },
				StreamFailure: func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, llmtestStreamFailure) },
				// The Responses wire reports the service's own reason.
				TruncatedReason: "max_output_tokens",
				Garbled:         func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, `{"garbled`) },
				Truncated: func(w http.ResponseWriter, _ *http.Request) {
					_, _ = io.WriteString(w, `{"id":"resp_llmtest","status":"incomplete","incomplete_details":{"reason":"max_output_tokens"},"output":[{"type":"function_call","call_id":"call_llmtest","name":"llmtest_tool","arguments":"{\"city\":"}]}`)
				},
				ReasoningCut: func(w http.ResponseWriter, _ *http.Request) {
					_, _ = io.WriteString(w, `{"id":"resp_llmtest","status":"incomplete","incomplete_details":{"reason":"max_output_tokens"},"output":[{"type":"reasoning","summary":[{"type":"summary_text","text":"thinking about it"}]}]}`)
				},
				Error: func(w http.ResponseWriter, _ *http.Request, status int) {
					w.WriteHeader(status)
					_, _ = io.WriteString(w, `{"error":{"code":"llmtest","message":"llmtest"}}`)
				},
			})
		})
	}
}

// TestStreamTextDeltas_0051 (0051-MADR D5): the harness's StreamText script
// gives at least two text deltas through the provider's own Stream, so that
// llmtest's stream retry check also shows that the wrapper merges none.
func TestStreamTextDeltas_0051(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, llmtestStream)
	}))
	defer srv.Close()
	p, err := New(llmprovider.WithAPIKey("xai-llmtest"), llmprovider.WithModel("grok-4.5"), llmprovider.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	assertTwoDeltas0051(t, p, "hello")
}

// assertTwoDeltas0051 streams a request from p and checks that its text came
// in two or more text deltas, joined equal to want.
func assertTwoDeltas0051(t *testing.T, p llmprovider.Provider, want string) {
	t.Helper()
	deltas, text := 0, ""
	for ev, err := range llmprovider.Stream(context.Background(), p, &llmprovider.Request{
		Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"}},
	}) {
		if err != nil {
			t.Fatalf("Stream: %v", err)
		}
		if ev.Type == llmprovider.EventTextDelta {
			deltas++
			text += ev.Text
		}
	}
	if deltas < 2 || text != want {
		t.Errorf("StreamText gave %d text deltas, joined %q; want two or more, joined %q", deltas, text, want)
	}
}
