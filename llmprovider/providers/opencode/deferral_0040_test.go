package opencode

import (
	"context"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0040-MADR D1 and D2 on OpenCode's messages route: a Claude model defers
// natively; another model on the same route (MiniMax) appends its loads.

func deferralRequest() *llmprovider.Request {
	return &llmprovider.Request{
		Tools: []llmprovider.Tool{
			{Name: "read_file"},
			{Name: "tool_search", Search: true},
			{Name: "get_weather", Deferred: true},
			{Name: "lookup_stock", Deferred: true},
		},
		Input: []llmprovider.Item{
			llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "weather in Paris"},
			llmprovider.FunctionCallItem{CallID: "c1", Name: "tool_search", Arguments: `{"query":"weather"}`},
			llmprovider.FunctionCallOutputItem{CallID: "c1", Output: "Loaded get_weather."},
			llmprovider.ToolLoadItem{CallID: "c1", Name: "get_weather"},
		},
	}
}

// messagesSent is the messages-route body sent for req on model.
func messagesSent(t *testing.T, model string, req *llmprovider.Request) map[string]any {
	t.Helper()
	var body map[string]any
	srv := captureServer(t, &body, `{"content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn"}`)
	p := build(t, llmprovider.ProviderOpencodeZen, apiKey(srv.URL, model, WithRoute(RouteMessages))...)
	if _, err := p.Generate(context.Background(), req); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	return body
}

// TestDeferral_MessagesRoute_0040: claude-haiku-5-5 sends defer_loading and
// a tool_reference; minimax-m3 sends the loaded tool after the others, and
// the result as text.
func TestDeferral_MessagesRoute_0040(t *testing.T) {
	native := messagesSent(t, "claude-haiku-5-5", deferralRequest())
	var deferred []string
	for _, raw := range native["tools"].([]any) {
		if tool := raw.(map[string]any); tool["defer_loading"] == true {
			deferred = append(deferred, tool["name"].(string))
		}
	}
	if len(deferred) != 2 || len(native["tools"].([]any)) != 4 {
		t.Errorf("claude-haiku-5-5: deferred %v of %d tools; want get_weather and lookup_stock of 4", deferred, len(native["tools"].([]any)))
	}
	msgs := native["messages"].([]any)
	turn := msgs[len(msgs)-1].(map[string]any)["content"].([]any)
	result := turn[0].(map[string]any)
	refs, ok := result["content"].([]any)
	if !ok || len(refs) != 1 || refs[0].(map[string]any)["type"] != "tool_reference" || len(turn) != 2 {
		t.Errorf("claude-haiku-5-5: last turn %v; want a tool_reference result and a text block", turn)
	}

	appended := messagesSent(t, "minimax-m3", deferralRequest())
	var names []string
	for _, raw := range appended["tools"].([]any) {
		tool := raw.(map[string]any)
		if tool["defer_loading"] != nil {
			t.Errorf("minimax-m3: defer_loading sent on %v", tool["name"])
		}
		names = append(names, tool["name"].(string))
	}
	if len(names) != 3 || names[2] != "get_weather" {
		t.Errorf("minimax-m3: tools %v; want read_file, tool_search, get_weather", names)
	}
	msgs = appended["messages"].([]any)
	turn = msgs[len(msgs)-1].(map[string]any)["content"].([]any)
	if text, _ := turn[0].(map[string]any)["content"].(string); text != "Loaded get_weather." || len(turn) != 1 {
		t.Errorf("minimax-m3: last turn %v; want one result with the text", turn)
	}
}
