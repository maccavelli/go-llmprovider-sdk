package transport

import (
	"context"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// slowProbe answers "hello" for every model after delay, and counts its
// calls.
func slowProbe(delay time.Duration) (func(context.Context, string) (string, error), *atomic.Int32) {
	var calls atomic.Int32
	return func(ctx context.Context, _ string) (string, error) {
		calls.Add(1)
		select {
		case <-time.After(delay):
			return "hello", nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}, &calls
}

var preferred0053 = []string{"a", "b", "c", "d", "e", "f", "g"}

// TestProbeFlight_SharedRun_0053 (0053-MADR D6, M7): five concurrent cold
// listings of one key send one run of probes, six, and each gets the
// healthy head and its own tail. Each sent its own: thirty.
func TestProbeFlight_SharedRun_0053(t *testing.T) {
	s := NewProbeStore()
	generate, calls := slowProbe(50 * time.Millisecond)
	key := ProbeKey{Provider: "p", BaseURL: "u", Credential: Fingerprint("k")}
	var wg sync.WaitGroup
	results := make([][]string, 5)
	for i := range 5 {
		wg.Go(func() {
			results[i] = ProbeGenerateHealthCached(context.Background(), s, key, preferred0053, 6, generate)
		})
	}
	wg.Wait()
	if n := calls.Load(); n != 6 {
		t.Errorf("%d probes for five concurrent listings; want 6", n)
	}
	for i, got := range results {
		if !slices.Equal(got, preferred0053) {
			t.Errorf("listing %d: %q; want %q, the head that answered and the tail", i, got, preferred0053)
		}
	}
}

// TestProbeFlight_CancelledClaimant_0053 (0053-MADR D6): the listing that
// starts the probes and is cancelled does not end them: a listing waiting on
// the run receives the models that answered, and the run is cached.
func TestProbeFlight_CancelledClaimant_0053(t *testing.T) {
	s := NewProbeStore()
	generate, calls := slowProbe(80 * time.Millisecond)
	key := ProbeKey{Provider: "p", BaseURL: "u", Credential: Fingerprint("k")}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	claimant := make(chan []string)
	go func() { claimant <- ProbeGenerateHealthCached(ctx, s, key, preferred0053, 6, generate) }()
	time.Sleep(5 * time.Millisecond)
	waiter := ProbeGenerateHealthCached(context.Background(), s, key, preferred0053, 6, generate)
	if got := <-claimant; got != nil {
		t.Errorf("the cancelled claimant: %q; want nil, as a run with no answer gives", got)
	}
	if !slices.Equal(waiter, preferred0053) || calls.Load() != 6 {
		t.Errorf("the waiter: %q after %d probes; want %q after 6", waiter, calls.Load(), preferred0053)
	}
	if again := ProbeGenerateHealthCached(context.Background(), s, key, preferred0053, 6, generate); !slices.Equal(again, preferred0053) ||
		calls.Load() != 6 {
		t.Errorf("a later listing: %q after %d probes; want the cached run, no probe", again, calls.Load())
	}
}

// TestProbeFlight_EmptyRunShared_0053 (0053-MADR D6, Q1): a run in which no
// probe answered is shared with its waiters and not cached past it, so the
// next listing probes again (0028-MADR D-A3).
func TestProbeFlight_EmptyRunShared_0053(t *testing.T) {
	s := NewProbeStore()
	var calls atomic.Int32
	generate := func(context.Context, string) (string, error) {
		calls.Add(1)
		time.Sleep(50 * time.Millisecond)
		return "no", nil // answered wrongly: dropped
	}
	key := ProbeKey{Provider: "p", BaseURL: "u", Credential: Fingerprint("k")}
	var wg sync.WaitGroup
	for range 3 {
		wg.Go(func() { _ = ProbeGenerateHealthCached(context.Background(), s, key, preferred0053, 6, generate) })
	}
	wg.Wait()
	first := calls.Load()
	_ = ProbeGenerateHealthCached(context.Background(), s, key, preferred0053, 6, generate)
	if first != 6 || calls.Load() != 12 {
		t.Errorf("%d probes for three concurrent listings, then %d more; want 6, then 6", first, calls.Load()-first)
	}
}
