//go:build live_gateways

package llmprovider_test

import (
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/opencode"
)

// liveStreamWith streams "Say hi" from id's model, and wants at least one text
// delta before EventDone (0031-MADR E1). A transient failure, a refused or
// unfunded model among them, skips (0027-MADR). Each call is one billed
// request.
func liveStreamWith(t *testing.T, id llmprovider.ProviderID, model string, reasoning *llmprovider.Reasoning, maxTokens int,
	opts ...llmprovider.Option) {
	t.Helper()
	key := llmprovider.LiveEnvKey(t, llmprovider.ProviderEnvVars()[id])
	p, err := providers.New(id, append([]llmprovider.Option{llmprovider.WithAPIKey(key), llmprovider.WithModel(model),
		llmprovider.WithMaxTokens(maxTokens), llmprovider.WithModelProbes(false)}, opts...)...)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := p.(llmprovider.Streamer); !ok {
		t.Fatalf("%s does not implement Streamer", id)
	}
	req := &llmprovider.Request{Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "Say hi"}},
		Reasoning: reasoning}
	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	deltas, done := 0, false
	for ev, err := range llmprovider.Stream(ctx, p, req) {
		if err != nil {
			llmprovider.SkipIfTransient(t, err)
			t.Fatalf("%s %s: after %d deltas: %v", id, model, deltas, err)
		}
		switch ev.Type {
		case llmprovider.EventTextDelta:
			if done {
				t.Fatalf("%s %s: a delta after EventDone", id, model)
			}
			deltas++
		case llmprovider.EventDone:
			done = true
		}
	}
	if deltas == 0 || !done {
		t.Fatalf("%s %s: %d text deltas, done %v; want at least one delta, then EventDone", id, model, deltas, done)
	}
	t.Logf("%s %s: %d text deltas, then EventDone", id, model, deltas)
}

// TestLive_ClaudeStreams streams from the Messages API.
func TestLive_ClaudeStreams(t *testing.T) {
	liveStreamWith(t, llmprovider.ProviderClaude, "claude-haiku-4-5", nil, 16)
}

// TestLive_OpencodeMessagesStreams streams from each gateway's messages
// route. qwen3.8-flash thinks first, so it is given a 1,024-token budget: at
// 16 tokens its answer went wholly to thinking (0031-PLAN-native-streaming-
// on-the-other-wires, Phase 1).
func TestLive_OpencodeMessagesStreams(t *testing.T) {
	for _, id := range []llmprovider.ProviderID{llmprovider.ProviderOpencodeZen, llmprovider.ProviderOpencodeGo} {
		t.Run(string(id), func(t *testing.T) {
			liveStreamWith(t, id, "qwen3.8-flash", &llmprovider.Reasoning{Budget: 1024}, 1040,
				opencode.WithRoute(opencode.RouteMessages), llmprovider.WithoutModelMetadata())
		})
	}
}
