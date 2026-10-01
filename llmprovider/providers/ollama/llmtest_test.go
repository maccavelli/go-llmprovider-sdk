package ollama

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/llmtest"
)

// TestConformance runs llmtest (0015-MADR D11) against Ollama.
func TestConformance(t *testing.T) {
	llmtest.Run(t, llmtest.Harness{
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
		Error: func(w http.ResponseWriter, _ *http.Request, status int) {
			w.WriteHeader(status)
			_, _ = io.WriteString(w, `{"error":"llmtest"}`)
		},
	})
}
