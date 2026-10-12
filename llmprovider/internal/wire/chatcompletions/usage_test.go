package chatcompletions

import (
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Usage counts as 0015-MADR amendment "what `Usage` counts" defines them:
// totals that hold their parts.

// TestDecode_Usage reads cached tokens from the three spellings pi reads
// (0017-REPORT P7), and Together's plain counts.
func TestDecode_Usage(t *testing.T) {
	choice := `"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"hi"}}]`
	for _, c := range []struct {
		name, usage string
		want        llmprovider.Usage
	}{
		{"prompt_tokens_details", `{"prompt_tokens":120,"completion_tokens":40,"prompt_tokens_details":{"cached_tokens":100},` +
			`"completion_tokens_details":{"reasoning_tokens":30}}`,
			llmprovider.Usage{InputTokens: 120, OutputTokens: 40, ReasoningTokens: 30, CachedTokens: 100}},
		{"prompt_cache_hit_tokens", `{"prompt_tokens":120,"completion_tokens":40,"prompt_cache_hit_tokens":90}`,
			llmprovider.Usage{InputTokens: 120, OutputTokens: 40, CachedTokens: 90}},
		{"cached_tokens", `{"prompt_tokens":120,"completion_tokens":40,"cached_tokens":80}`,
			llmprovider.Usage{InputTokens: 120, OutputTokens: 40, CachedTokens: 80}},
		{"details first, as pi reads them", `{"prompt_tokens":120,"completion_tokens":40,"prompt_tokens_details":{"cached_tokens":100},` +
			`"prompt_cache_hit_tokens":90,"cached_tokens":80}`,
			llmprovider.Usage{InputTokens: 120, OutputTokens: 40, CachedTokens: 100}},
		{"together", `{"prompt_tokens":7,"completion_tokens":3,"total_tokens":10}`,
			llmprovider.Usage{InputTokens: 7, OutputTokens: 3}},
	} {
		resp, err := Decode(strings.NewReader(`{"id":"c",` + choice + `,"usage":` + c.usage + `}`))
		if err != nil || resp.Usage != c.want {
			t.Errorf("%s: Usage = %+v (err %v), want %+v", c.name, resp.Usage, err, c.want)
		}
	}
	resp, err := Decode(strings.NewReader(`{"id":"c",` + choice + `}`))
	if err != nil || resp.Usage != (llmprovider.Usage{}) {
		t.Errorf("not reported: Usage = %+v (err %v), want zero", resp.Usage, err)
	}
}

// The cache-write count (0036-MADR D2, and its amendment of 2026-10-12):
// prompt_tokens_details.cache_write_tokens, else Kilo's top-level
// cache_creation_input_tokens, inside prompt_tokens.
func TestDecode_CacheWrites(t *testing.T) {
	choice := `"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"hi"}}]`
	want := llmprovider.Usage{InputTokens: 120, OutputTokens: 40, CachedTokens: 100, CacheWriteTokens: 5}
	for _, c := range []struct{ name, usage string }{
		{"cache_write_tokens", `{"prompt_tokens":120,"completion_tokens":40,"prompt_tokens_details":{"cached_tokens":100,"cache_write_tokens":5}}`},
		{"Kilo's top-level spelling", `{"prompt_tokens":120,"completion_tokens":40,"prompt_tokens_details":{"cached_tokens":100},"cache_creation_input_tokens":5}`},
		{"the details before the top level", `{"prompt_tokens":120,"completion_tokens":40,` +
			`"prompt_tokens_details":{"cached_tokens":100,"cache_write_tokens":5},"cache_creation_input_tokens":7}`},
	} {
		resp, err := Decode(strings.NewReader(`{"id":"c",` + choice + `,"usage":` + c.usage + `}`))
		if err != nil || resp.Usage != want {
			t.Errorf("%s: Usage = %+v (err %v), want %+v", c.name, resp.Usage, err, want)
		}
	}
}

// TestEvents_Usage: a stream's usage is its last chunk's, as
// stream_options.include_usage asks, in either write spelling.
func TestEvents_Usage(t *testing.T) {
	want := llmprovider.Usage{InputTokens: 9, OutputTokens: 4, CachedTokens: 2, CacheWriteTokens: 1}
	for _, c := range []struct{ name, usage string }{
		{"cache_write_tokens", `{"prompt_tokens":9,"completion_tokens":4,"prompt_tokens_details":{"cached_tokens":2,"cache_write_tokens":1}}`},
		{"Kilo's top-level spelling", `{"prompt_tokens":9,"completion_tokens":4,"prompt_tokens_details":{"cached_tokens":2},"cache_creation_input_tokens":1}`},
	} {
		stream := chunks(d(`{"role":"assistant","content":"hi"}`, ""), d(`{}`, "stop"),
			`{"id":"c1","model":"m","choices":[],"usage":`+c.usage+`}`)
		events, _, err := collect(t, stream)
		if err != nil || len(events) == 0 || events[len(events)-1].Type != llmprovider.EventDone {
			t.Fatalf("%s: stream did not end in EventDone (err %v)", c.name, err)
		}
		if got := events[len(events)-1].Response.Usage; got != want {
			t.Errorf("%s: Usage = %+v, want %+v", c.name, got, want)
		}
	}
}
