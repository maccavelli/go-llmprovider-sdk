package claude

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/llmtest"
)

// llmtestMessagesStream is the text reply as a Messages event stream: its
// text in a delta, then the stop (0031-MADR E6).
const llmtestMessagesStream = "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"model\":\"llmtest-model\"}}\n\n" +
	"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n" +
	"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"hello\"}}\n\n" +
	"event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n" +
	"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"}}\n\n" +
	"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"

// llmtestMessagesStreamFailure is a Messages event stream that fails after
// one delta.
const llmtestMessagesStreamFailure = "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"model\":\"llmtest-model\"}}\n\n" +
	"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n" +
	"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"hel\"}}\n\n" +
	"event: error\ndata: {\"type\":\"error\",\"error\":{\"type\":\"overloaded_error\",\"message\":\"llmtest\"}}\n\n"

// TestConformance runs llmtest (0015-MADR D11) against Claude.
func TestConformance(t *testing.T) {
	llmtest.Run(t, llmtest.Harness{
		Model: "llmtest-model",
		New: func(baseURL string, opts ...llmprovider.Option) (llmprovider.Provider, error) {
			return New(append([]llmprovider.Option{llmprovider.WithAPIKey("sk-ant-llmtest"),
				llmprovider.WithModel("claude-haiku-4-5"), llmprovider.WithBaseURL(baseURL)}, opts...)...)
		},
		Text: func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"model":"llmtest-model","stop_reason":"end_turn","content":[{"type":"text","text":"hello"}]}`)
		},
		ToolCall: func(w http.ResponseWriter, _ *http.Request, tool string) {
			_ = json.NewEncoder(w).Encode(map[string]any{"content": []any{map[string]any{
				"type": "tool_use", "id": "toolu_llmtest", "name": tool, "input": map[string]any{}}}})
		},
		Fidelity:      true,
		SendsStrict:   true, // 0034-MADR D4
		SendsIsError:  true,
		StrictTools:   true,
		StreamText:    func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, llmtestMessagesStream) },
		StreamFailure: func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, llmtestMessagesStreamFailure) },
		Garbled:       func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, `{"garbled`) },
		Truncated: func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"model":"llmtest-model","stop_reason":"max_tokens","content":[{"type":"tool_use","id":"toolu_llmtest","name":"llmtest_tool","input":{}}]}`)
		},
		ReasoningCut: func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"model":"llmtest-model","stop_reason":"max_tokens","content":[{"type":"thinking","thinking":"thinking about it","signature":"sig"}]}`)
		},
		Error: func(w http.ResponseWriter, _ *http.Request, status int) {
			w.WriteHeader(status)
			_, _ = io.WriteString(w, `{"type":"error","error":{"type":"llmtest","message":"llmtest"}}`)
		},
	})
}
