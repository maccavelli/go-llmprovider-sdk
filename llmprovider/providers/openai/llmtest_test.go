package openai

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/llmtest"
)

const (
	llmtestText = `{"id":"resp_llmtest","output":[{"type":"message","content":[{"type":"output_text","text":"hello"}]}]}`
)

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
		New: func(baseURL string, opts ...llmprovider.Option) (llmprovider.Provider, error) {
			return New(append([]llmprovider.Option{credential(), llmprovider.WithModel("gpt-5.5"),
				llmprovider.WithBaseURL(baseURL)}, opts...)...)
		},
		Text:     func(w http.ResponseWriter, _ *http.Request) { write(w, llmtestText) },
		ToolCall: func(w http.ResponseWriter, _ *http.Request, tool string) { write(w, llmtestCall(tool)) },
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
		llmtest.Run(t, harness(func() llmprovider.Option {
			return llmprovider.WithTokenSource(&llmprovider.OAuthSession{Issuer: llmprovider.DefaultOpenAIIssuer,
				Access: "chatgpt-access", Expiry: time.Now().Add(time.Hour), TokenURL: "http://127.0.0.1:1/never-refresh"})
		}, true))
	})
}
