package wire

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestResponsesTools_Strict (0032-MADR D1, D2): a nil Strict sends no strict
// key; a set one sends its value.
func TestResponsesTools_Strict(t *testing.T) {
	no, yes := false, true
	for name, tc := range map[string]struct {
		strict *bool
		want   string
	}{"nil": {nil, ""}, "false": {&no, `"strict":false`}, "true": {&yes, `"strict":true`}} {
		raw, err := json.Marshal(ResponsesTools([]llmprovider.Tool{{Name: "t", Strict: tc.strict}}))
		if err != nil {
			t.Fatal(err)
		}
		got := string(raw)
		if tc.want == "" && strings.Contains(got, `"strict"`) || tc.want != "" && !strings.Contains(got, tc.want) {
			t.Errorf("%s: tools %s; want %q", name, got, tc.want)
		}
	}
}
