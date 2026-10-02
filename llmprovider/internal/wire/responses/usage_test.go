package responses

import (
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Usage counts as 0015-MADR amendment "what `Usage` counts" defines them:
// totals that hold their parts.

func TestDecode_Usage(t *testing.T) {
	for _, c := range []struct {
		name, body string
		want       llmprovider.Usage
	}{
		{"reported", `{"id":"r","status":"completed","output":[],"usage":{"input_tokens":120,` +
			`"input_tokens_details":{"cached_tokens":100},"output_tokens":40,"output_tokens_details":{"reasoning_tokens":30},"total_tokens":160}}`,
			llmprovider.Usage{InputTokens: 120, OutputTokens: 40, ReasoningTokens: 30, CachedTokens: 100}},
		{"not reported", `{"id":"r","status":"completed","output":[]}`, llmprovider.Usage{}},
	} {
		resp, err := Decode(strings.NewReader(c.body))
		if err != nil || resp.Usage != c.want {
			t.Errorf("%s: Usage = %+v (err %v), want %+v", c.name, resp.Usage, err, c.want)
		}
	}
}

// TestReadStream_Usage: a stream's usage is response.completed's.
func TestReadStream_Usage(t *testing.T) {
	stream := "data: {\"type\":\"response.created\",\"response\":{\"id\":\"r\"}}\n\n" +
		"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"r\",\"usage\":{\"input_tokens\":12," +
		"\"input_tokens_details\":{\"cached_tokens\":2},\"output_tokens\":5,\"output_tokens_details\":{\"reasoning_tokens\":3}}}}\n\n"
	resp, err := ReadStream("p", strings.NewReader(stream))
	want := llmprovider.Usage{InputTokens: 12, OutputTokens: 5, ReasoningTokens: 3, CachedTokens: 2}
	if err != nil || resp.Usage != want {
		t.Fatalf("Usage = %+v (err %v), want %+v", resp.Usage, err, want)
	}
}
