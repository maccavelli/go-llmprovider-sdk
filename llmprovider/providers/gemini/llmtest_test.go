package gemini

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/llmtest"
)

// TestConformance runs llmtest (0015-MADR D11) against Gemini, with and
// without stored interactions.
func TestConformance(t *testing.T) {
	for name, extra := range map[string][]llmprovider.Option{"unstored": nil, "stored": {WithStore(true)}} {
		t.Run(name, func(t *testing.T) {
			llmtest.Run(t, llmtest.Harness{
				Model: "llmtest-model",
				New: func(baseURL string, opts ...llmprovider.Option) (llmprovider.Provider, error) {
					base := []llmprovider.Option{llmprovider.WithAPIKey("gemini-llmtest"),
						llmprovider.WithModel("gemini-3.7-flash"), llmprovider.WithBaseURL(baseURL)}
					return New(append(append(base, extra...), opts...)...)
				},
				Text: func(w http.ResponseWriter, _ *http.Request) {
					_, _ = io.WriteString(w, `{"id":"v1_llmtest","model":"llmtest-model","status":"completed","steps":[`+
						`{"type":"model_output","content":[{"type":"text","text":"hello"}]}]}`)
				},
				ToolCall: func(w http.ResponseWriter, _ *http.Request, tool string) {
					_ = json.NewEncoder(w).Encode(map[string]any{"id": "v1_llmtest", "status": "requires_action",
						"steps": []any{map[string]any{"type": "function_call", "id": "call_llmtest", "name": tool,
							"arguments": map[string]any{}}}})
				},
				Fidelity:     true,
				SendsIsError: true,
				StrictTools:  true,
				StreamText:   func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, llmtestInteractionStream) },
				StreamFailure: func(w http.ResponseWriter, _ *http.Request) {
					_, _ = io.WriteString(w, llmtestInteractionStreamFailure)
				},
				// An Interaction gives no reason; the decoder reports its
				// status (interactions.go).
				TruncatedReason: "incomplete",
				Garbled:         func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, `{"garbled`) },
				Truncated: func(w http.ResponseWriter, _ *http.Request) {
					_, _ = io.WriteString(w, `{"id":"v1_llmtest","status":"incomplete","steps":[{"type":"function_call","id":"call_llmtest","name":"llmtest_tool","arguments":{}}]}`)
				},
				ReasoningCut: func(w http.ResponseWriter, _ *http.Request) {
					_, _ = io.WriteString(w, `{"id":"v1_llmtest","status":"incomplete","steps":[{"type":"thought","signature":"sig","summary":[{"type":"text","text":"thinking about it"}]}]}`)
				},
				Error: func(w http.ResponseWriter, _ *http.Request, status int) {
					w.WriteHeader(status)
					_, _ = io.WriteString(w, `{"error":{"code":"llmtest","message":"llmtest"}}`)
				},
				// Gemini refuses a key with 400 API_KEY_INVALID, measured
				// live (0020-MADR F23 amendment; 0026-MADR F6), and the
				// Interactions API wraps the error in an array (0026-PLAN D12).
				AuthFailure: func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusBadRequest)
					_, _ = io.WriteString(w, `[{"error":{"code":400,"message":"API key not valid. Please pass a valid API key.",`+
						`"status":"INVALID_ARGUMENT","details":[{"@type":"type.googleapis.com/google.rpc.ErrorInfo",`+
						`"reason":"API_KEY_INVALID","domain":"googleapis.com"}]}}]`)
				},
			})
		})
	}
}

// llmtestInteractionStream is the text reply as an Interactions event stream
// (0031-MADR E6).
const llmtestInteractionStream = "event: interaction.created\ndata: {\"event_type\":\"interaction.created\",\"interaction\":{\"id\":\"v1_llmtest\",\"model\":\"llmtest-model\",\"status\":\"in_progress\"}}\n\nevent: step.start\ndata: {\"event_type\":\"step.start\",\"index\":0,\"step\":{\"type\":\"model_output\"}}\n\nevent: step.delta\ndata: {\"event_type\":\"step.delta\",\"index\":0,\"delta\":{\"type\":\"text\",\"text\":\"hello\"}}\n\nevent: step.stop\ndata: {\"event_type\":\"step.stop\",\"index\":0}\n\nevent: interaction.completed\ndata: {\"event_type\":\"interaction.completed\",\"interaction\":{\"id\":\"v1_llmtest\",\"model\":\"llmtest-model\",\"status\":\"completed\"}}\n\nevent: done\ndata: [DONE]\n\n"

// llmtestInteractionStreamFailure is an Interactions event stream that fails
// after one delta.
const llmtestInteractionStreamFailure = "event: interaction.created\ndata: {\"event_type\":\"interaction.created\",\"interaction\":{\"id\":\"v1_llmtest\",\"model\":\"llmtest-model\",\"status\":\"in_progress\"}}\n\nevent: step.start\ndata: {\"event_type\":\"step.start\",\"index\":0,\"step\":{\"type\":\"model_output\"}}\n\nevent: step.delta\ndata: {\"event_type\":\"step.delta\",\"index\":0,\"delta\":{\"type\":\"text\",\"text\":\"hel\"}}\n\nevent: error\ndata: {\"event_type\":\"error\",\"error\":{\"code\":503,\"message\":\"llmtest\",\"status\":\"UNAVAILABLE\"}}\n\n"
