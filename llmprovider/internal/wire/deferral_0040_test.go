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
