package opencode

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/llmtest"
)

// TestConformance runs llmtest (0015-MADR D11) against each gateway, on the
// chat and messages routes (the two wire families OpenCode adds beyond the
// moved providers').
func TestConformance(t *testing.T) {
	for _, tc := range []struct {
		name    string
		gateway llmprovider.ProviderID
		model   string
		route   Route
	}{
		{"zen-chat", llmprovider.ProviderOpencodeZen, "glm-5.3", RouteChatCompletions},
		{"go-messages", llmprovider.ProviderOpencodeGo, "qwen3.8-flash", RouteMessages},
	} {
		t.Run(tc.name, func(t *testing.T) {
			llmtest.Run(t, llmtest.Harness{
				Model: "llmtest-model",
				New: func(baseURL string, opts ...llmprovider.Option) (llmprovider.Provider, error) {
					return newFor(tc.gateway)(append(append([]llmprovider.Option{llmprovider.WithAPIKey("opencode-llmtest"),
						llmprovider.WithModel(tc.model), llmprovider.WithBaseURL(baseURL), WithRoute(tc.route)}, opts...), metadataOff()...)...)
				},
				Text: func(w http.ResponseWriter, _ *http.Request) {
					if tc.route == RouteMessages {
						_, _ = io.WriteString(w, `{"model":"llmtest-model","stop_reason":"end_turn","content":[{"type":"text","text":"hello"}]}`)
						return
					}
					_, _ = io.WriteString(w, `{"model":"llmtest-model","choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"hello"}}]}`)
				},
				ToolCall: func(w http.ResponseWriter, _ *http.Request, tool string) {
					if tc.route == RouteMessages {
						_ = json.NewEncoder(w).Encode(map[string]any{"content": []any{map[string]any{
							"type": "tool_use", "id": "toolu_llmtest", "name": tool, "input": map[string]any{}}}})
						return
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{
						"role": "assistant", "tool_calls": []any{map[string]any{"id": "call_llmtest", "type": "function",
							"function": map[string]any{"name": tool, "arguments": "{}"}}}}}}})
				},
				Error: func(w http.ResponseWriter, _ *http.Request, status int) {
					w.WriteHeader(status)
					_, _ = io.WriteString(w, `{"type":"error","error":{"type":"llmtest","message":"llmtest"}}`)
				},
			})
		})
	}
}
