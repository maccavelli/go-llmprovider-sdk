package together

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/llmtest"
)

// TestConformance runs llmtest (0015-MADR D11) against Together.
func TestConformance(t *testing.T) {
	llmtest.Run(t, llmtest.Harness{
		Model: "llmtest-model",
		// The harness's Generate checks are of a plain request: each names its own
		// limit, since one whose limit the SDK chose or left out is read over the
		// stream (0088-MADR ST88, ST88e). The stream is checked through Stream.
		New: func(baseURL string, opts ...llmprovider.Option) (llmprovider.Provider, error) {
			return New(append([]llmprovider.Option{llmprovider.WithAPIKey("tg_llmtest"),
				llmprovider.WithModel("openai/gpt-oss-120b"), llmprovider.WithBaseURL(baseURL), llmprovider.WithMaxTokens(8192)}, opts...)...)
		},
		Text: func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, togetherText)
		},
		ToolCall: func(w http.ResponseWriter, _ *http.Request, tool string) {
			_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{
				"role": "assistant", "tool_calls": []any{map[string]any{"id": "call_llmtest", "type": "function",
					"function": map[string]any{"name": tool, "arguments": "{}"}}}}}}})
		},
		Fidelity:      true,
		SendsStrict:   true,
		StrictTools:   true,
		StreamText:    func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, llmtestChatStream) },
		StreamFailure: func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, llmtestChatStreamFailure) },
		Garbled:       func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, `{"garbled`) },
		Truncated: func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"model":"llmtest-model","choices":[{"finish_reason":"length","message":{"role":"assistant","tool_calls":[{"id":"call_llmtest","type":"function","function":{"name":"llmtest_tool","arguments":"{\"city\":"}}]}}]}`)
		},
		ReasoningCut: func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"model":"llmtest-model","choices":[{"finish_reason":"length","message":{"role":"assistant","content":"","reasoning":"thinking about it"}}]}`)
		},
		Error: func(w http.ResponseWriter, _ *http.Request, status int) {
			w.WriteHeader(status)
			_, _ = io.WriteString(w, `{"error":"llmtest"}`)
		},
	})
}

// llmtestChatStream is the text reply as a Chat Completions event stream: its
// text in two deltas, the finish, then [DONE] (0031-MADR E6).
const llmtestChatStream = "data: {\"id\":\"t-1\",\"model\":\"llmtest-model\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"he\"},\"finish_reason\":null}]}\n\ndata: {\"id\":\"t-1\",\"model\":\"llmtest-model\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"llo\"},\"finish_reason\":null}]}\n\ndata: {\"id\":\"t-1\",\"model\":\"llmtest-model\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n"

// llmtestChatStreamFailure is a Chat Completions event stream that fails
// after one delta, with an error inside the 200.
const llmtestChatStreamFailure = "data: {\"id\":\"t-1\",\"model\":\"llmtest-model\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"hel\"},\"finish_reason\":null}]}\n\ndata: {\"error\":{\"message\":\"llmtest\",\"code\":503}}\n\n"

// TestStreamTextDeltas_0051 (0051-MADR D5): the harness's StreamText script
// gives at least two text deltas through the provider's own Stream, so that
// llmtest's stream retry check also shows that the wrapper merges none.
func TestStreamTextDeltas_0051(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, llmtestChatStream)
	}))
	defer srv.Close()
	p, err := New(llmprovider.WithAPIKey("tg_llmtest"), llmprovider.WithModel("openai/gpt-oss-120b"),
		llmprovider.WithBaseURL(srv.URL))
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
