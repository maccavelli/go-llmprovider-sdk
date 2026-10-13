package transport

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"slices"
	"strings"
	"sync"
	"time"
)

// probeTimeout bounds one probe; tests shorten it.
var probeTimeout = 5 * time.Second

// ProbeGenerateHealth runs a tiny generate against each of the first limit
// candidates, and returns those that did not fail followed by every later
// candidate, unprobed, in preferred order (0038-MADR D4). The providers pass
// catalog.MaxListed. A probe that failed, or answered wrongly, drops its
// model. One that timed out, such as a model a local server is still
// loading, keeps it (0020-MADR F21, Q3 a). When no probe answered, it returns
// nil, and the provider keeps its listing.
func ProbeGenerateHealth(ctx context.Context, preferred []string, limit int, generate func(ctx context.Context, modelID string) (string, error)) []string {
	return withTail(probeHead(ctx, preferred, limit, generate), preferred, limit)
}

// probeHead probes the first limit candidates and returns those that
// answered, in preferred order.
func probeHead(ctx context.Context, preferred []string, limit int, generate func(ctx context.Context, modelID string) (string, error)) []string {
	if len(preferred) == 0 {
		return nil
	}
	candidates := preferred
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	type result struct {
		id    string
		ok    bool
		index int
	}
	ch := make(chan result, len(candidates))
	var wg sync.WaitGroup

	for i, id := range candidates {
		wg.Add(1)
		go func(idx int, modelID string) {
			defer wg.Done()
			tCtx, cancel := context.WithTimeout(ctx, probeTimeout)
			defer cancel()
			out, err := generate(tCtx, modelID)
			ok := err == nil && strings.Contains(strings.ToLower(out), "hello")
			// The probe's own timeout, not the caller's: the model is slow,
			// not broken.
			timedOut := err != nil && errors.Is(tCtx.Err(), context.DeadlineExceeded) && ctx.Err() == nil
			ok = ok || timedOut
			ch <- result{id: modelID, ok: ok, index: idx}
		}(i, id)
	}
	go func() {
		wg.Wait()
		close(ch)
	}()

	healthy := make(map[string]struct{})
	for r := range ch {
		if r.ok {
			healthy[strings.ToLower(r.id)] = struct{}{}
		}
	}

	var out []string
	for _, id := range candidates {
		if _, ok := healthy[strings.ToLower(id)]; ok {
			out = append(out, id)
		}
	}
	return out
}

// withTail appends the candidates past limit, unprobed, to the probed head,
// skipping any the head already holds. An empty head stays nil.
func withTail(head, preferred []string, limit int) []string {
	if len(head) == 0 || len(preferred) <= limit {
		return head
	}
	out := slices.Clone(head)
	for _, id := range preferred[max(limit, 0):] {
		if !slices.ContainsFunc(out, func(h string) bool { return strings.EqualFold(h, id) }) {
			out = append(out, id)
		}
	}
	return out
}

// ProbeKey identifies one listing's probes: the provider, its base URL, and
// the fingerprint of the credential the probes sent (0028-MADR D-A3).
// Credential is Fingerprint's, never the credential.
type ProbeKey struct {
	Provider, BaseURL, Credential string
}

// probeCacheTTL is how long the models that answered a probe are kept (Q7).
const probeCacheTTL = 10 * time.Minute

// probeCacheEntry is one listing's healthy probed models and when they were
// probed.
type probeCacheEntry struct {
	healthy []string
	at      time.Time
}

var (
	probeCacheMu sync.Mutex
	// probeCache is the process's probe results, in memory only.
	probeCache = map[ProbeKey]probeCacheEntry{}
	// probeCacheNow is the cache's clock. Tests move it.
	probeCacheNow = time.Now
)

// Fingerprint is the SHA-256 of a credential, in hexadecimal: the probe
// cache's key holds it, never the credential. No credential is "".
func Fingerprint(credential string) string {
	if credential == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(credential))
	return hex.EncodeToString(sum[:])
}

// ProbeGenerateHealthCached is ProbeGenerateHealth, answered from the
// process's cache for ten minutes after a listing for the same key probed:
// a listing within that time sends no probe. A result with no model is not
// cached, so the next listing probes again (0028-MADR D-A3). The cache holds
// the probed head only; the current listing's tail is appended to it
// (0038-MADR D4).
func ProbeGenerateHealthCached(
	ctx context.Context,
	key ProbeKey,
	preferred []string,
	limit int,
	generate func(ctx context.Context, modelID string) (string, error),
) []string {
	probeCacheMu.Lock()
	e, ok := probeCache[key]
	if ok && probeCacheNow().Sub(e.at) < probeCacheTTL {
		probeCacheMu.Unlock()
		return withTail(slices.Clone(e.healthy), preferred, limit)
	}
	delete(probeCache, key)
	probeCacheMu.Unlock()
	healthy := probeHead(ctx, preferred, limit, generate)
	if len(healthy) > 0 {
		probeCacheMu.Lock()
		probeCache[key] = probeCacheEntry{healthy: slices.Clone(healthy), at: probeCacheNow()}
		probeCacheMu.Unlock()
	}
	return withTail(healthy, preferred, limit)
}
