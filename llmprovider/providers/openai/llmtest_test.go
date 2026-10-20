package openai

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

const (
	llmtestText = `{"id":"resp_llmtest","model":"llmtest-model","output":[{"type":"message","content":[{"type":"output_text","text":"hello"}]}]}`
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

func llmtestCall(tool string) string {
	call, _ := json.Marshal(map[string]any{"id": "resp_llmtest", "output": []any{map[string]any{
		"type": "function_call", "call_id": "call_llmtest", "name": tool, "arguments": "{}"}}})
	return string(call)
}

// harness wires the provider to llmtest's fake server. A ChatGPT session's
// replies are event streams.
func harness(credential func() llmprovider.Option, sse bool) llmtest.Harness {
	write := func(w http.ResponseWriter, body string) {
		if sse {
			body = stream(body)
		}
		_, _ = io.WriteString(w, body)
	}
	return llmtest.Harness{
		Model: "llmtest-model",
		New: func(baseURL string, opts ...llmprovider.Option) (llmprovider.Provider, error) {
			return New(append([]llmprovider.Option{credential(), llmprovider.WithModel("gpt-5.5"),
				llmprovider.WithBaseURL(baseURL)}, opts...)...)
		},
		Text:          func(w http.ResponseWriter, _ *http.Request) { write(w, llmtestText) },
		ToolCall:      func(w http.ResponseWriter, _ *http.Request, tool string) { write(w, llmtestCall(tool)) },
		Fidelity:      true,
		SendsStrict:   true,
		StrictTools:   true,
		StreamText:    func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, llmtestStream) },
		StreamFailure: func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, llmtestStreamFailure) },
		// The Responses wire reports the service's own reason.
		TruncatedReason: "max_output_tokens",
		// A session's replies are streams: a garbled event, and the
		// response.incomplete event a cut stream ends with.
		Garbled: func(w http.ResponseWriter, _ *http.Request) {
			if sse {
				_, _ = io.WriteString(w, "data: {\"garbled\n\n")
				return
			}
			_, _ = io.WriteString(w, `{"garbled`)
		},
		Truncated: func(w http.ResponseWriter, _ *http.Request) {
			if sse {
				_, _ = io.WriteString(w, `data: {"type":"response.output_item.done","item":{"type":"function_call","call_id":"call_llmtest","name":"llmtest_tool","arguments":"{\"city\":"}}`+"\n\n"+
					`data: {"type":"response.incomplete","response":{"id":"resp_llmtest","status":"incomplete","incomplete_details":{"reason":"max_output_tokens"}}}`+"\n\n")
				return
			}
			write(w, `{"id":"resp_llmtest","status":"incomplete","incomplete_details":{"reason":"max_output_tokens"},"output":[{"type":"function_call","call_id":"call_llmtest","name":"llmtest_tool","arguments":"{\"city\":"}]}`)
		},
		ReasoningCut: func(w http.ResponseWriter, _ *http.Request) {
			if sse {
				_, _ = io.WriteString(w, `data: {"type":"response.output_item.done","item":{"type":"reasoning","summary":[{"type":"summary_text","text":"thinking about it"}]}}`+"\n\n"+
					`data: {"type":"response.incomplete","response":{"id":"resp_llmtest","status":"incomplete","incomplete_details":{"reason":"max_output_tokens"}}}`+"\n\n")
				return
			}
			write(w, `{"id":"resp_llmtest","status":"incomplete","incomplete_details":{"reason":"max_output_tokens"},"output":[{"type":"reasoning","summary":[{"type":"summary_text","text":"thinking about it"}]}]}`)
		},
		// Codex's plan refusal is a 403 usage_not_included (MADR 0012 §1.1).
		NotPermitted: func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = io.WriteString(w, `{"error":{"type":"usage_not_included","message":"plan"}}`)
		},
		Error: func(w http.ResponseWriter, _ *http.Request, status int) {
			w.WriteHeader(status)
			_, _ = io.WriteString(w, `{"error":{"message":"llmtest"}}`)
		},
	}
}

// TestConformance runs llmtest (0015-MADR D11) with an API key and with a
// ChatGPT session. The session cannot refresh, so a 401 stays a 401.
func TestConformance(t *testing.T) {
	t.Run("api-key", func(t *testing.T) {
		llmtest.Run(t, harness(func() llmprovider.Option { return llmprovider.WithAPIKey("sk-llmtest") }, false))
	})
	t.Run("chatgpt", func(t *testing.T) {
		h := harness(func() llmprovider.Option {
			return llmprovider.WithTokenSource(&auth.OAuthSession{Issuer: auth.DefaultOpenAIIssuer,
				Access: "chatgpt-access", Expiry: time.Now().Add(time.Hour), TokenURL: "http://127.0.0.1:1/never-refresh"})
		}, true)
		// The session makes this the ChatGPT provider: the check's own
		// source would build an API-key one. The session's 401 refresh is
		// TestOpenAI_OAuth401RetriesOnceAfterRefresh.
		h.NoReauth = "the ChatGPT session decides the mode"
		llmtest.Run(t, h)
	})
}

// TestStreamTextDeltas_0051 (0051-MADR D5): the harness's StreamText script
// gives at least two text deltas through the provider's own Stream, so that
// llmtest's stream retry check also shows that the wrapper merges none.
func TestStreamTextDeltas_0051(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, llmtestStream)
	}))
	defer srv.Close()
	p, err := New(llmprovider.WithAPIKey("sk-llmtest"), llmprovider.WithModel("gpt-5.5"), llmprovider.WithBaseURL(srv.URL))
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
