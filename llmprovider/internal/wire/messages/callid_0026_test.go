package messages

import (
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// anthropicToolID is Anthropic's documented rule for a tool_use id.
var anthropicToolID = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// TestFromItems_CallIDsFitAnthropicRule (0026-MADR F8): a Messages request
// built from another wire's items, such as Gemini's "name#index" ids, has
// tool_use ids Anthropic accepts, distinct across turns, and each
// tool_result pairs with the call before it. The caller's items are not
// changed.
func TestFromItems_CallIDsFitAnthropicRule(t *testing.T) {
	long := strings.Repeat("x", 100)
	items := []llmprovider.Item{
		llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "weather in two cities"},
		llmprovider.FunctionCallItem{CallID: "get_weather#0", Name: "get_weather", Arguments: `{"city":"Oslo"}`},
		llmprovider.FunctionCallItem{CallID: "get_weather#1", Name: "get_weather", Arguments: `{"city":"Rome"}`},
		llmprovider.FunctionCallOutputItem{CallID: "get_weather#0", Output: "oslo"},
		llmprovider.FunctionCallOutputItem{CallID: "get_weather#1", Output: "rome"},
		llmprovider.MessageItem{Role: llmprovider.RoleAssistant, Text: "done"},
		llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "and again"},
		llmprovider.FunctionCallItem{CallID: "get_weather#0", Name: "get_weather", Arguments: `{"city":"Lima"}`},
		llmprovider.FunctionCallOutputItem{CallID: "get_weather#0", Output: "lima"},
		llmprovider.FunctionCallItem{CallID: long, Name: "f", Arguments: `{}`},
		llmprovider.FunctionCallOutputItem{CallID: long, Output: "long"},
		llmprovider.FunctionCallItem{CallID: "toolu_ok", Name: "f", Arguments: `{}`},
		llmprovider.FunctionCallOutputItem{CallID: "toolu_ok", Output: "ok"},
	}
	before := slices.Clone(items)

	var ids []string
	pairs := map[string]string{} // tool_result id → its output
	for _, m := range FromItems(items) {
		list := m.Content.blocks // typed blocks (0028-PLAN D14, Phase 9b; D16)
		if list == nil {
			continue
		}
		for _, b := range *list {
			switch b.Type {
			case "tool_use":
				ids = append(ids, deref(b.ID))
			case "tool_result":
				pairs[deref(b.ToolUseID)] = deref(b.Content)
			}
		}
	}
	for _, id := range ids {
		if !anthropicToolID.MatchString(id) || len(id) > 64 {
			t.Errorf("tool_use id %q breaks Anthropic's rule", id)
		}
	}
	if len(ids) != 5 || len(slices.Compact(slices.Sorted(slices.Values(ids)))) != 5 {
		t.Fatalf("tool_use ids %q; want 5 distinct", ids)
	}
	for i, want := range []string{"oslo", "rome", "lima", "long", "ok"} {
		if got := pairs[ids[i]]; got != want {
			t.Errorf("call %d (%q) pairs with result %q; want %q", i, ids[i], got, want)
		}
	}
	if ids[4] != "toolu_ok" {
		t.Errorf("a valid, unique id became %q; want it kept", ids[4])
	}
	if !reflect.DeepEqual(items, before) {
		t.Error("FromItems changed the caller's items")
	}
}
