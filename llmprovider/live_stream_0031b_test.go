//go:build live_gateways

package llmprovider_test

import (
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/gemini"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/opencode"
)

// liveStreamWith streams "Say hi" from id's model, and wants at least one text
// delta before EventDone (0031-MADR E1). A transient failure, a refused or
// unfunded model among them, skips (0027-MADR). Each call is one billed
// request.
func liveStreamWith(t *testing.T, id llmprovider.ProviderID, model string, reasoning *llmprovider.Reasoning, maxTokens int,
	opts ...llmprovider.Option) {
	t.Helper()
	liveStreamKey(t, id, llmprovider.ProviderEnvVars()[id], model, reasoning, maxTokens, opts...)
}

// liveStreamKey is liveStreamWith with the key read from keyVar.
func liveStreamKey(t *testing.T, id llmprovider.ProviderID, keyVar, model string, reasoning *llmprovider.Reasoning, maxTokens int,
	opts ...llmprovider.Option) {
	t.Helper()
	key := llmprovider.LiveEnvKey(t, keyVar)
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

// TestLive_ClaudeStreams streams from the Messages API, on Haiku 4.5 until
// 0034-MADR D5.
func TestLive_ClaudeStreams(t *testing.T) {
	liveStreamWith(t, llmprovider.ProviderClaude, "claude-haiku-5-5", nil, 16)
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

// chatLiveTokens is the output cap of the Chat Completions live streams: the
// gateways' default models reason first, and at 16 tokens some answered with
// reasoning alone (0031-PLAN-native-streaming-on-the-other-wires, Phase 1).
const chatLiveTokens = 512

// TestLive_HuggingFaceStreams streams from the Hugging Face router.
func TestLive_HuggingFaceStreams(t *testing.T) {
	liveStreamWith(t, llmprovider.ProviderHuggingFace, catalog.Static(llmprovider.ProviderHuggingFace)[0], nil, chatLiveTokens)
}

// TestLive_KiloStreams streams from the Kilo gateway.
func TestLive_KiloStreams(t *testing.T) {
	liveStreamWith(t, llmprovider.ProviderKilo, catalog.Static(llmprovider.ProviderKilo)[0], nil, chatLiveTokens)
}

// TestLive_TogetherStreams streams from Together AI.
func TestLive_TogetherStreams(t *testing.T) {
	liveStreamWith(t, llmprovider.ProviderTogether, catalog.Static(llmprovider.ProviderTogether)[0], nil, chatLiveTokens)
}

// TestLive_OpencodeChatStreams streams from each gateway's chat_completions
// route.
func TestLive_OpencodeChatStreams(t *testing.T) {
	for _, id := range []llmprovider.ProviderID{llmprovider.ProviderOpencodeZen, llmprovider.ProviderOpencodeGo} {
		t.Run(string(id), func(t *testing.T) {
			liveStreamWith(t, id, "glm-5.3-flash", nil, chatLiveTokens,
				opencode.WithRoute(opencode.RouteChatCompletions), llmprovider.WithoutModelMetadata())
		})
	}
}

// TestLive_OllamaStreams streams from a local Ollama server, when one
// answers; Ollama takes no key, and its requests are not billed.
func TestLive_OllamaStreams(t *testing.T) {
	model := ollamaModel()
	if model == "" {
		t.Skip("no local Ollama server answered")
	}
	p, err := providers.New(llmprovider.ProviderOllama, llmprovider.WithModel(model), llmprovider.WithMaxTokens(16),
		llmprovider.WithModelProbes(false))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	deltas, done := 0, false
	for ev, err := range llmprovider.Stream(ctx, p, &llmprovider.Request{Input: []llmprovider.Item{
		llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "Say hi"}}}) {
		if err != nil {
			t.Fatalf("ollama %s: after %d deltas: %v", model, deltas, err)
		}
		switch ev.Type {
		case llmprovider.EventTextDelta:
			deltas++
		case llmprovider.EventDone:
			done = true
		}
	}
	if deltas == 0 || !done {
		t.Fatalf("ollama %s: %d text deltas, done %v; want at least one delta, then EventDone", model, deltas, done)
	}
	t.Logf("ollama %s: %d text deltas, then EventDone", model, deltas)
}

// TestLive_OpencodeGoogleStreams streams generateContent through the google
// route's request: from Google's own endpoint, whose evidence stands for the
// route (0031-PLAN-native-streaming-on-the-other-wires D1), and from Zen's
// route. Effort low and a 1,024-token cap: at 16 tokens gemini-3.7-flash's
// answer went wholly to thinking (D2).
func TestLive_OpencodeGoogleStreams(t *testing.T) {
	low := &llmprovider.Reasoning{Effort: llmprovider.EffortLow}
	t.Run("google-endpoint", func(t *testing.T) {
		liveStreamKey(t, llmprovider.ProviderOpencodeZen, "GEMINI_API_KEY", catalog.Static(llmprovider.ProviderGemini)[0], low, 1024,
			opencode.WithRoute(opencode.RouteGoogle), llmprovider.WithBaseURL("https://generativelanguage.googleapis.com/v1beta"),
			llmprovider.WithoutModelMetadata())
	})
	t.Run("opencode-zen", func(t *testing.T) {
		liveStreamWith(t, llmprovider.ProviderOpencodeZen, "gemini-3.5-flash-lite", low, 1024,
			opencode.WithRoute(opencode.RouteGoogle), llmprovider.WithoutModelMetadata())
	})
}

// TestLive_GeminiStreams streams from the Interactions API, stored, so that
// it also shows the streamed Response carries the interaction's id, which
// continuation needs (0031-MADR E3). Effort low and a 1,024-token cap: at 16
// tokens gemini-3.7-flash's answer went wholly to thinking (0031-PLAN-native-
// streaming-on-the-other-wires D2). One billed request.
func TestLive_GeminiStreams(t *testing.T) {
	key := llmprovider.LiveEnvKey(t, llmprovider.ProviderEnvVars()[llmprovider.ProviderGemini])
	model := catalog.Static(llmprovider.ProviderGemini)[0]
	p, err := providers.New(llmprovider.ProviderGemini, llmprovider.WithAPIKey(key), llmprovider.WithModel(model),
		llmprovider.WithMaxTokens(1024), llmprovider.WithModelProbes(false), gemini.WithStore(true))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := llmprovider.LiveCtx(t)
	defer cancel()
	deltas := 0
	var done *llmprovider.Response
	for ev, err := range llmprovider.Stream(ctx, p, &llmprovider.Request{Reasoning: &llmprovider.Reasoning{Effort: llmprovider.EffortLow},
		Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "Say hi"}}}) {
		if err != nil {
			llmprovider.SkipIfTransient(t, err)
			t.Fatalf("gemini %s: after %d deltas: %v", model, deltas, err)
		}
		switch ev.Type {
		case llmprovider.EventTextDelta:
			deltas++
		case llmprovider.EventDone:
			done = ev.Response
		}
	}
	if deltas == 0 || done == nil || done.ID == "" {
		t.Fatalf("gemini %s: %d text deltas, Response %+v; want a delta, then EventDone with the interaction's id", model, deltas, done)
	}
	t.Logf("gemini %s: %d text deltas, then EventDone with an interaction id", model, deltas)
}
