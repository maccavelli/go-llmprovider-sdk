package responses

import (
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// cutWarnings0088 is the run's WarningToolCallCut details.
func cutWarnings0088(r tfRun) []string {
	if r.done == nil {
		return nil
	}
	var out []string
	for _, w := range r.done.Warnings {
		if w.Kind == llmprovider.WarningToolCallCut {
			out = append(out, w.Detail)
		}
	}
	return out
}

// TestCutAnswer_OpenCallNoCallID_0088 (0088-MADR ST115, "Fix before v1.34.0
// (Recommended)"; ST116, "Match by call ID, else item ID (Recommended)"): an
// open call is closed only by a finished item that matches its call_id, else
// its id. A call announced with no call_id and never delivered is named at the
// cut, though a message's .done, which carries no call_id either, came after
// it. One delivered with no call_id, by its .done or from the terminal, is
// kept, with no warning.
func TestCutAnswer_OpenCallNoCallID_0088(t *testing.T) {
	msg := tfMsg("msg_1", "Hi")
	whole := tfCall("fc_1", "", "lookup", `{"x":1}`)
	lost := tfCreated + tfAdded(0, tfCall("fc_1", "", "lookup", "")) + tfAdded(1, tfMsg("msg_1", "")) +
		tfDone(1, msg) + tfCut(msg)
	r := tfRead(Decoder{}, lost)
	r.check(t, "a call with no call_id, never done", llmprovider.FinishLength, []string{"message:Hi"})
	if w := cutWarnings0088(r); len(w) != 1 || !strings.Contains(w[0], `"lookup"`) {
		t.Errorf("a call with no call_id, never done: warnings %q; want one WarningToolCallCut naming lookup", w)
	}

	for _, tc := range []struct {
		name, stream string
		want         []string
	}{
		{"delivered by its .done", tfCreated + tfAdded(0, tfCall("fc_1", "", "lookup", "")) + tfDone(0, whole) +
			tfAdded(1, tfMsg("msg_1", "")) + tfDone(1, msg) + tfCut(whole, msg),
			[]string{`call:lookup {"x":1}`, "message:Hi"}},
		{"taken from the terminal", tfCreated + tfAdded(0, tfCall("fc_1", "", "lookup", "")) +
			tfAdded(1, tfMsg("msg_1", "")) + tfDone(1, msg) + tfCut(whole, msg),
			[]string{"message:Hi", `call:lookup {"x":1}`}},
	} {
		r := tfRead(Decoder{}, tc.stream)
		r.check(t, "a call with no call_id, "+tc.name, llmprovider.FinishLength, tc.want)
		if w := cutWarnings0088(r); len(w) != 0 {
			t.Errorf("a call with no call_id, %s: warnings %q; want none, the call was delivered", tc.name, w)
		}
	}

	// ST116 as narrowed (2026-11-27): two different call_ids never match
	// through an id, so a finished call_2 sharing call_1's id leaves call_1
	// open, named at the cut.
	other := tfCall("fc_1", "call_2", "other", `{"x":1}`)
	twoIDs := tfCreated + tfAdded(0, tfCall("fc_1", "call_1", "lookup", "")) + tfDone(1, other) + tfCut(other)
	r = tfRead(Decoder{}, twoIDs)
	r.check(t, "two call_ids, one id", llmprovider.FinishLength, []string{`call:other {"x":1}`})
	if w := cutWarnings0088(r); len(w) != 1 || !strings.Contains(w[0], "call_1") {
		t.Errorf("two call_ids, one id: warnings %q; want one WarningToolCallCut naming call_1", w)
	}
}
