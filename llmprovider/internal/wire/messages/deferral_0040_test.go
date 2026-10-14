package messages

import (
	"encoding/json"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0040-MADR D2 and A1: a load is a tool_reference inside its result, and the
// result's text follows the turn's results as its own block, the form the
// Phase 0 probe measured. 0039's breakpoint skips deferred tools.

func marshal(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestFromItemsLoading_Load_0040: one result loading two tools, beside a
// plain result in the same turn. The text block follows every result.
func TestFromItemsLoading_Load_0040(t *testing.T) {
	items := []llmprovider.Item{
		llmprovider.MessageItem{Text: "hi"},
		llmprovider.FunctionCallItem{CallID: "c1", Name: "tool_search", Arguments: `{"q":"w"}`},
		llmprovider.FunctionCallItem{CallID: "c2", Name: "read_file", Arguments: `{}`},
		llmprovider.FunctionCallOutputItem{CallID: "c1", Output: "Loaded two."},
		llmprovider.ToolLoadItem{CallID: "c1", Name: "get_weather"},
		llmprovider.ToolLoadItem{CallID: "c1", Name: "lookup_stock"},
		llmprovider.FunctionCallOutputItem{CallID: "c2", Output: "file"},
		llmprovider.MessageItem{Role: llmprovider.RoleAssistant, Text: "ok"},
	}
	got := marshal(t, FromItemsLoading(items))
	want := `[{"content":"hi","role":"user"},` +
		`{"content":[{"id":"c1","input":{"q":"w"},"name":"tool_search","type":"tool_use"},` +
		`{"id":"c2","input":{},"name":"read_file","type":"tool_use"}],"role":"assistant"},` +
		`{"content":[{"content":[{"tool_name":"get_weather","type":"tool_reference"},` +
		`{"tool_name":"lookup_stock","type":"tool_reference"}],"tool_use_id":"c1","type":"tool_result"},` +
		`{"content":"file","tool_use_id":"c2","type":"tool_result"},` +
		`{"text":"Loaded two.","type":"text"}],"role":"user"},` +
		`{"content":"ok","role":"assistant"}]`
	if got != want {
		t.Errorf("FromItemsLoading:\n got  %s\n want %s", got, want)
	}
}

// TestFromItemsLoading_Unchanged_0040: with no load, FromItemsLoading is
// FromItems byte for byte, and FromItems drops every ToolLoadItem.
func TestFromItemsLoading_Unchanged_0040(t *testing.T) {
	plain := []llmprovider.Item{
		llmprovider.MessageItem{Text: "hi"},
		llmprovider.FunctionCallItem{CallID: "c1", Name: "f", Arguments: `{}`},
		llmprovider.FunctionCallOutputItem{CallID: "c1", Output: "x", IsError: true},
	}
	if a, b := marshal(t, FromItemsLoading(plain)), marshal(t, FromItems(plain)); a != b {
		t.Errorf("with no load:\n FromItemsLoading %s\n FromItems        %s", a, b)
	}
	withLoad := append(append([]llmprovider.Item{}, plain...), llmprovider.ToolLoadItem{CallID: "c1", Name: "g"})
	if a, b := marshal(t, FromItems(withLoad)), marshal(t, FromItems(plain)); a != b {
		t.Errorf("FromItems sent a load:\n with %s\n without %s", a, b)
	}
}

// TestFromItemsLoading_EmptyOutput_0040: a loading result with no text sends
// no text block.
func TestFromItemsLoading_EmptyOutput_0040(t *testing.T) {
	items := []llmprovider.Item{
		llmprovider.FunctionCallItem{CallID: "c1", Name: "s", Arguments: `{}`},
		llmprovider.FunctionCallOutputItem{CallID: "c1"},
		llmprovider.ToolLoadItem{CallID: "c1", Name: "g"},
	}
	want := `[{"content":[{"id":"c1","input":{},"name":"s","type":"tool_use"}],"role":"assistant"},` +
		`{"content":[{"content":[{"tool_name":"g","type":"tool_reference"}],"tool_use_id":"c1","type":"tool_result"}],"role":"user"}]`
	if got := marshal(t, FromItemsLoading(items)); got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

// TestFromItemsLoading_RemappedID_0040: a call id Anthropic refuses is
// rewritten, and the load follows its result's new id.
func TestFromItemsLoading_RemappedID_0040(t *testing.T) {
	items := []llmprovider.Item{
		llmprovider.FunctionCallItem{CallID: "fn#1", Name: "s", Arguments: `{}`},
		llmprovider.FunctionCallOutputItem{CallID: "fn#1", Output: "x"},
		llmprovider.ToolLoadItem{CallID: "fn#1", Name: "g"},
	}
	msgs := FromItemsLoading(items)
	result := (*msgs[1].Content.blocks)[0]
	if result.Content.blocks == nil || len(*result.Content.blocks) != 1 {
		t.Fatalf("the remapped result lost its load: %s", marshal(t, msgs))
	}
	if *result.ToolUseID == "fn#1" {
		t.Errorf("the call id was not remapped: %s", marshal(t, msgs))
	}
}

// TestAddCache_Deferred_0040: with no system prompt, the breakpoint is on
// the last tool that is not deferred.
func TestAddCache_Deferred_0040(t *testing.T) {
	body := map[string]any{"tools": []map[string]any{
		{"name": "a"}, {"name": "b"}, {"name": "c", "defer_loading": true}, {"name": "d", "defer_loading": true},
	}}
	AddCache(body, llmprovider.CacheShort)
	tools := body["tools"].([]map[string]any)
	for i, tool := range tools {
		_, marked := tool[keyCacheControl]
		if marked != (i == 1) {
			t.Errorf("tool %d (%v): marked %v; want only b marked", i, tool["name"], marked)
		}
	}
	// Every tool deferred is refused before the network (D1), but AddCache
	// still marks none of them.
	body = map[string]any{"tools": []map[string]any{{"name": "c", "defer_loading": true}}}
	AddCache(body, llmprovider.CacheShort)
	if _, marked := body["tools"].([]map[string]any)[0][keyCacheControl]; marked {
		t.Error("a deferred tool was marked")
	}
}
