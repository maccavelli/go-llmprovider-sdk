//go:build live_gateways

package llmprovider_test

import (
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestLive_GeminiFlashLiteEffortThinks (0026-MADR F38, live first): does
// gemini-2.5-flash-lite think at each effort, as the provider sends it? Low
// and medium send no thinking setting (interactions.go thinkingLevel), and
// Google documents the model's default as thinking off. It records the
// reasoning tokens and thought summary of each effort. It REQUIRES
// GEMINI_API_KEY.
func TestLive_GeminiFlashLiteEffortThinks(t *testing.T) {
	key := llmprovider.LiveEnvKey(t, "GEMINI_API_KEY")
	for _, effort := range []llmprovider.Effort{llmprovider.EffortLow, llmprovider.EffortMedium, llmprovider.EffortHigh} {
		t.Run(string(effort), func(t *testing.T) {
			ctx, cancel := llmprovider.LiveCtx(t)
			defer cancel()
			res, err := liveGemini(t, key, "gemini-2.5-flash-lite").Generate(ctx, &llmprovider.Request{
				Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser,
					Text: "A train leaves at 09:47 and the trip takes 2 h 38 min, with a 17 min stop. When does it arrive? Think it through."}},
				Reasoning: &llmprovider.Reasoning{Effort: effort},
			})
			llmprovider.SkipIfTransient(t, err)
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			summary := false
			for _, item := range res.Output {
				if _, ok := item.(llmprovider.ReasoningItem); ok {
					summary = true
				}
			}
			t.Logf("gemini-2.5-flash-lite effort %s: reasoning tokens %d, thought summary %t", effort,
				res.Usage.ReasoningTokens, summary)
		})
	}
}
