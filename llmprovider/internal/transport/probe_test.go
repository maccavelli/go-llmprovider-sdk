package transport

import (
	"context"
	"fmt"
	"slices"
	"testing"
)

// Ported from llmprovider's probe_test.go (0015-PLAN S7b), with the limit
// passed in.

func TestProbeGenerateHealth(t *testing.T) {
	// Empty candidates
	if res := ProbeGenerateHealth(context.Background(), nil, 3, nil); res != nil {
		t.Errorf("expected nil for empty candidates, got %v", res)
	}

	// 2 candidates: m1 succeeds with "Hello", m2 fails with error, m3 succeeds without "hello"
	candidates := []string{"m1", "m2", "m3"}
	gen := func(_ context.Context, modelID string) (string, error) {
		switch modelID {
		case "m1":
			return "Hello world", nil
		case "m2":
			return "", fmt.Errorf("timeout")
		case "m3":
			return "bad response", nil
		}
		return "", nil
	}

	healthy := ProbeGenerateHealth(context.Background(), candidates, 3, gen)
	if len(healthy) != 1 || healthy[0] != "m1" {
		t.Errorf("expected only [m1], got %v", healthy)
	}
}

// TestProbeGenerateHealth_Limit: no more than limit candidates are probed, in
// order. The rest come back unprobed (0038-MADR D4).
func TestProbeGenerateHealth_Limit(t *testing.T) {
	probe, calls := failingProbe()
	got := ProbeGenerateHealth(context.Background(), []string{"a", "b", "c"}, 2, probe)
	if !slices.Equal(got, []string{"a", "b", "c"}) || calls.Load() != 2 {
		t.Errorf("ProbeGenerateHealth = %v after %d probes, want [a b c] after 2", got, calls.Load())
	}
}
