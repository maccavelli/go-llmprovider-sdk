package responses

import (
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestInput_SkipsForeignReasoning (0021-MADR W7): encrypted reasoning another
// wire issued is not sent here; this wire's, and an item with no Format, are.
func TestInput_SkipsForeignReasoning(t *testing.T) {
	for format, sent := range map[string]bool{"messages": false, "responses": true, "": true} {
		input := Input([]llmprovider.Item{llmprovider.ReasoningItem{Encrypted: "e", Format: format}})
		if (len(input) == 1) != sent {
			t.Errorf("Format %q: input %v; want sent %v", format, input, sent)
		}
	}
}
