package transport

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
)

// TestProbeGenerateHealth_TimeoutKeepsTheModel (0020-MADR F21, Q3 a): only a
// probe that failed, or answered wrongly, drops a model. One that timed out,
// such as a model still loading, stays listed.
func TestProbeGenerateHealth_TimeoutKeepsTheModel(t *testing.T) {
	old := probeTimeout
	probeTimeout = 100 * time.Millisecond
	t.Cleanup(func() { probeTimeout = old })
	got := ProbeGenerateHealth(context.Background(), []string{"ok", "broken", "wrong", "slow"}, 10,
		func(ctx context.Context, model string) (string, error) {
			switch model {
			case "ok":
				return "hello", nil
			case "broken":
				return "", errors.New("model failed to load")
			case "wrong":
				return "goodbye", nil
			default: // slow: still loading when the probe gives up
				<-ctx.Done()
				return "", ctx.Err()
			}
		})
	if want := []string{"ok", "slow"}; !slices.Equal(got, want) {
		t.Errorf("ProbeGenerateHealth = %v, want %v", got, want)
	}
}
