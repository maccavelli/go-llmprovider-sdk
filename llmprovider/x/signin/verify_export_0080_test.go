package signin

import (
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/internal/signinflow"
)

// SetVerifyLimit is VerifyKilo's own bound, shortened for a test; it returns
// what puts the default back. It exists in test builds only, so the package
// exports nothing for it (0080-MADR D10).
func SetVerifyLimit(d time.Duration) (restore func()) {
	previous := verifyLimit
	verifyLimit = d
	return func() { verifyLimit = previous }
}

// TestVerifyLimitStartsAtTheDefault_0080 (the owner's ruling of 2026-11-17,
// relayed by the coordinator: "Its own 10 s bound; a stall is "cannot say""):
// VerifyKilo's own bound starts as the one constant the wizard's default
// discovery limit is too, and that constant is 10 s, as the decision says. The
// stall tests shorten the bound, so they cannot hold this.
func TestVerifyLimitStartsAtTheDefault_0080(t *testing.T) {
	if verifyLimit != signinflow.DefaultCheckLimit {
		t.Errorf("verifyLimit %v; want signinflow.DefaultCheckLimit, %v", verifyLimit, signinflow.DefaultCheckLimit)
	}
	if verifyLimit != 10*time.Second {
		t.Errorf("verifyLimit %v; want 10 s", verifyLimit)
	}
}
