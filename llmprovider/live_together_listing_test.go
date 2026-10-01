//go:build live_gateways

package llmprovider

import (
	"os"
	"testing"
)

// Together's live listing test, which reads the listing directly; the
// generation tests build the provider through its own package, in
// live_together_test.go (0015-PLAN S7).

// togetherLiveKey skips unless LLMPROVIDER_LIVE_TOGETHER=1 and TOGETHER_API_KEY
// are set: every call is billed.
func togetherLiveKey(t *testing.T) string {
	t.Helper()
	if os.Getenv("LLMPROVIDER_LIVE_TOGETHER") != "1" {
		t.Skip("LLMPROVIDER_LIVE_TOGETHER unset: live Together calls are billed")
	}
	return liveEnvKey(t, "TOGETHER_API_KEY")
}
