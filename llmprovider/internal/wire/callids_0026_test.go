package wire

import (
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestRemapCallIDs_Edges (0026-MADR F8): an empty id, a result with no call
// before it, a remapped id that collides with a real one, a collision at the
// length limit, and a request that needs no change.
func TestRemapCallIDs_Edges(t *testing.T) {
	at64 := strings.Repeat("y", 64)
	got := RemapCallIDs([]llmprovider.Item{
		llmprovider.FunctionCallItem{CallID: "a_0", Name: "a"},
		llmprovider.FunctionCallItem{CallID: "a#0", Name: "a"},
		llmprovider.FunctionCallItem{CallID: "", Name: "b"},
		llmprovider.FunctionCallOutputItem{CallID: "orphan#1"},
		llmprovider.FunctionCallItem{CallID: at64, Name: "c"},
		llmprovider.FunctionCallItem{CallID: at64, Name: "c"},
		llmprovider.FunctionCallOutputItem{CallID: at64},
	}, AnthropicCallIDs)
	ids := make([]string, len(got))
	for i, item := range got {
		switch v := item.(type) {
		case llmprovider.FunctionCallItem:
			ids[i] = v.CallID
		case llmprovider.FunctionCallOutputItem:
			ids[i] = v.CallID
		}
	}
	want := []string{"a_0", "a_0_2", "call", "orphan_1", at64, at64[:62] + "_2", at64[:62] + "_2"}
	for i := range want {
		if ids[i] != want[i] {
			t.Errorf("item %d: id %q; want %q", i, ids[i], want[i])
		}
	}

	clean := []llmprovider.Item{
		llmprovider.FunctionCallItem{CallID: "toolu_1", Name: "a"},
		llmprovider.FunctionCallOutputItem{CallID: "toolu_1"},
	}
	if got := RemapCallIDs(clean, AnthropicCallIDs); &got[0] != &clean[0] {
		t.Error("a request with valid, unique ids was copied; want it returned as it is")
	}
}
