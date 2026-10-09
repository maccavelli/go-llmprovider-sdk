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

// ProbeGenerateHealth runs a tiny generate against each candidate and returns
// those that did not fail, preserving preferred order. At most limit
// candidates are probed; the providers pass llmprovider.MaxListedModels. A
// probe that failed, or answered wrongly, drops its model. One that timed out,
// such as a model a local server is still loading, keeps it (0020-MADR F21,
// Q3 a).
func ProbeGenerateHealth(ctx context.Context, preferred []string, limit int, generate func(ctx context.Context, modelID string) (string, error)) []string {
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

// ProbeKey identifies one listing's probes: the provider, its base URL, and
// the fingerprint of the credential the probes sent (0028-MADR D-A3).
// Credential is Fingerprint's, never the credential.
type ProbeKey struct {
	Provider, BaseURL, Credential string
}

// probeCacheTTL is how long the models that answered a probe are kept (Q7).
const probeCacheTTL = 10 * time.Minute

// probeCacheEntry is one listing's healthy models and when they were probed.
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
// cached, so the next listing probes again (0028-MADR D-A3).
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
		return slices.Clone(e.healthy)
	}
	delete(probeCache, key)
	probeCacheMu.Unlock()
	healthy := ProbeGenerateHealth(ctx, preferred, limit, generate)
	if len(healthy) > 0 {
		probeCacheMu.Lock()
		probeCache[key] = probeCacheEntry{healthy: slices.Clone(healthy), at: probeCacheNow()}
		probeCacheMu.Unlock()
	}
	return healthy
}
