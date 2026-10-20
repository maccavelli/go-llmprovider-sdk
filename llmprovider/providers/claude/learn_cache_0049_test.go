package claude

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// TestCache_KeepsLearnedRefusalsApart_0049 (0049-MADR D3): a refusal learned
// by a provider built WithCache(&a) is known to another provider of a, whose
// first forced request is degraded at once, and not to one of b, which forces
// first and learns it again.
func TestCache_KeepsLearnedRefusalsApart_0049(t *testing.T) {
	srv, rs := newRefusingServer(t, callReply0034, false, http.StatusBadRequest, refusal0034)
	var a, b catalog.Cache
	const model = "claude-future-0049"
	for i, step := range []struct {
		cache *catalog.Cache
		want  string
	}{
		{&a, "tool,none"},
		{&a, "tool,none,none"},
		{&b, "tool,none,none,tool,none"},
	} {
		p := build(t, apiKey(srv.URL, model, catalog.WithCache(step.cache))...)
		if call, err := llmprovider.GenerateToolCall(context.Background(), p, weatherRequest()); err != nil || call.Name != "get_weather" {
			t.Fatalf("step %d: GenerateToolCall = %+v, %v; want the call", i, call, err)
		}
		if choices, _ := rs.seen(); strings.Join(choices, ",") != step.want {
			t.Fatalf("step %d: tool_choice %v; want %s", i, choices, step.want)
		}
	}
}
