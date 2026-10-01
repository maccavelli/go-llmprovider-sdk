//go:build live_gateways

// liveEnvKey, for the live tests of this package and, through
// live_export_test.go, the external ones. The thinking-shape tests that were
// here are in live_claude_test.go, live_gemini_test.go and
// live_opencode_test.go (0015-PLAN S7).
package llmprovider

import (
	"os"
	"testing"
)

// liveEnvKey returns a credential from the environment or skips.
func liveEnvKey(t *testing.T, name string) string {
	t.Helper()
	key := os.Getenv(name)
	if key == "" {
		t.Skipf("%s unset", name)
	}
	return key
}
