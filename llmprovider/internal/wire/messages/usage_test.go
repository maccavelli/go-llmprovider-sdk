package messages

import (
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Usage counts as 0015-MADR amendment "what `Usage` counts" defines them:
// totals that hold their parts.

// TestDecode_Usage: Anthropic counts cache reads and writes outside
// input_tokens, so InputTokens adds them, and the writes are CacheWriteTokens
// (0036-MADR D2); it reports no reasoning count.
func TestDecode_Usage(t *testing.T) {
	body := `{"content":[{"type":"text","text":"hi"}],"usage":{"input_tokens":15,"cache_creation_input_tokens":5,` +
		`"cache_read_input_tokens":100,"output_tokens":40}}`
	resp, err := Decode(strings.NewReader(body))
	want := llmprovider.Usage{InputTokens: 120, OutputTokens: 40, CachedTokens: 100, CacheWriteTokens: 5}
	if err != nil || resp.Usage != want {
		t.Fatalf("Usage = %+v (err %v), want %+v", resp.Usage, err, want)
	}
	resp, err = Decode(strings.NewReader(`{"content":[{"type":"text","text":"hi"}]}`))
	if err != nil || resp.Usage != (llmprovider.Usage{}) {
		t.Errorf("not reported: Usage = %+v (err %v), want zero", resp.Usage, err)
	}
}

// The cache-write counts (0036-MADR D1, D2): cache_creation_input_tokens is
// CacheWriteTokens, and cache_creation's 1-hour count CacheWrite1hTokens.
const cacheWriteUsage = `{"input_tokens":15,"cache_read_input_tokens":100,"cache_creation_input_tokens":5,` +
	`"cache_creation":{"ephemeral_5m_input_tokens":3,"ephemeral_1h_input_tokens":2},"output_tokens":40}`

var cacheWriteWant = llmprovider.Usage{InputTokens: 120, OutputTokens: 40, CachedTokens: 100, CacheWriteTokens: 5, CacheWrite1hTokens: 2}

func TestDecode_CacheWrites(t *testing.T) {
	for _, c := range []struct {
		name, usage string
		want        llmprovider.Usage
	}{
		{"with cache_creation", cacheWriteUsage, cacheWriteWant},
		{"without cache_creation", `{"input_tokens":15,"cache_read_input_tokens":100,"cache_creation_input_tokens":5,"output_tokens":40}`,
			llmprovider.Usage{InputTokens: 120, OutputTokens: 40, CachedTokens: 100, CacheWriteTokens: 5}},
	} {
		resp, err := Decode(strings.NewReader(`{"content":[{"type":"text","text":"hi"}],"usage":` + c.usage + `}`))
		if err != nil || resp.Usage != c.want {
			t.Errorf("%s: Usage = %+v (err %v), want %+v", c.name, resp.Usage, err, c.want)
		}
	}
}

// TestEvents_Usage: a stream's counts are cumulative. message_delta's
// replace message_start's, and a count it leaves out keeps message_start's.
func TestEvents_Usage(t *testing.T) {
	startWith := `{"type":"message_start","message":{"model":"m","usage":{"input_tokens":15,"cache_read_input_tokens":100,` +
		`"cache_creation_input_tokens":5,"cache_creation":{"ephemeral_5m_input_tokens":3,"ephemeral_1h_input_tokens":2},"output_tokens":1}}}`
	startBare := `{"type":"message_start","message":{"model":"m","usage":{"input_tokens":15,"output_tokens":1}}}`
	for _, c := range []struct{ name, start, final string }{
		{"message_delta with the output only", startWith, `{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":40}}`},
		{"message_delta repeating every count", startWith, `{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":` + cacheWriteUsage + `}`},
		{"message_delta carrying counts message_start left out", startBare,
			`{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":` + cacheWriteUsage + `}`},
	} {
		stream := sse(c.start, blockStart("0", `{"type":"text","text":""}`), delta("0", `{"type":"text_delta","text":"hi"}`),
			blockStop("0"), c.final, stop)
		events, _, err := collect(t, stream)
		if err != nil || len(events) == 0 || events[len(events)-1].Type != llmprovider.EventDone {
			t.Fatalf("%s: stream did not end in EventDone (err %v)", c.name, err)
		}
		if got := events[len(events)-1].Response.Usage; got != cacheWriteWant {
			t.Errorf("%s: Usage = %+v, want %+v", c.name, got, cacheWriteWant)
		}
	}
}
