package chatcompletions

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestBody_Strict (0032-MADR D2, Q2): function.strict is sent only when the
// gateway sends it (Opts.SendStrict) and the tool sets it.
func TestBody_Strict(t *testing.T) {
	no := false
	for name, tc := range map[string]struct {
		strict *bool
		send   bool
		want   string
	}{
		"set, sent":     {&no, true, `"strict":false`},
		"set, not sent": {&no, false, ""},
		"nil, sent":     {nil, true, ""},
	} {
		body := Body("m", 16, nil, Opts{Tools: []llmprovider.Tool{{Name: "t", Strict: tc.strict}}, SendStrict: tc.send})
		raw, err := json.Marshal(body["tools"])
		if err != nil {
			t.Fatal(err)
		}
		got := string(raw)
		if tc.want == "" && strings.Contains(got, `"strict"`) || tc.want != "" && !strings.Contains(got, tc.want) {
			t.Errorf("%s: tools %s; want %q", name, got, tc.want)
		}
	}
}
