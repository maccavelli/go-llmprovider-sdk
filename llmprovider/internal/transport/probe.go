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
// Credential is Fingerprint's, never the credential. Scope parts one
// credential's probes further, such as by Anthropic's workspace, and is
// empty for none (0064-MADR, amendment "the probe cache").
type ProbeKey struct {
	Provider, BaseURL, Credential, Scope string
}

// probeCacheTTL is how long the models that answered a probe are kept (Q7).
const probeCacheTTL = 10 * time.Minute

// probeCacheEntry is one listing's healthy probed models and when they were
// probed.
type probeCacheEntry struct {
	healthy []string
	at      time.Time
}

// ProbeStore holds probe results in memory, by ProbeKey: the process's, or a
// catalog.Cache's (0048-MADR D1). Make one with NewProbeStore. It is safe for
// concurrent use.
type ProbeStore struct {
	mu sync.Mutex
	m  map[ProbeKey]probeCacheEntry
	// flights holds, under mu, each key's probe run in flight (0053-MADR D6).
	flights map[ProbeKey]*probeFlight
	// closed is set by Close, under mu. A closed store starts no run
	// (0071-MADR D1).
	closed bool
	// running counts the runs in flight. Each is counted under mu, so that
	// no run starts after Close has set closed.
	running sync.WaitGroup
	// life is cancelled by Close, and ends every run in flight. A zero store
	// has none, and its runs end only by their own bounds.
	life context.Context
	end  context.CancelFunc
}

// probeFlight is one run of a listing's probes, and the models that
// answered, which every caller waiting on it receives. healthy is written
// once, before done is closed.
type probeFlight struct {
	done    chan struct{}
	healthy []string
}

// NewProbeStore returns an empty store.
func NewProbeStore() *ProbeStore {
	life, end := context.WithCancel(context.Background())
	return &ProbeStore{m: map[ProbeKey]probeCacheEntry{}, flights: map[ProbeKey]*probeFlight{}, life: life, end: end}
}

// Close stops s from starting runs and cancels the runs in flight
// (0071-MADR D1). What s holds stays readable. Calling it again does
// nothing more. It does not wait; Wait does.
func (s *ProbeStore) Close() {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	if s.end != nil {
		s.end()
	}
}

// Wait returns once no run s started is in flight. It is called after
// Close: before it, a run could start while Wait waits on none, which
// sync.WaitGroup forbids.
func (s *ProbeStore) Wait() {
	s.running.Wait()
}

var (
	// defaultProbes is the process's probe results, in memory only.
	defaultProbes = NewProbeStore()
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

// ProbeGenerateHealthCached is ProbeGenerateHealth, answered from s for ten
// minutes after a listing for the same key probed: a listing within that time
// sends no probe. A nil s is the process's store (0048-MADR D1). A result with
// no model is not cached, so the next listing probes again (0028-MADR D-A3).
// The store holds the probed head only; the current listing's tail is
// appended to it (0038-MADR D4).
//
// One run of probes per key at a time: listings that miss the cache while
// a run is in flight wait for it and share its result, empty included, so N
// concurrent listings send one run's probes. The run is detached from every
// caller, each probe still bounded by probeTimeout; a caller whose ctx ends
// first gets what a run with no answer gives, and its provider keeps its
// listing (0053-MADR D6).
//
// The run is s's: Close cancels it, and Wait waits for it. A closed s starts
// no run, and answers what a run with no answer gives, unless it holds a
// fresh result or a run is already in flight (0071-MADR D1, D2).
func ProbeGenerateHealthCached(
	ctx context.Context,
	s *ProbeStore,
	key ProbeKey,
	preferred []string,
	limit int,
	generate func(ctx context.Context, modelID string) (string, error),
) []string {
	if s == nil {
		s = defaultProbes
	}
	s.mu.Lock()
	e, ok := s.m[key]
	if ok && probeCacheNow().Sub(e.at) < probeCacheTTL {
		s.mu.Unlock()
		return withTail(slices.Clone(e.healthy), preferred, limit)
	}
	delete(s.m, key)
	if s.flights == nil {
		s.flights = map[ProbeKey]*probeFlight{}
	}
	flight, running := s.flights[key]
	if !running && s.closed {
		// A closed store starts no run: the caller gets what a run with no
		// answer gives, and its provider keeps its listing (0071-MADR D2).
		s.mu.Unlock()
		return withTail(nil, preferred, limit)
	}
	if !running {
		flight = &probeFlight{done: make(chan struct{})}
		s.flights[key] = flight
		detached, cancel := context.WithCancel(context.WithoutCancel(ctx))
		stop := func() bool { return false }
		if s.life != nil {
			stop = context.AfterFunc(s.life, cancel)
		}
		s.running.Go(func() {
			defer cancel()
			defer stop()
			healthy := probeHead(detached, preferred, limit, generate)
			s.mu.Lock()
			if len(healthy) > 0 {
				s.m[key] = probeCacheEntry{healthy: slices.Clone(healthy), at: probeCacheNow()}
			}
			flight.healthy = healthy
			delete(s.flights, key)
			s.mu.Unlock()
			close(flight.done)
		})
	}
	s.mu.Unlock()
	select {
	case <-flight.done:
		return withTail(slices.Clone(flight.healthy), preferred, limit)
	case <-ctx.Done():
		return withTail(nil, preferred, limit)
	}
}
