package opencode

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestStrict_0032 (0032-MADR D2): a tool's Strict is sent on the responses
// and chat_completions routes, and not on messages or google; a nil Strict
// sends no strict key anywhere.
func TestStrict_0032(t *testing.T) {
	no := false
	replies := map[Route]struct{ reply, want string }{
		RouteResponses:       {`{"output":[{"type":"message","content":[{"type":"output_text","text":"ok"}]}]}`, `"strict":false`},
		RouteChatCompletions: {`{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"ok"}}]}`, `"strict":false`},
		RouteMessages:        {`{"stop_reason":"end_turn","content":[{"type":"text","text":"ok"}]}`, ""},
		RouteGoogle:          {`{"candidates":[{"finishReason":"STOP","content":{"parts":[{"text":"ok"}]}}]}`, ""},
	}
	for route, r := range replies {
		for _, strict := range []*bool{nil, &no} {
			want := r.want
			if strict == nil {
				want = ""
			}
			var body map[string]any
			srv := captureServer(t, &body, r.reply)
			p := build(t, llmprovider.ProviderOpencodeGo, apiKey(srv.URL, "some-model", WithRoute(route))...)
			req := text("hi")
			req.Tools = []llmprovider.Tool{{Name: "record_city", Strict: strict}}
			if _, err := p.Generate(context.Background(), req); err != nil {
				t.Fatal(err)
			}
			raw, err := json.Marshal(body["tools"])
			if err != nil {
				t.Fatal(err)
			}
			tools := string(raw)
			if want == "" && strings.Contains(tools, `"strict"`) || want != "" && !strings.Contains(tools, want) {
				t.Errorf("%s, strict %v: sent tools %s; want %q", route, strict != nil, tools, want)
			}
		}
	}
}
