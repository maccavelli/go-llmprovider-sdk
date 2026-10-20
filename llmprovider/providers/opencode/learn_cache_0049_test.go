package opencode

import (
	"context"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// TestCache_KeepsLearnedRefusalsApart_0049 (0049-MADR D3): on Zen's messages
// route, a refusal learned by a provider built WithCache(&a) is known to
// another provider of a, and not to one of b, which forces first and learns
// it again.
func TestCache_KeepsLearnedRefusalsApart_0049(t *testing.T) {
	srv, seen := refusingServer(t, callReply0034, refusal0034, false)
	var a, b catalog.Cache
	const model = "claude-future-zen-0049"
	for i, step := range []struct {
		cache *catalog.Cache
		want  string
	}{
		{&a, "tool,none"},
		{&a, "tool,none,none"},
		{&b, "tool,none,none,tool,none"},
	} {
		p := build(t, llmprovider.ProviderOpencodeZen, apiKey(srv.URL, model, catalog.WithCache(step.cache))...)
		if _, err := llmprovider.GenerateToolCall(context.Background(), p, weatherRequest0034()); err != nil {
			t.Fatalf("step %d: GenerateToolCall: %v", i, err)
		}
		if got := seen(); got != step.want {
			t.Fatalf("step %d: tool_choice %s; want %s", i, got, step.want)
		}
	}
}
