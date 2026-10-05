package messages

import (
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestMessages_SkipsForeignReasoning (0021-MADR W7): a signature or encrypted
// reasoning another wire issued is not replayed here; the item is text only,
// which this wire cannot replay. This wire's, and an item with no Format, are.
func TestMessages_SkipsForeignReasoning(t *testing.T) {
	for _, c := range []struct {
		item llmprovider.ReasoningItem
		sent bool
	}{
		{llmprovider.ReasoningItem{Signature: "s", Text: "t", Format: "responses"}, false},
		{llmprovider.ReasoningItem{Encrypted: "e", Format: "chatcompletions"}, false},
		{llmprovider.ReasoningItem{Signature: "s", Text: "t", Format: "messages"}, true},
		{llmprovider.ReasoningItem{Signature: "s", Text: "t"}, true},
	} {
		if got := FromItems([]llmprovider.Item{c.item}); (len(got) == 1) != c.sent {
			t.Errorf("%+v: %v; want sent %v", c.item, got, c.sent)
		}
	}
}
