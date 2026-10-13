package transport

import (
	"context"
	"errors"
	"slices"
	"sync/atomic"
	"testing"
)

// 0038-MADR D4: a full listing probes only its first limit ids. A probed id
// that fails is dropped, every later id is kept unprobed, and the order holds.

// nineIDs is a listing longer than the probe limit.
var nineIDs = []string{"m1", "m2", "m3", "m4", "m5", "m6", "m7", "m8", "m9"}

// failingProbe answers "hello" for every model but those in fail, and counts
// its calls.
func failingProbe(fail ...string) (func(context.Context, string) (string, error), *atomic.Int32) {
	var calls atomic.Int32
	return func(_ context.Context, model string) (string, error) {
		calls.Add(1)
		if slices.Contains(fail, model) {
			return "", errors.New("probe failed")
		}
		return "hello", nil
	}, &calls
}

// TestProbe_KeepsTheTail: the failed head id is dropped, and the ids past the
// limit come back unprobed, in order.
func TestProbe_KeepsTheTail(t *testing.T) {
	probe, calls := failingProbe("m2")
	got := ProbeGenerateHealth(context.Background(), nineIDs, 6, probe)
	want := []string{"m1", "m3", "m4", "m5", "m6", "m7", "m8", "m9"}
	if !slices.Equal(got, want) || calls.Load() != 6 {
		t.Errorf("ProbeGenerateHealth = %v after %d probes; want %v after 6", got, calls.Load(), want)
	}
}

// TestProbe_AllHeadFail: when no probe answers, the result is empty, so the
// provider returns its listing unchanged, as before 0038.
func TestProbe_AllHeadFail(t *testing.T) {
	probe, calls := failingProbe(nineIDs...)
	if got := ProbeGenerateHealth(context.Background(), nineIDs, 6, probe); got != nil || calls.Load() != 6 {
		t.Errorf("ProbeGenerateHealth = %v after %d probes; want nil after 6", got, calls.Load())
	}
}

// TestProbe_ShortList: a list within the limit is probed whole, and only its
// healthy ids come back.
func TestProbe_ShortList(t *testing.T) {
	probe, calls := failingProbe("m3")
	got := ProbeGenerateHealth(context.Background(), nineIDs[:4], 6, probe)
	if want := []string{"m1", "m2", "m4"}; !slices.Equal(got, want) || calls.Load() != 4 {
		t.Errorf("ProbeGenerateHealth = %v after %d probes; want %v after 4", got, calls.Load(), want)
	}
}

// TestProbeCached_KeepsTheTail: a cached listing returns the cached head and
// the current tail, and sends no probe.
func TestProbeCached_KeepsTheTail(t *testing.T) {
	resetProbeCache(t)
	key := ProbeKey{Provider: "openai", BaseURL: "https://example.test", Credential: Fingerprint("k")}
	probe, calls := failingProbe("m2")
	want := []string{"m1", "m3", "m4", "m5", "m6", "m7", "m8", "m9"}
	for i := range 2 {
		if got := ProbeGenerateHealthCached(context.Background(), key, nineIDs, 6, probe); !slices.Equal(got, want) {
			t.Errorf("listing %d: %v; want %v", i+1, got, want)
		}
	}
	if calls.Load() != 6 {
		t.Errorf("%d probes in all; want 6, the second listing answered from the cache", calls.Load())
	}
}
