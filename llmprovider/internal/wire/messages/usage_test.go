package messages

import (
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Usage counts as 0015-MADR amendment "what `Usage` counts" defines them:
// totals that hold their parts.

// TestDecode_Usage: Anthropic counts cache reads and writes outside
// input_tokens, so InputTokens adds them; it reports no reasoning count.
func TestDecode_Usage(t *testing.T) {
	body := `{"content":[{"type":"text","text":"hi"}],"usage":{"input_tokens":15,"cache_creation_input_tokens":5,` +
		`"cache_read_input_tokens":100,"output_tokens":40}}`
	resp, err := Decode(strings.NewReader(body))
	want := llmprovider.Usage{InputTokens: 120, OutputTokens: 40, CachedTokens: 100}
	if err != nil || resp.Usage != want {
		t.Fatalf("Usage = %+v (err %v), want %+v", resp.Usage, err, want)
	}
	resp, err = Decode(strings.NewReader(`{"content":[{"type":"text","text":"hi"}]}`))
	if err != nil || resp.Usage != (llmprovider.Usage{}) {
		t.Errorf("not reported: Usage = %+v (err %v), want zero", resp.Usage, err)
	}
}
