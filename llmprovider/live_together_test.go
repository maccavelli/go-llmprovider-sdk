//go:build live_gateways

package llmprovider

import (
	"errors"
	"os"
	"strings"
	"testing"
)

// togetherLiveKey skips unless LLMPROVIDER_LIVE_TOGETHER=1 and TOGETHER_API_KEY
// are set: every call is billed.
func togetherLiveKey(t *testing.T) string {
	t.Helper()
	if os.Getenv("LLMPROVIDER_LIVE_TOGETHER") != "1" {
		t.Skip("LLMPROVIDER_LIVE_TOGETHER unset: live Together calls are billed")
	}
	return liveEnvKey(t, "TOGETHER_API_KEY")
}

// TestLive_TogetherWire confirms MADR 0017 D1's wire shapes against the
// service: text and a forced tool, the thinking path's reasoning {"enabled":
// true} on a toggleable model, and reasoning_effort on gpt-oss.
func TestLive_TogetherWire(t *testing.T) {
	key := togetherLiveKey(t)
	for _, tc := range []struct {
		name, model, effort string
		thinking, tool      bool
	}{
		{name: "text", model: "openai/gpt-oss-120b"},
		{name: "forced tool", model: "openai/gpt-oss-120b", tool: true},
		{name: "thinking toggle", model: "deepseek-ai/DeepSeek-V4.1-Flash", thinking: true},
		{name: "thinking with effort", model: "openai/gpt-oss-120b", effort: "high", thinking: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := liveCtx(t)
			defer cancel()
			p, err := NewTogether(key, tc.model, WithMaxTokens(512), WithReasoningEffort(tc.effort))
			if err != nil {
				t.Fatal(err)
			}
			var out string
			switch {
			case tc.tool:
				out, err = p.GenerateWithTool(ctx, "What is the weather in Paris? Use the tool.", weatherTool)
			case tc.thinking:
				out, err = p.GenerateThinking(ctx, "Reply with only the word ALPHA")
			default:
				out, err = p.Generate(ctx, "Reply with only the word ALPHA")
			}
			if errors.Is(err, ErrRateLimited) {
				t.Skipf("rate limited: %v", err)
			}
			if err != nil {
				t.Fatalf("%s: %v", tc.model, err)
			}
			if tc.tool && !strings.Contains(out, "Paris") {
				t.Errorf("tool arguments %q do not name Paris", out)
			}
			if !tc.tool && !strings.Contains(strings.ToUpper(out), "ALPHA") {
				t.Errorf("reply %q does not contain ALPHA", out)
			}
		})
	}
}

// TestLive_TogetherListing confirms the bare-array listing: chat models come
// back, curated, and no generation is sent.
func TestLive_TogetherListing(t *testing.T) {
	key := togetherLiveKey(t)
	ctx, cancel := liveCtx(t)
	defer cancel()
	usable, err := fetchTogetherUsable(ctx, key, ApplyOptions(nil))
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(usable) == 0 {
		t.Fatal("no chat models listed")
	}
	t.Logf("%d chat models; first %v", len(usable), usable[:min(len(usable), 5)])
}
