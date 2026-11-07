package catalog

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// 0071-PLAN-give-a-cache-a-lifetime-and-wait-for-a-refresh.md Phase 1: a
// Cache's lifetime (0071-MADR D1, D2).

// lingering0071 is how long work under test lingers after its context ends,
// so that a Close that returns before the work has ended is seen to.
const lingering0071 = 50 * time.Millisecond

// gatedSource0071 gives its token at once for its first free calls. Each
// later call reports on entered, then waits for its context to end and
// lingers; or, when stubborn, waits for release alone.
type gatedSource0071 struct {
	free     int32
	stubborn bool
	calls    atomic.Int32
	entered  chan struct{}
	release  chan struct{}
}

func newGatedSource0071(free int32, stubborn bool) *gatedSource0071 {
	return &gatedSource0071{free: free, stubborn: stubborn, entered: make(chan struct{}, 8), release: make(chan struct{})}
}

func (s *gatedSource0071) Token(ctx context.Context) (llmprovider.Token, error) {
	token := llmprovider.Token{Value: "key-0071"}
	if s.calls.Add(1) <= s.free {
		return token, nil
	}
	select {
	case s.entered <- struct{}{}:
	default:
	}
	if s.stubborn {
		<-s.release
		return token, nil
	}
	<-ctx.Done()
	time.Sleep(lingering0071)
	return llmprovider.Token{}, ctx.Err()
}

// slowTransport0071 sends nothing: each request reports on entered, waits
// for its context to end, and lingers.
type slowTransport0071 struct{ entered chan struct{} }

func (t slowTransport0071) RoundTrip(r *http.Request) (*http.Response, error) {
	select {
	case t.entered <- struct{}{}:
	default:
	}
	<-r.Context().Done()
	time.Sleep(lingering0071)
	return nil, r.Context().Err()
}

func newSlowClient0071() (*http.Client, chan struct{}) {
	entered := make(chan struct{}, 8)
	return &http.Client{Transport: slowTransport0071{entered: entered}}, entered
}

// waitEntered0071 fails t unless entered reports within two seconds.
func waitEntered0071(t *testing.T, entered <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatalf("%s never started", what)
	}
}

// closeWithin0071 is c.Close under a deadline of d.
func closeWithin0071(c *Cache, d time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	return c.Close(ctx)
}

// inFlight0071 counts what the cache's store has in flight: background
// listing fetches, claimed listings and models.dev fetches.
func inFlight0071(c *Cache) (fetches, claims, metadata int) {
	c.value()
	s := c.st
	s.listingMu.Lock()
	fetches, claims = len(s.listingInFlight), len(s.listingClaims)
	s.listingMu.Unlock()
	s.metadataMu.Lock()
	metadata = len(s.metadataFetches)
	s.metadataMu.Unlock()
	return fetches, claims, metadata
}

// listingOpts0071 point Together's listing at s, through c, without the
// models.dev document.
func listingOpts0071(s *lookupServer, c *Cache, more ...llmprovider.Option) []llmprovider.Option {
	return append([]llmprovider.Option{llmprovider.WithBaseURL(s.URL), llmprovider.WithoutModelMetadata(), WithCache(c)},
		more...)
}

// TestCacheClose_WaitsForFetch_0071: Close cancels a CachedModel fetch in
// flight and returns nil only once it has ended (0071-MADR D1).
func TestCacheClose_WaitsForFetch_0071(t *testing.T) {
	s := newLookupServer(t, lookupDoc, http.StatusOK, togetherListing0033)
	c := new(Cache)
	src := newGatedSource0071(1, false)
	CachedModel(context.Background(), llmprovider.ProviderTogether, src, "org/together-x", listingOpts0071(s, c)...)
	waitEntered0071(t, src.entered, "the background fetch")
	if err := closeWithin0071(c, 2*time.Second); err != nil {
		t.Fatalf("Close = %v; want nil once the fetch has ended", err)
	}
	if fetches, _, _ := inFlight0071(c); fetches != 0 {
		t.Errorf("Close returned with %d fetch(es) in flight; want 0", fetches)
	}
}

