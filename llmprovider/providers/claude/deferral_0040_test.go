package claude

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0040-MADR D2, A1 and D6 through the provider: a model that defers natively
// sends defer_loading and tool_reference; an older one appends its loads.

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
		Cache: llmprovider.CacheShort,
	}
}

// sent is the request body the provider sent for req on model.
func sent(t *testing.T, model string, req *llmprovider.Request) map[string]any {
	t.Helper()
	var body map[string]any
	srv := captureServer(t, &body, `{"content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn"}`)
	p := build(t, apiKey(srv.URL, model)...)
	if _, err := p.Generate(context.Background(), req); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	return body
}

func toolNames(body map[string]any) (names []string, deferred map[string]bool, cached string) {
	deferred = map[string]bool{}
	for _, raw := range body["tools"].([]any) {
		tool := raw.(map[string]any)
		name := tool["name"].(string)
		names = append(names, name)
		if tool["defer_loading"] == true {
			deferred[name] = true
		}
		if _, ok := tool["cache_control"]; ok {
			cached = name
		}
	}
	return names, deferred, cached
}

// lastUserTurn is the JSON of the body's last message's content.
func lastUserTurn(t *testing.T, body map[string]any) string {
	t.Helper()
	msgs := body["messages"].([]any)
	b, err := json.Marshal(msgs[len(msgs)-1].(map[string]any)["content"])
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestDeferral_Native_0040: claude-haiku-5-5 sends every tool, the deferred
// ones with defer_loading, the cache marker on the last tool that is not
// deferred, and the load as a tool_reference with its text beside it.
func TestDeferral_Native_0040(t *testing.T) {
	body := sent(t, "claude-haiku-5-5", deferralRequest())
	names, deferred, cached := toolNames(body)
	if len(names) != 4 || !deferred["get_weather"] || !deferred["lookup_stock"] || deferred["tool_search"] || deferred["read_file"] {
		t.Errorf("tools %v, deferred %v; want all four, the last two deferred", names, deferred)
	}
	if cached != "tool_search" {
		t.Errorf("cache marker on %q, want tool_search, the last tool not deferred", cached)
	}
	want := `[{"content":[{"tool_name":"get_weather","type":"tool_reference"}],"tool_use_id":"c1","type":"tool_result"},` +
		`{"text":"Loaded get_weather.","type":"text"}]`
	if got := lastUserTurn(t, body); got != want {
		t.Errorf("last turn:\n got  %s\n want %s", got, want)
	}
}

// TestDeferral_Append_0040: claude-opus-4-1, which has no tool search, gets
// the tools that are not deferred and then the loaded one, no defer_loading,
// and the result as plain text.
func TestDeferral_Append_0040(t *testing.T) {
	body := sent(t, "claude-opus-4-1", deferralRequest())
	names, deferred, cached := toolNames(body)
	if len(deferred) != 0 {
		t.Errorf("defer_loading sent on %v", deferred)
	}
	if got := len(names); got != 3 || names[0] != "read_file" || names[1] != "tool_search" || names[2] != "get_weather" {
		t.Errorf("tools %v, want read_file, tool_search, get_weather", names)
	}
	if cached != "get_weather" {
		t.Errorf("cache marker on %q, want the last tool sent, get_weather", cached)
	}
	want := `[{"content":"Loaded get_weather.","tool_use_id":"c1","type":"tool_result"}]`
	if got := lastUserTurn(t, body); got != want {
		t.Errorf("last turn:\n got  %s\n want %s", got, want)
	}
}

// TestDeferral_None_0040: a request with nothing deferred is sent as before,
// on either model.
func TestDeferral_None_0040(t *testing.T) {
	req := &llmprovider.Request{Tools: []llmprovider.Tool{{Name: "read_file"}},
		Input: []llmprovider.Item{llmprovider.MessageItem{Text: "hi"}}}
	for _, model := range []string{"claude-haiku-5-5", "claude-opus-4-1"} {
		names, deferred, _ := toolNames(sent(t, model, req))
		if len(names) != 1 || len(deferred) != 0 {
			t.Errorf("%s: tools %v, deferred %v", model, names, deferred)
		}
	}
}
