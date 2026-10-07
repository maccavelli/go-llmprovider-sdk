package llmprovider

import (
	"testing"
	"time"
)

// TestRetry_ServerWaitNeverExceedsMaxDelay (0026-MADR F22): "MaxDelay caps
// each wait" holds for a wait the service asked for, after its jitter too.
func TestRetry_ServerWaitNeverExceedsMaxDelay(t *testing.T) {
	p := RetryPolicy{MaxDelay: 100 * time.Millisecond}.withDefaults()
	for _, asked := range []time.Duration{90 * time.Millisecond, 100 * time.Millisecond} {
		err := &APIError{Kind: ErrRateLimited, Status: 429, RetryAfter: asked}
		longest := time.Duration(0)
		for range 50 {
			d, ok := p.wait(1, err)
			if !ok {
				t.Fatalf("a wait of %s within MaxDelay %s was refused", asked, p.MaxDelay)
			}
			longest = max(longest, d)
		}
		if longest > p.MaxDelay || longest < asked {
			t.Errorf("asked %s: longest wait over 50 runs = %s; want within [%s, %s]", asked, longest, asked, p.MaxDelay)
		}
	}
}
