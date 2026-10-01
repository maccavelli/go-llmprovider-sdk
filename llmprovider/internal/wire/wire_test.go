package wire

import (
	"fmt"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestToolArguments: an object decodes, empty arguments are an empty object,
// and anything else is kept under "arguments" rather than dropped.
func TestToolArguments(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{`{"city":"Paris"}`, "map[city:Paris]"},
		{"", "map[]"},
		{"  ", "map[]"},
		{`["not","an","object"]`, `map[arguments:["not","an","object"]]`},
		{`null`, "map[arguments:null]"},
		{`{"subject":"fix: tru`, `map[arguments:{"subject":"fix: tru]`},
	} {
		if got := fmt.Sprint(ToolArguments(tc.in)); got != tc.want {
			t.Errorf("ToolArguments(%q) = %s, want %s", tc.in, got, tc.want)
		}
	}
}

// TestSystemPrompt: the system items' text, in order, joined by a blank line;
// other items and empty system text are left out.
func TestSystemPrompt(t *testing.T) {
	items := []llmprovider.Item{
		llmprovider.MessageItem{Role: RoleSystem, Text: "Be brief."},
		llmprovider.MessageItem{Role: RoleUser, Text: "hi"},
		llmprovider.MessageItem{Role: RoleSystem, Text: ""},
		llmprovider.FunctionCallItem{CallID: "c", Name: "n"},
		llmprovider.MessageItem{Role: RoleSystem, Text: "Answer in French."},
	}
	if got := SystemPrompt(items); got != "Be brief.\n\nAnswer in French." {
		t.Errorf("SystemPrompt = %q", got)
	}
	if got := SystemPrompt([]llmprovider.Item{llmprovider.MessageItem{Role: RoleUser, Text: "hi"}}); got != "" {
		t.Errorf("no system items: SystemPrompt = %q, want empty", got)
	}
}
