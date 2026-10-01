package llmprovider

import (
	"strings"
	"testing"
)

// TestModelLabel was in descriptor_test.go, whose descriptors moved to the
// provider packages (0015-PLAN S8, commit 1).
func TestModelLabel(t *testing.T) {
	if got := ModelLabel(ProviderClaude, "claude-haiku-4-5"); !strings.Contains(got, "Haiku") {
		t.Errorf("ModelLabel = %q, want a human label", got)
	}
	// Unknown models degrade to the bare id rather than disappearing.
	if got := ModelLabel(ProviderKilo, "some/unlisted-model"); got != "some/unlisted-model" {
		t.Errorf("unknown model label = %q, want the bare id", got)
	}
	if got := ModelLabel(ProviderGemini, ""); got != "" {
		t.Errorf("empty model label = %q, want empty", got)
	}
}
