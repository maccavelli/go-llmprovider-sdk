package grok

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/llmtest"
)

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
				Fidelity:    true,
				StrictTools: true,
				// The Responses wire reports the service's own reason.
				TruncatedReason: "max_output_tokens",
				Garbled:         func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, `{"garbled`) },
				Truncated: func(w http.ResponseWriter, _ *http.Request) {
					_, _ = io.WriteString(w, `{"id":"resp_llmtest","status":"incomplete","incomplete_details":{"reason":"max_output_tokens"},"output":[{"type":"function_call","call_id":"call_llmtest","name":"llmtest_tool","arguments":"{\"city\":"}]}`)
				},
				Error: func(w http.ResponseWriter, _ *http.Request, status int) {
					w.WriteHeader(status)
					_, _ = io.WriteString(w, `{"error":{"code":"llmtest","message":"llmtest"}}`)
				},
			})
		})
	}
}
