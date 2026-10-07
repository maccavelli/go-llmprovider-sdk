package opencode

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/llmtest"
)

// routeReplies are one route's replies in its own wire's shape.
type routeReplies struct {
	text, truncated, reasoningCut string
	toolCall                      func(tool string) any
	// truncatedReason is the Reason the route's wire reports for a cut
	// answer; empty is "length".
	truncatedReason string
}

// conformanceReplies holds each route's replies. The google and responses
// routes run the generateContent and Responses wires, which no other provider
// runs through llmtest (0026-MADR F56).
var conformanceReplies = map[Route]routeReplies{
	RouteChatCompletions: {
		text:      `{"model":"llmtest-model","choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"hello"}}]}`,
		truncated: `{"model":"llmtest-model","choices":[{"finish_reason":"length","message":{"role":"assistant","tool_calls":[{"id":"call_llmtest","type":"function","function":{"name":"llmtest_tool","arguments":"{\"city\":"}}]}}]}`,
		reasoningCut: `{"model":"llmtest-model","choices":[{"finish_reason":"length","message":{"role":"assistant","content":"",` +
			`"reasoning_content":"thinking about it"}}]}`,
		toolCall: func(tool string) any {
			return map[string]any{"choices": []any{map[string]any{"message": map[string]any{
				"role": "assistant", "tool_calls": []any{map[string]any{"id": "call_llmtest", "type": "function",
					"function": map[string]any{"name": tool, "arguments": "{}"}}}}}}}
		},
	},
	RouteMessages: {
		text:         `{"model":"llmtest-model","stop_reason":"end_turn","content":[{"type":"text","text":"hello"}]}`,
		truncated:    `{"model":"llmtest-model","stop_reason":"max_tokens","content":[{"type":"tool_use","id":"toolu_llmtest","name":"llmtest_tool","input":{}}]}`,
		reasoningCut: `{"model":"llmtest-model","stop_reason":"max_tokens","content":[{"type":"thinking","thinking":"thinking about it","signature":"sig"}]}`,
		toolCall: func(tool string) any {
			return map[string]any{"content": []any{map[string]any{
				"type": "tool_use", "id": "toolu_llmtest", "name": tool, "input": map[string]any{}}}}
		},
	},
	RouteGoogle: {
		text: `{"modelVersion":"llmtest-model","candidates":[{"finishReason":"STOP","content":{"parts":[{"text":"hello"}]}}]}`,
		truncated: `{"modelVersion":"llmtest-model","candidates":[{"finishReason":"MAX_TOKENS","content":{"parts":[` +
			`{"functionCall":{"name":"llmtest_tool","args":{}}}]}}]}`,
		reasoningCut: `{"modelVersion":"llmtest-model","candidates":[{"finishReason":"MAX_TOKENS","content":{"parts":[` +
			`{"text":"thinking about it","thought":true}]}}]}`,
		toolCall: func(tool string) any {
			return map[string]any{"modelVersion": "llmtest-model", "candidates": []any{map[string]any{
				"finishReason": "STOP", "content": map[string]any{"parts": []any{map[string]any{
					"functionCall": map[string]any{"name": tool, "args": map[string]any{}}}}}}}}
		},
	},
	RouteResponses: {
		text: `{"id":"resp_llmtest","model":"llmtest-model","output":[{"type":"message","content":[{"type":"output_text","text":"hello"}]}]}`,
		truncated: `{"id":"resp_llmtest","status":"incomplete","incomplete_details":{"reason":"max_output_tokens"},` +
			`"output":[{"type":"function_call","call_id":"call_llmtest","name":"llmtest_tool","arguments":"{\"city\":"}]}`,
		reasoningCut: `{"id":"resp_llmtest","status":"incomplete","incomplete_details":{"reason":"max_output_tokens"},` +
			`"output":[{"type":"reasoning","summary":[{"type":"summary_text","text":"thinking about it"}]}]}`,
		toolCall: func(tool string) any {
			return map[string]any{"id": "resp_llmtest", "output": []any{map[string]any{
				"type": "function_call", "call_id": "call_llmtest", "name": tool, "arguments": "{}"}}}
		},
		// The Responses wire reports the service's own reason.
		truncatedReason: "max_output_tokens",
	},
}

// TestConformance runs llmtest (0015-MADR D11) against each gateway, on every
// route: chat completions and messages, and google and responses (0026-MADR
// F56).
func TestConformance(t *testing.T) {
	for _, tc := range []struct {
		name    string
		gateway llmprovider.ProviderID
		model   string
		route   Route
	}{
		{"zen-chat", llmprovider.ProviderOpencodeZen, "glm-5.3", RouteChatCompletions},
		{"go-messages", llmprovider.ProviderOpencodeGo, "qwen3.8-flash", RouteMessages},
		{"zen-google", llmprovider.ProviderOpencodeZen, "gemini-3.8-flash", RouteGoogle},
		{"go-responses", llmprovider.ProviderOpencodeGo, "gpt-6-luna", RouteResponses},
	} {
		t.Run(tc.name, func(t *testing.T) {
			replies := conformanceReplies[tc.route]
			write := func(body string) func(http.ResponseWriter, *http.Request) {
				return func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, body) }
			}
			llmtest.Run(t, llmtest.Harness{
				Model: "llmtest-model",
				New: func(baseURL string, opts ...llmprovider.Option) (llmprovider.Provider, error) {
					return newFor(tc.gateway)(append(append([]llmprovider.Option{llmprovider.WithAPIKey("opencode-llmtest"),
						llmprovider.WithModel(tc.model), llmprovider.WithBaseURL(baseURL), WithRoute(tc.route)}, opts...), metadataOff()...)...)
				},
				Text: write(replies.text),
				ToolCall: func(w http.ResponseWriter, _ *http.Request, tool string) {
					_ = json.NewEncoder(w).Encode(replies.toolCall(tool))
				},
				Fidelity:        true,
				StrictTools:     true,
				Garbled:         write(`{"garbled`),
				Truncated:       write(replies.truncated),
				TruncatedReason: replies.truncatedReason,
				ReasoningCut:    write(replies.reasoningCut),
				// OpenCode refuses a model or a route with 403 (0012-MADR,
				// amendment 2026-10-05).
				NotPermitted: func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusForbidden)
					_, _ = io.WriteString(w, `{"type":"error","error":{"type":"error","message":"Your organization does not have access to this model"}}`)
				},
				Error: func(w http.ResponseWriter, _ *http.Request, status int) {
					w.WriteHeader(status)
					_, _ = io.WriteString(w, `{"type":"error","error":{"type":"llmtest","message":"llmtest"}}`)
				},
			})
		})
	}
}
