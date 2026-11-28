package responses

import (
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestTerminalFill_LastCallDropped_0095 (0095-MADR FU02): a cut stream whose
// server-last call is incomplete, and so dropped and absent from the
// response, after a whole call that takes no arguments. The dropped call was
// the one the limit cut; nothing else is cut, so the earlier call is kept
// and yielded, and the dropped one is named.
func TestTerminalFill_LastCallDropped_0095(t *testing.T) {
	callA := tfCall("fc_a", "call_a", "now", "{}")
	callB := `{"id":"fc_b","type":"function_call","status":"incomplete","call_id":"call_b","name":"record_city","arguments":"{\"ci"}`
	stream := tfCreated + tfAdded(0, callA) + tfDone(0, callA) + tfAdded(1, callB) + tfDone(1, callB) + tfCut(callA, callB)
	r := tfRead(Decoder{}, stream)
	r.check(t, "the last call dropped as incomplete", llmprovider.FinishLength, []string{"call:now {}"})
	if r.done == nil {
		return
	}
	named := 0
	for _, w := range r.done.Warnings {
		if w.Kind == llmprovider.WarningToolCallCut {
			named++
		}
	}
	if named != 1 {
		t.Errorf("warnings %+v; want one WarningToolCallCut, for the dropped call", r.done.Warnings)
	}
}
