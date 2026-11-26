package wire

import (
	"errors"
	"iter"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// reasoningStream yields mib reasoning deltas of 1 MiB each, then the done
// event, as a Responses stream that sends a long reasoning reply does.
func reasoningStream(mib int) iter.Seq2[llmprovider.Event, error] {
	chunk := strings.Repeat("r", 1<<20)
	return func(yield func(llmprovider.Event, error) bool) {
		for range mib {
			if !yield(llmprovider.Event{Type: llmprovider.EventReasoningDelta, Text: chunk}, nil) {
				return
			}
		}
		yield(llmprovider.Event{Type: llmprovider.EventDone, Response: &llmprovider.Response{ID: "r1"}}, nil)
	}
}

// TestCollect_StreamAnswerLimit_0091 (0091-MADR ST71, "Raise the bound"): an
// answer read over the stream is bounded at 32 MiB, reasoning counted: 20
// MiB of reasoning, over the old 16 MiB, is returned; 33 MiB is
// ErrIncomplete, so a runaway stream still stops.
func TestCollect_StreamAnswerLimit_0091(t *testing.T) {
	if resp, err := Collect("p", reasoningStream(20)); err != nil || resp.ID != "r1" {
		t.Errorf("20 MiB of reasoning: %v, %v; want the response", resp, err)
	}
	if _, err := Collect("p", reasoningStream(33)); !errors.Is(err, llmprovider.ErrIncomplete) {
		t.Errorf("33 MiB of reasoning: %v; want ErrIncomplete", err)
	}
}
