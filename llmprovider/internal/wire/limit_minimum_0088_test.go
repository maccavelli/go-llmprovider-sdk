package wire

import "testing"

// TestResponsesLimits_Minimum_0088 (0088-MADR ST113, R5): a limit the SDK
// chose, held on the Responses wire with no room left, reaches 1, as on every
// other wire. The Responses wire's minimum is 1 until a record makes one of its
// routes send a limit the SDK chose; that record re-adds W8's 16 (0089-MADR).
func TestResponsesLimits_Minimum_0088(t *testing.T) {
	r := &LimitRetry{Wire: ResponsesLimits}
	got, err := r.Choose(LimitInput{Sends: true, Output: 60000, Source: "layer", Window: 100000, Estimate: 100000})
	if err != nil || !got.Sent || got.Tokens != 1 {
		t.Errorf("Responses, output 60,000, W 100,000, an estimate of 100,000: sent %d (%v), err %v; want 1",
			got.Tokens, got.Sent, err)
	}
}
