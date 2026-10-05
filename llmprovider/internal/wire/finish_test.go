package wire

import (
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestFinish (0021-MADR W3, and its amendment "the empty finish reason and
// the cut answer's reason"): a call turns "stop", or no reason, into
// "tool_calls"; any other reason, and a reply without a call, keep theirs.
func TestFinish(t *testing.T) {
	table := map[string]llmprovider.FinishReason{"end_turn": llmprovider.FinishStop, "max_tokens": llmprovider.FinishLength}
	for _, c := range []struct {
		raw     string
		hasCall bool
		want    llmprovider.FinishReason
	}{
		{"end_turn", true, llmprovider.FinishToolCalls},
		{"", true, llmprovider.FinishToolCalls},
		{"", false, ""},
		{"end_turn", false, llmprovider.FinishStop},
		{"max_tokens", true, llmprovider.FinishLength},
		{"pause_turn", true, "pause_turn"},
	} {
		if got := Finish(c.raw, table, c.hasCall); got != c.want {
			t.Errorf("Finish(%q, call %v) = %q, want %q", c.raw, c.hasCall, got, c.want)
		}
	}
}
