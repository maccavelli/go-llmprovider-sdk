package wire

import (
	"encoding/json"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestToolArguments: an object is passed through, empty arguments are an
// empty object, and anything else is kept under "arguments" rather than
// dropped. Each is compared as the JSON it encodes to (0021-MADR W1).
func TestToolArguments(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{`{"city":"Paris"}`, `{"city":"Paris"}`},
		{"", `{}`},
		{"  ", `{}`},
		{`["not","an","object"]`, `{"arguments":"[\"not\",\"an\",\"object\"]"}`},
		{`null`, `{"arguments":"null"}`},
		{`{"subject":"fix: tru`, `{"arguments":"{\"subject\":\"fix: tru"}`},
	} {
		got, err := json.Marshal(ToolArguments(tc.in))
		if err != nil || string(got) != tc.want {
			t.Errorf("ToolArguments(%q) = %s, %v; want %s", tc.in, got, err, tc.want)
		}
	}
}

// TestToolArguments_PassesValidJSONThrough (0021-MADR W1): a call's arguments
// are replayed byte for byte, compacted: a large integer keeps its digits and
// the keys their order.
func TestToolArguments_PassesValidJSONThrough(t *testing.T) {
	got, err := json.Marshal(ToolArguments(`{"z": 1, "id": 12345678901234567890}`))
	if want := `{"z":1,"id":12345678901234567890}`; err != nil || string(got) != want {
		t.Errorf("ToolArguments = %s, %v; want %s", got, err, want)
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
