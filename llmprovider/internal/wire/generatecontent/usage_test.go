package generatecontent

import (
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Usage counts as 0015-MADR amendment "what `Usage` counts" defines them:
// totals that hold their parts.

// TestDecode_Usage: generateContent counts thoughts outside
// candidatesTokenCount, so OutputTokens adds them.
func TestDecode_Usage(t *testing.T) {
	parts := `"candidates":[{"content":{"role":"model","parts":[{"text":"hi"}]}}]`
	resp, err := Decode(strings.NewReader(`{` + parts + `,"usageMetadata":{"promptTokenCount":120,` +
		`"cachedContentTokenCount":100,"candidatesTokenCount":10,"thoughtsTokenCount":30,"totalTokenCount":160}}`))
	want := llmprovider.Usage{InputTokens: 120, OutputTokens: 40, ReasoningTokens: 30, CachedTokens: 100}
	if err != nil || resp.Usage != want {
		t.Fatalf("Usage = %+v (err %v), want %+v", resp.Usage, err, want)
	}
	resp, err = Decode(strings.NewReader(`{` + parts + `}`))
	if err != nil || resp.Usage != (llmprovider.Usage{}) {
		t.Errorf("not reported: Usage = %+v (err %v), want zero", resp.Usage, err)
	}
}
