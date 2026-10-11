package claude

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestStrict_0032 (0032-MADR D2, amended by 0034-MADR D4): a tool's Strict,
// when set, is sent as its strict, and a nil Strict sends no strict key.
func TestStrict_0032(t *testing.T) {
	no := false
	for name, tc := range map[string]struct {
		strict *bool
		want   string
	}{"nil": {nil, ""}, "false": {&no, `"strict":false`}} {
		srv, got := sseServer(t, `{"model":"m","stop_reason":"end_turn","content":[{"type":"text","text":"ok"}]}`)
		p := build(t, apiKey(srv.URL, "claude-haiku-4-5")...)
		req := text("hi")
		req.Tools = []llmprovider.Tool{{Name: "record_city", Strict: tc.strict}}
		if _, err := p.Generate(context.Background(), req); err != nil {
			t.Fatal(err)
		}
		got.mu.Lock()
		raw, err := json.Marshal(got.body["tools"])
		got.mu.Unlock()
		if err != nil {
			t.Fatal(err)
		}
		tools := string(raw)
		if tc.want == "" && strings.Contains(tools, `"strict"`) || tc.want != "" && !strings.Contains(tools, tc.want) {
			t.Errorf("%s: sent tools %s; want %q", name, tools, tc.want)
		}
	}
}