// TestCacheClose_MetadataFetch_0071: the same for a models.dev fetch.
func TestCacheClose_MetadataFetch_0071(t *testing.T) {
	c := new(Cache)
	client, entered := newSlowClient0071()
	CachedMetadataWith(context.Background(), llmprovider.ProviderTogether, WithCache(c),
		llmprovider.WithModelMetadataURL("http://127.0.0.1:1/api.json"), llmprovider.WithHTTPClient(client))
	waitEntered0071(t, entered, "the models.dev fetch")
	if err := closeWithin0071(c, 2*time.Second); err != nil {
		t.Fatalf("Close = %v; want nil once the fetch has ended", err)
	}
	if _, _, metadata := inFlight0071(c); metadata != 0 {
		t.Errorf("Close returned with %d models.dev fetch(es) in flight; want 0", metadata)
	}
}

// TestCacheClose_Deadline_0071: with a fetch that ignores its cancellation,
// Close returns ctx's error at its deadline; once the fetch ends, a second
// Close returns nil, since each call waits under its own ctx (0071-MADR Q5).
func TestCacheClose_Deadline_0071(t *testing.T) {
	s := newLookupServer(t, lookupDoc, http.StatusOK, togetherListing0033)
	c := new(Cache)
	src := newGatedSource0071(1, true)
	CachedModel(context.Background(), llmprovider.ProviderTogether, src, "org/together-x", listingOpts0071(s, c)...)
	waitEntered0071(t, src.entered, "the background fetch")
	start := time.Now()
	err := closeWithin0071(c, 100*time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Close with a stubborn fetch = %v; want context.DeadlineExceeded", err)
	}
	if took := time.Since(start); took > 2*time.Second {
		t.Errorf("Close took %v past a 100ms deadline", took)
	}
	close(src.release)
	if err := closeWithin0071(c, 2*time.Second); err != nil {
		t.Errorf("second Close after the fetch was released = %v; want nil", err)
	}
}

// TestCacheClose_NoStartAfter_0071: after Close, CachedModel and
// CachedMetadataWith start nothing (0071-MADR D2).
func TestCacheClose_NoStartAfter_0071(t *testing.T) {
	s := newLookupServer(t, lookupDoc, http.StatusOK, togetherListing0033)
	c := new(Cache)
	if err := closeWithin0071(c, time.Second); err != nil {
		t.Fatal(err)
	}
	src := newGatedSource0071(1, false)
	client, _ := newSlowClient0071()
	CachedModel(context.Background(), llmprovider.ProviderTogether, src, "org/together-x", listingOpts0071(s, c)...)
	CachedMetadataWith(context.Background(), llmprovider.ProviderTogether, WithCache(c),
		llmprovider.WithModelMetadataURL("http://127.0.0.1:1/api.json"), llmprovider.WithHTTPClient(client))
	if fetches, claims, metadata := inFlight0071(c); fetches+claims+metadata != 0 {
		t.Errorf("after Close: %d fetch(es), %d claim(s), %d models.dev fetch(es) started; want none",
			fetches, claims, metadata)
	}
	// Anything a broken start began ends before the test does.
	_ = closeWithin0071(c, 2*time.Second)
	if n := s.listings.Load(); n != 0 {
		t.Errorf("%d listing request(s) after Close; want 0", n)
	}
}

// TestCacheClose_Refuses_0071: after Close, List, LookupModel and
// LookupMetadataWith given the cache refuse before the network, even with
// a fresh listing and document cached (0071-MADR D2, Q1, Q4).
func TestCacheClose_Refuses_0071(t *testing.T) {
	s := newLookupServer(t, lookupDoc, http.StatusOK, togetherListing0033)
	c := new(Cache)
	src := llmprovider.NewStaticToken("key-0071-refuses")
	opts := append(lookupOpts(s), WithCache(c))
	ctx := context.Background()
	// Warm the cache, so that only the refusal stands between a call and
	// an answer.
	if _, err := List(ctx, llmprovider.ProviderTogether, src, opts...); err != nil {
		t.Fatalf("List before Close: %v", err)
	}
	if _, err := LookupModel(ctx, llmprovider.ProviderTogether, src, "org/together-x", opts...); err != nil {
		t.Fatalf("LookupModel before Close: %v", err)
	}
	if _, err := LookupMetadataWith(ctx, llmprovider.ProviderTogether, opts...); err != nil {
		t.Fatalf("LookupMetadataWith before Close: %v", err)
	}
	before := s.listings.Load()
	if err := closeWithin0071(c, time.Second); err != nil {
		t.Fatal(err)
	}
	_, listErr := List(ctx, llmprovider.ProviderTogether, src, opts...)
	_, lookupErr := LookupModel(ctx, llmprovider.ProviderTogether, src, "org/together-x", opts...)
	_, metaErr := LookupMetadataWith(ctx, llmprovider.ProviderTogether, opts...)
	for name, err := range map[string]error{"List": listErr, "LookupModel": lookupErr, "LookupMetadataWith": metaErr} {
		if !errors.Is(err, llmprovider.ErrInvalidRequest) || err.Error() != errCacheClosed.Error() {
			t.Errorf("%s after Close = %v; want %v", name, err, errCacheClosed)
		}
	}
	if n := s.listings.Load(); n != before {
		t.Errorf("%d listing request(s) after Close; want 0", n-before)
	}
}

