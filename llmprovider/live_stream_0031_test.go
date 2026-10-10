//go:build live_gateways

package llmprovider_test

import (
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
)

// liveStream streams "Say hi" from id's model with at most 16 output tokens,
// and wants at least one text delta before EventDone (0031-MADR D1). A
// transient failure, a refused model among them, skips (0027-MADR; 0031-PLAN
// D1). Each call is one billed request.
func liveStream(t *testing.T, id llmprovider.ProviderID, model string, effort llmprovider.Effort) {
	t.Helper()
	key := llmprovider.LiveEnvKey(t, llmprovider.ProviderEnvVars()[id])
	p, err := providers.New(id, llmprovider.WithAPIKey(key), llmprovider.WithModel(model), llmprovider.WithMaxTokens(16),
		llmprovider.WithModelProbes(false))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := p.(llmprovider.Streamer); !ok {
		t.Fatalf("%s does not implement Streamer", id)
	}
	req := &llmprovider.Request{Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "Say hi"}}}
	if effort != "" {
		req.Reasoning = &llmprovider.Reasoning{Effort: effort}
	}
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

// TestLive_OpenAIStreams streams from the Responses API with an API key.
func TestLive_OpenAIStreams(t *testing.T) {
	liveStream(t, llmprovider.ProviderOpenAI, "gpt-4.1-mini", "")
}

// TestLive_GrokStreams streams from xAI's Responses API.
func TestLive_GrokStreams(t *testing.T) {
	liveStream(t, llmprovider.ProviderGrok, "grok-4.6", "")
}

// TestLive_OpencodeStreams streams from each gateway's responses route, at
// effort low: at the default effort a 16-token answer can go wholly to
// reasoning (0031-PLAN D2).
func TestLive_OpencodeStreams(t *testing.T) {
	for _, id := range []llmprovider.ProviderID{llmprovider.ProviderOpencodeZen, llmprovider.ProviderOpencodeGo} {
		t.Run(string(id), func(t *testing.T) {
			liveStream(t, id, responsesModel(t, id), llmprovider.EffortLow)
		})
	}
}
