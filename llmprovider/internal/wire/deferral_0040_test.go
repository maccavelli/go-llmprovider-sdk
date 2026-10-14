package wire

import (
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0040-MADR D2 and A5: defer_loading on the native Messages form only, and a
// load that keeps its result's rewritten call id.

// TestMessagesToolsNative_0040: defer_loading on each deferred tool, and on
// no other; MessagesTools sends it on none.
func TestMessagesToolsNative_0040(t *testing.T) {
	tools := []llmprovider.Tool{{Name: "a"}, {Name: "s", Search: true}, {Name: "d", Deferred: true}}
	native := MessagesToolsNative(tools)
	for i, tool := range native {
		_, has := tool["defer_loading"]
		if has != (i == 2) {
			t.Errorf("native tool %d (%v): defer_loading present %v; want only on d", i, tool[KeyName], has)
		}
	}
	if native[2]["defer_loading"] != true {
		t.Errorf("defer_loading = %v, want true", native[2]["defer_loading"])
	}
	for i, tool := range MessagesTools(tools) {
		if _, has := tool["defer_loading"]; has {
			t.Errorf("MessagesTools tool %d sent defer_loading", i)
		}
	}
}

// TestAddResponsesToolsNative_0040: the Search tool is a client tool_search,
// with its description and schema and no strict or name; every other tool is
// the function AddResponsesTools sends.
func TestAddResponsesToolsNative_0040(t *testing.T) {
	strict := true
	schema := map[string]any{"type": "object", "properties": map[string]any{"query": map[string]any{"type": "string"}}}
	tools := []llmprovider.Tool{{Name: "read_file", Strict: &strict},
		{Name: "tool_search", Description: "Finds tools.", Schema: schema, Search: true, Strict: &strict}}
	native, plain := map[string]any{}, map[string]any{}
	AddResponsesToolsNative(native, tools, "")
	AddResponsesTools(plain, tools, "")
	got := native["tools"].([]map[string]any)
	want := plain["tools"].([]map[string]any)
	if len(got) != 2 || got[0][KeyName] != want[0][KeyName] || got[0]["strict"] != true {
		t.Fatalf("native tools %v; want read_file as AddResponsesTools sends it first", got)
	}
	search := got[1]
	if search[KeyType] != "tool_search" || search["execution"] != "client" || search[keyDescription] != "Finds tools." {
		t.Errorf("search entry %v; want a client tool_search with its description", search)
	}
	if _, ok := search[KeyName]; ok {
		t.Errorf("search entry sent a name: %v", search)
	}
	if _, ok := search["strict"]; ok {
		t.Errorf("search entry sent strict: %v", search)
	}
	if params, ok := search[keyParameters].(map[string]any); !ok || params["type"] != "object" {
		t.Errorf("search parameters %v; want the tool's schema", search[keyParameters])
	}
	// A forced choice of another tool is sent as AddResponsesTools sends it.
	native = map[string]any{}
	AddResponsesToolsNative(native, tools, llmprovider.ForceTool("read_file"))
	if choice, _ := native[keyToolChoice].(map[string]any); choice[KeyName] != "read_file" {
		t.Errorf("tool_choice %v; want read_file forced", native[keyToolChoice])
	}
}

// TestRemapCallIDs_ToolLoad_0040: a ToolLoadItem takes the id its call's
// result is sent with.
func TestRemapCallIDs_ToolLoad_0040(t *testing.T) {
	items := []llmprovider.Item{
		llmprovider.FunctionCallItem{CallID: "fn#1", Name: "s", Arguments: "{}"},
		llmprovider.FunctionCallOutputItem{CallID: "fn#1", Output: "x"},
		llmprovider.ToolLoadItem{CallID: "fn#1", Name: "g"},
	}
	out := RemapCallIDs(items, AnthropicCallIDs)
	result := out[1].(llmprovider.FunctionCallOutputItem)
	load := out[2].(llmprovider.ToolLoadItem)
	if result.CallID == "fn#1" {
		t.Fatalf("the result's id was not remapped: %+v", out)
	}
	if load.CallID != result.CallID {
		t.Errorf("the load's id %q differs from its result's %q", load.CallID, result.CallID)
	}

	// A repeated id: the second call is made unique, and a load after the
	// second result must take that id, not the first call's.
	repeated := []llmprovider.Item{
		llmprovider.FunctionCallItem{CallID: "fn#1", Name: "s", Arguments: "{}"},
		llmprovider.FunctionCallOutputItem{CallID: "fn#1", Output: "x"},
		llmprovider.FunctionCallItem{CallID: "fn#1", Name: "s", Arguments: "{}"},
		llmprovider.FunctionCallOutputItem{CallID: "fn#1", Output: "y"},
		llmprovider.ToolLoadItem{CallID: "fn#1", Name: "g"},
	}
	out = RemapCallIDs(repeated, AnthropicCallIDs)
	first := out[1].(llmprovider.FunctionCallOutputItem).CallID
	second := out[3].(llmprovider.FunctionCallOutputItem).CallID
	load = out[4].(llmprovider.ToolLoadItem)
	if first == second {
		t.Fatalf("the repeated id was not made unique: %+v", out)
	}
	if load.CallID != second {
		t.Errorf("the load's id %q, want the second result's %q (the first's is %q)", load.CallID, second, first)
	}
}