// TestCacheClose_CountUnderLock_0071: a start that has passed its closed
// check is waited for by a Close that runs then, because the start counts
// its work before it releases the lock (0071-MADR D1, fact 1). The hook holds
// each start between its check and its count while Close runs. A start that
// counted after releasing the lock would let Close return first.
func TestCacheClose_CountUnderLock_0071(t *testing.T) {
	for _, site := range []string{"startListingFetch", "claimListing"} {
		t.Run(site, func(t *testing.T) {
			s := newLookupServer(t, lookupDoc, http.StatusOK, togetherListing0033)
			c := new(Cache)
			type result struct {
				err                       error
				fetches, claims, metadata int
			}
			closed := make(chan result, 1)
			var once sync.Once
			listingStartHook = func() {
				once.Do(func() {
					returned := make(chan struct{})
					go func() {
						err := closeWithin0071(c, 2*time.Second)
						f, cl, m := inFlight0071(c)
						closed <- result{err, f, cl, m}
						close(returned)
					}()
					// Under the lock, Close cannot run until the start has
					// counted its work. Give a broken start the time to let
					// it.
					select {
					case <-returned:
					case <-time.After(200 * time.Millisecond):
					}
				})
			}
			t.Cleanup(func() { listingStartHook = nil })
			client, _ := newSlowClient0071()
			opts := listingOpts0071(s, c, llmprovider.WithHTTPClient(client))
			src := llmprovider.NewStaticToken("key-0071-" + site)
			if site == "startListingFetch" {
				CachedModel(context.Background(), llmprovider.ProviderTogether, src, "org/together-x", opts...)
			} else {
				_, _ = List(context.Background(), llmprovider.ProviderTogether, src, opts...)
			}
			var r result
			select {
			case r = <-closed:
			case <-time.After(5 * time.Second):
				t.Fatal("Close never returned")
			}
			if r.err != nil {
				t.Fatalf("Close = %v; want nil", r.err)
			}
			if r.fetches+r.claims+r.metadata != 0 {
				t.Errorf("Close returned with %d fetch(es), %d claim(s), %d models.dev fetch(es) in flight; want none",
					r.fetches, r.claims, r.metadata)
			}
		})
	}
}

// TestCacheClose_Race_0071: 64 goroutines use one cache while it closes.
// Under -race nothing races, no group panics, and nothing is in flight or
// sent once Close has returned.
func TestCacheClose_Race_0071(t *testing.T) {
	s := newLookupServer(t, lookupDoc, http.StatusOK, togetherListing0033)
	c := new(Cache)
	opts := append(lookupOpts(s), WithCache(c))
	src := llmprovider.NewStaticToken("key-0071-race")
	probes := c.value().Probes
	var closedFlag atomic.Bool
	var late atomic.Int32
	generate := func(context.Context, string) (string, error) {
		if closedFlag.Load() {
			late.Add(1)
		}
		return "hello", nil
	}
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range 64 {
		wg.Go(func() {
			<-start
			ctx := context.Background()
			switch i % 3 {
			case 0:
				CachedModel(ctx, llmprovider.ProviderTogether, src, "org/together-x", opts...)
			case 1:
				_, _ = List(ctx, llmprovider.ProviderTogether, src, opts...)
			default:
				key := transport.ProbeKey{Provider: "together", BaseURL: s.URL, Credential: strconv.Itoa(i)}
				transport.ProbeGenerateHealthCached(ctx, probes, key, []string{"org/together-x"}, 1, generate)
			}
		})
	}
	close(start)
	if err := closeWithin0071(c, 5*time.Second); err != nil {
		t.Fatalf("Close = %v; want nil", err)
	}
	closedFlag.Store(true)
	sent := s.listings.Load()
	if fetches, claims, metadata := inFlight0071(c); fetches+claims+metadata != 0 {
		t.Errorf("Close returned with %d fetch(es), %d claim(s), %d models.dev fetch(es) in flight", fetches, claims, metadata)
	}
	wg.Wait()
	if fetches, claims, metadata := inFlight0071(c); fetches+claims+metadata != 0 {
		t.Errorf("after Close, %d fetch(es), %d claim(s), %d models.dev fetch(es) started", fetches, claims, metadata)
	}
	if n := s.listings.Load(); n != sent {
		t.Errorf("%d listing request(s) sent after Close returned", n-sent)
	}
	if n := late.Load(); n != 0 {
		t.Errorf("%d probe(s) ran after Close returned", n)
	}
}

