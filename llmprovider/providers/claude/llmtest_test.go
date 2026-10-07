package claude

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/llmtest"
)

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
		Fidelity:    true,
		StrictTools: true,
		Garbled:     func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, `{"garbled`) },
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
