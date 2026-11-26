package chatcompat

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/llmtest"
)

// TestConformance runs llmtest (0015-MADR D11; R44) against a chatcompat
// provider.
func TestConformance(t *testing.T) {
	llmtest.Run(t, llmtest.Harness{
		Model: "llmtest-model",
		// The harness's Generate checks are of a plain request: each names its own
		// limit, since one whose limit the SDK chose or left out is read over the
		// stream (0088-MADR ST88, ST88e). The stream is checked through Stream.
		New: func(baseURL string, opts ...llmprovider.Option) (llmprovider.Provider, error) {
			return New(append([]llmprovider.Option{testCache(t), WithID(testID), llmprovider.WithAPIKey("sk-llmtest"),
				llmprovider.WithModel("llmtest-model"), llmprovider.WithBaseURL(baseURL), llmprovider.WithMaxTokens(8192)}, opts...)...)
		},
		Text: func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, fxChat)
		},
		ToolCall: func(w http.ResponseWriter, _ *http.Request, tool string) {
			_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{
				"role": "assistant", "tool_calls": []any{map[string]any{"id": "call_llmtest", "type": "function",
					"function": map[string]any{"name": tool, "arguments": "{}"}}}}}}})
		},
		Fidelity:      true,
		SendsStrict:   true,
		StrictTools:   true,
		StreamText:    func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, chatStream) },
		StreamFailure: func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, chatStreamFailure) },
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

// chatStream is the text reply as a Chat Completions event stream: its text
// in two deltas, the finish, then [DONE] (0031-MADR E6).
const chatStream = "data: {\"id\":\"cmpl-acme\",\"model\":\"llmtest-model\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"he\"},\"finish_reason\":null}]}\n\ndata: {\"id\":\"cmpl-acme\",\"model\":\"llmtest-model\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"llo\"},\"finish_reason\":null}]}\n\ndata: {\"id\":\"cmpl-acme\",\"model\":\"llmtest-model\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n"

// chatStreamFailure is a Chat Completions event stream that fails after one
// delta, with an error inside the 200.
const chatStreamFailure = "data: {\"id\":\"cmpl-acme\",\"model\":\"llmtest-model\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"hel\"},\"finish_reason\":null}]}\n\ndata: {\"error\":{\"message\":\"llmtest\",\"code\":503}}\n\n"

// TestConformance_Responses_0091 runs llmtest (R44) against a chatcompat
// provider built with WithResponses, over a scripted Responses server
// (0091-MADR ST59): its capabilities are unchanged by the option, the same
// with and without it (D6).
func TestConformance_Responses_0091(t *testing.T) {
	chat := build(t, "http://127.0.0.1:1").Capabilities()
	if got := build(t, "http://127.0.0.1:1", WithResponses()).Capabilities(); got != chat {
		t.Errorf("capabilities with WithResponses %+v; want the Chat path's %+v", got, chat)
	}
	write := func(body string) func(http.ResponseWriter, *http.Request) {
		return func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, body) }
	}
	llmtest.Run(t, llmtest.Harness{
		Model: "llmtest-model",
		// Each Generate names its limit, so it is a plain request; the stream
		// is checked through Stream (0088-MADR ST88).
		New: func(baseURL string, opts ...llmprovider.Option) (llmprovider.Provider, error) {
			return New(append([]llmprovider.Option{testCache(t), WithID(testID), WithResponses(), llmprovider.WithAPIKey("sk-llmtest"),
				llmprovider.WithModel("llmtest-model"), llmprovider.WithBaseURL(baseURL), llmprovider.WithMaxTokens(8192)}, opts...)...)
		},
		Text: write(`{"id":"resp_llmtest","model":"llmtest-model","output":[{"type":"message","content":[{"type":"output_text","text":"hello"}]}]}`),
		ToolCall: func(w http.ResponseWriter, _ *http.Request, tool string) {
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "resp_llmtest", "output": []any{map[string]any{
				"type": "function_call", "call_id": "call_llmtest", "name": tool, "arguments": "{}"}}})
		},
		Fidelity:    true,
		SendsStrict: true,
		StrictTools: true,
		StreamText: write("data: {\"type\":\"response.output_text.delta\",\"delta\":\"hel\"}\n\n" +
			"data: {\"type\":\"response.output_text.delta\",\"delta\":\"lo\"}\n\n" +
			"data: {\"type\":\"response.output_item.done\",\"item\":{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"hello\"}]}}\n\n" +
			"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_llmtest\",\"model\":\"llmtest-model\"}}\n\n"),
		StreamFailure: write("data: {\"type\":\"response.output_text.delta\",\"delta\":\"hel\"}\n\n" +
			"data: {\"type\":\"response.failed\",\"response\":{\"error\":{\"code\":\"server_error\",\"message\":\"llmtest\"}}}\n\n"),
		Garbled: write(`{"garbled`),
		Truncated: write(`{"id":"resp_llmtest","status":"incomplete","incomplete_details":{"reason":"max_output_tokens"},` +
			`"output":[{"type":"function_call","call_id":"call_llmtest","name":"llmtest_tool","arguments":"{\"city\":"}]}`),
		TruncatedReason: "max_output_tokens",
		ReasoningCut: write(`{"id":"resp_llmtest","status":"incomplete","incomplete_details":{"reason":"max_output_tokens"},` +
			`"output":[{"type":"reasoning","summary":[{"type":"summary_text","text":"thinking about it"}]}]}`),
		Error: func(w http.ResponseWriter, _ *http.Request, status int) {
			w.WriteHeader(status)
			_, _ = io.WriteString(w, `{"error":{"message":"llmtest"}}`)
		},
	})
}