// TestCacheClose_ProbeRun_0071: Close cancels a probe run through the
// cache's probe store, and returns only once it has ended (0071-MADR D1).
func TestCacheClose_ProbeRun_0071(t *testing.T) {
	c := new(Cache)
	entered := make(chan struct{}, 1)
	var ended atomic.Bool
	key := transport.ProbeKey{Provider: "together", BaseURL: "http://127.0.0.1:1", Credential: "key-0071-probe"}
	go transport.ProbeGenerateHealthCached(context.Background(), c.value().Probes, key, []string{"org/together-x"}, 1,
		func(ctx context.Context, _ string) (string, error) {
			entered <- struct{}{}
			<-ctx.Done()
			time.Sleep(lingering0071)
			ended.Store(true)
			return "", ctx.Err()
		})
	waitEntered0071(t, entered, "the probe run")
	if err := closeWithin0071(c, 2*time.Second); err != nil {
		t.Fatalf("Close = %v; want nil once the probe run has ended", err)
	}
	if !ended.Load() {
		t.Error("Close returned before the probe run had ended")
	}
}

// TestCachedOrFetched_NoSecondLeader_0071: a caller that found no fresh
// listing, and claims only after another caller's fetch has stored one and
// ended, takes that listing and sends no second request (0071-PLAN
// Deviation 4; 0053-MADR D4). The hook holds the second caller in that gap
// while the first lists to the end.
func TestCachedOrFetched_NoSecondLeader_0071(t *testing.T) {
	s := newLookupServer(t, lookupDoc, http.StatusOK, togetherListing0033)
	c := new(Cache)
	src := llmprovider.NewStaticToken("key-0071-leader")
	opts := listingOpts0071(s, c)
	var held atomic.Bool
	var firstErr error
	listingClaimHook = func() {
		if held.CompareAndSwap(false, true) {
			_, firstErr = List(context.Background(), llmprovider.ProviderTogether, src, opts...)
		}
	}
	t.Cleanup(func() { listingClaimHook = nil })
	cat, err := List(context.Background(), llmprovider.ProviderTogether, src, opts...)
	if err != nil || firstErr != nil {
		t.Fatalf("List = %v; the first, %v", err, firstErr)
	}
	if !cat.Live {
		t.Errorf("the held caller's listing is not live: %+v", cat)
	}
	if n := s.listings.Load(); n != 1 {
		t.Errorf("%d listing request(s) for two callers of one key; want 1", n)
	}
}

// TestCacheClose_Zero_0071: a Cache never used closes at once, and is
// closed when it is given to a call later (0071-MADR D1).
func TestCacheClose_Zero_0071(t *testing.T) {
	s := newLookupServer(t, lookupDoc, http.StatusOK, togetherListing0033)
	var c Cache
	if err := closeWithin0071(&c, time.Second); err != nil {
		t.Fatalf("Close of an unused Cache = %v; want nil", err)
	}
	_, err := List(context.Background(), llmprovider.ProviderTogether, llmprovider.NewStaticToken("key-0071-zero"),
		listingOpts0071(s, &c)...)
	if !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Errorf("List given a Cache closed before its first use = %v; want the closed-cache refusal", err)
	}
	if n := s.listings.Load(); n != 0 {
		t.Errorf("%d listing request(s); want 0", n)
	}
}
