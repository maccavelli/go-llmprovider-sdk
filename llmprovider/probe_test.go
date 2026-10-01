package llmprovider

import (
	"context"
	"fmt"
	"testing"
)

func TestProbeGenerateHealth(t *testing.T) {
	// Empty candidates
	if res := ProbeGenerateHealth(context.Background(), nil, nil); res != nil {
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

	healthy := ProbeGenerateHealth(context.Background(), candidates, gen)
	if len(healthy) != 1 || healthy[0] != "m1" {
		t.Errorf("expected only [m1], got %v", healthy)
	}
}
