package wire

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestMessagesTools_Strict_0034 (0034-MADR D4, amending 0032-MADR D2): a nil
// Strict sends no strict key; a set one sends its value, at the tool's top
// level.
func TestMessagesTools_Strict_0034(t *testing.T) {
	no, yes := false, true
	for name, tc := range map[string]struct {
		strict *bool
		want   string
	}{"nil": {nil, ""}, "false": {&no, `"strict":false`}, "true": {&yes, `"strict":true`}} {
		raw, err := json.Marshal(MessagesTools([]llmprovider.Tool{{Name: "t", Strict: tc.strict}}))
		if err != nil {
			t.Fatal(err)
		}
		got := string(raw)
		if tc.want == "" && strings.Contains(got, `"strict"`) || tc.want != "" && !strings.Contains(got, tc.want) {
			t.Errorf("%s: tools %s; want %q", name, got, tc.want)
		}
	}
}
