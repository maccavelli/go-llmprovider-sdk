package transport

import (
	"context"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// 0028-MADR D-A3: the models that answered a probe are cached for ten
// minutes, keyed by provider, base URL and the credential's fingerprint.

// countingProbe answers "hello" for every model, or fails every one, and
// counts its calls.
func countingProbe(healthy bool) (func(context.Context, string) (string, error), *atomic.Int32) {
	var calls atomic.Int32
	return func(context.Context, string) (string, error) {
		calls.Add(1)
		if !healthy {
			return "", context.Canceled
		}
		return "hello", nil
	}, &calls
}

// resetProbeCache empties the cache on both sides of a test.
func resetProbeCache(t *testing.T) {
	t.Helper()
	clear := func() {
		probeCacheMu.Lock()
		defer probeCacheMu.Unlock()
		probeCache = map[ProbeKey]probeCacheEntry{}
	}
	clear()
	t.Cleanup(clear)
}

// TestProbeCache_ExpiresAfterTTL: within the TTL a listing is answered from
// the cache; past it, the models are probed again.
func TestProbeCache_ExpiresAfterTTL(t *testing.T) {
	resetProbeCache(t)
	key := ProbeKey{Provider: "openai", BaseURL: "https://example.test", Credential: Fingerprint("k")}
	models := []string{"a", "b", "c"}
	probe, calls := countingProbe(true)
	if got := ProbeGenerateHealthCached(context.Background(), key, models, 10, probe); len(got) != 3 || calls.Load() != 3 {
		t.Fatalf("first: %v after %d probes; want 3 models after 3", got, calls.Load())
	}
	if got := ProbeGenerateHealthCached(context.Background(), key, models, 10, probe); len(got) != 3 || calls.Load() != 3 {
		t.Fatalf("within the TTL: %v after %d probes; want the cached 3, no probe", got, calls.Load())
	}
	later := time.Now().Add(probeCacheTTL + time.Second)
	probeCacheNow = func() time.Time { return later }
	t.Cleanup(func() { probeCacheNow = time.Now })
	if _ = ProbeGenerateHealthCached(context.Background(), key, models, 10, probe); calls.Load() != 6 {
		t.Errorf("past the TTL: %d probes in all; want 6", calls.Load())
	}
}

// TestProbeCache_EmptyResultNotCached: when every probe fails, nothing is
// cached, and the next listing probes again.
func TestProbeCache_EmptyResultNotCached(t *testing.T) {
	resetProbeCache(t)
	key := ProbeKey{Provider: "claude", BaseURL: "https://example.test", Credential: Fingerprint("k")}
	probe, calls := countingProbe(false)
	for range 2 {
		if got := ProbeGenerateHealthCached(context.Background(), key, []string{"a", "b"}, 10, probe); len(got) != 0 {
			t.Fatalf("every probe failed, yet %v answered", got)
		}
	}
	if n := calls.Load(); n != 4 {
		t.Errorf("probes = %d; want 4, both listings probing", n)
	}
}

// TestProbeCache_ReturnsACopy: a caller that changes the slice it was given,
// probed or cached, does not change the cache.
func TestProbeCache_ReturnsACopy(t *testing.T) {
	resetProbeCache(t)
	key := ProbeKey{Provider: "gemini", BaseURL: "https://example.test"}
	probe, _ := countingProbe(true)
	for i := range 3 {
		got := ProbeGenerateHealthCached(context.Background(), key, []string{"a", "b"}, 10, probe)
		if got[0] != "a" {
			t.Fatalf("call %d = %v; want it unchanged by an earlier caller", i, got)
		}
		got[0] = "changed"
	}
}

// TestProbeCache_FingerprintOnly: a credential becomes 64 hexadecimal
// characters of SHA-256, never itself; no credential stays empty.
func TestProbeCache_FingerprintOnly(t *testing.T) {
	const secret = "sk-proj-not-a-real-key-0123456789"
	fp := Fingerprint(secret)
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(fp) || strings.Contains(fp, secret) {
		t.Errorf("Fingerprint = %q; want 64 hex characters", fp)
	}
	if fp != Fingerprint(secret) || fp == Fingerprint(secret+"x") {
		t.Error("Fingerprint is not a function of the credential alone")
	}
	if Fingerprint("") != "" {
		t.Errorf("Fingerprint(\"\") = %q; want \"\"", Fingerprint(""))
	}
}
