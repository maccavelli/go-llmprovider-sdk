package ollama

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

// TestConformance runs llmtest (0015-MADR D11) against Ollama.
func TestConformance(t *testing.T) {
	llmtest.Run(t, llmtest.Harness{
		Model: "llama3.2:latest",
		New: func(baseURL string, opts ...llmprovider.Option) (llmprovider.Provider, error) {
			return New(append([]llmprovider.Option{llmprovider.WithModel("llama3.3"), llmprovider.WithBaseURL(baseURL)}, opts...)...)
		},
		Text: func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, fxOllamaChat)
		},
		ToolCall: func(w http.ResponseWriter, _ *http.Request, tool string) {
			_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{
				"role": "assistant", "tool_calls": []any{map[string]any{"id": "call_llmtest", "type": "function",
					"function": map[string]any{"name": tool, "arguments": "{}"}}}}}}})
		},
		Fidelity:      true,
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
const llmtestChatStream = "data: {\"id\":\"chatcmpl-1\",\"model\":\"llama3.2:latest\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"AL\"},\"finish_reason\":null}]}\n\ndata: {\"id\":\"chatcmpl-1\",\"model\":\"llama3.2:latest\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"PHA\"},\"finish_reason\":null}]}\n\ndata: {\"id\":\"chatcmpl-1\",\"model\":\"llama3.2:latest\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n"

// llmtestChatStreamFailure is a Chat Completions event stream that fails
// after one delta, with an error inside the 200.
const llmtestChatStreamFailure = "data: {\"id\":\"chatcmpl-1\",\"model\":\"llama3.2:latest\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"ALP\"},\"finish_reason\":null}]}\n\ndata: {\"error\":{\"message\":\"llmtest\",\"code\":503}}\n\n"

// TestStreamTextDeltas_0051 (0051-MADR D5): the harness's StreamText script
// gives at least two text deltas through the provider's own Stream, so that
// llmtest's stream retry check also shows that the wrapper merges none.
func TestStreamTextDeltas_0051(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, llmtestChatStream)
	}))
	defer srv.Close()
	p, err := New(llmprovider.WithModel("llama3.3"), llmprovider.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	assertTwoDeltas0051(t, p, "ALPHA")
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
