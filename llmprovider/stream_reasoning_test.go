package llmprovider

import "testing"

// TestResponseEvents_NoEmptyReasoningDelta (0020-MADR F24): reasoning held
// only in encrypted form has no text to stream; it still arrives as an item.
func TestResponseEvents_NoEmptyReasoningDelta(t *testing.T) {
	events := responseEvents(&Response{Output: []Item{ReasoningItem{Encrypted: "ENC"}, ReasoningItem{Text: "plan"}}})
	var deltas, items int
	for _, e := range events {
		switch e.Type {
		case EventReasoningDelta:
			deltas++
			if e.Text == "" {
				t.Error("an empty reasoning delta was streamed")
			}
		case EventItem:
			items++
		}
	}
	if deltas != 1 || items != 2 {
		t.Errorf("%d reasoning deltas and %d items, want 1 and 2", deltas, items)
	}
}
