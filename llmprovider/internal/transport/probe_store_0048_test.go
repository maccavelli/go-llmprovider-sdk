package transport

import (
	"context"
	"testing"
)

// TestProbeStore_KeepsProbesApart_0048: two stores, and the process's, do not
// share a probe result (0048-MADR D1).
func TestProbeStore_KeepsProbesApart_0048(t *testing.T) {
	resetProbeCache(t)
	key := ProbeKey{Provider: "openai", BaseURL: "https://example.test", Credential: Fingerprint("k-0048")}
	models := []string{"a", "b", "c"}
	probe, calls := countingProbe(true)
	a, b := NewProbeStore(), NewProbeStore()
	for i, step := range []struct {
		store *ProbeStore
		want  int32
	}{
		{a, 3}, {a, 3}, {b, 6}, {nil, 9}, {nil, 9},
	} {
		if got := ProbeGenerateHealthCached(context.Background(), step.store, key, models, 10, probe); len(got) != 3 {
			t.Fatalf("step %d: %v, want three models", i, got)
		}
		if got := calls.Load(); got != step.want {
			t.Fatalf("step %d: %d probes in all, want %d", i, got, step.want)
		}
	}
	if got := ProbeGenerateHealthCached(context.Background(), nil, key, models, 10, probe); len(got) != 3 || calls.Load() != 9 {
		t.Errorf("ProbeGenerateHealthCached(nil) after the process's store probed: %v, %d probes; want the cached 3, 9", got, calls.Load())
	}
}
