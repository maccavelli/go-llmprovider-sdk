package gemini

import (
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Usage counts as 0015-MADR amendment "what `Usage` counts" defines them:
// totals that hold their parts.

// TestDecodeInteraction_Usage: thoughts are outside total_output_tokens, as
// in generateContent, as TestLive_GeminiInteractionsThoughtTokens measured on
// 2026-10-02.
func TestDecodeInteraction_Usage(t *testing.T) {
	steps := `"id":"v1","status":"completed","steps":[{"type":"model_output","content":[{"type":"text","text":"hi"}]}]`
	resp, err := decodeInteraction(strings.NewReader(`{` + steps + `,"usage":{"total_input_tokens":120,` +
		`"total_cached_tokens":100,"total_output_tokens":10,"total_thought_tokens":30,"total_tokens":160}}`))
	want := llmprovider.Usage{InputTokens: 120, OutputTokens: 40, ReasoningTokens: 30, CachedTokens: 100}
	if err != nil || resp.Usage != want {
		t.Fatalf("Usage = %+v (err %v), want %+v", resp.Usage, err, want)
	}
	resp, err = decodeInteraction(strings.NewReader(`{` + steps + `}`))
	if err != nil || resp.Usage != (llmprovider.Usage{}) {
		t.Errorf("not reported: Usage = %+v (err %v), want zero", resp.Usage, err)
	}
}
