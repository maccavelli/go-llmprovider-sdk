package catalog

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/cacheref"
)

// Cache holds what the catalog and a provider keep in memory between calls:
// the listings and their models' metadata, the models.dev document, the
// models that answered a probe (0048-MADR D1), and the models a provider
// learned refuse a forced tool choice (0049-MADR D3). Without WithCache, every
// listing and provider of the process shares one; a Cache given with WithCache
// is shared only by the calls and providers that are given it, so a test or a
// process with more than one configuration can keep them apart.
//
// The zero value is ready to use. A Cache is safe for concurrent use. It must
// not be copied after its first use.
type Cache struct {
	once sync.Once
	ref  *cacheref.Value
	// st is ref's catalog state, kept typed for Close.
	st *store
}

// value is c's option value, made on first use.
func (c *Cache) value() *cacheref.Value {
	c.once.Do(func() {
		c.st = newStore()
		c.ref = cacheref.New(c.st)
	})
	return c.ref
}

// WithCache makes a listing (List, ListRaw, the lookups) or a provider's New
// use c in place of the process's cache. A built-in provider, or a chatcompat
// one, given it uses it for its listings, its model facts, its probes and the
// refusals it learns. It is a shared option (llmprovider.SharedOption): every
// provider's New takes it, and a provider outside this module, which cannot
// read it, ignores it (0061-MADR, amendment "Phase 4's options"). A nil c is
// the process's cache, as no option is. The last WithCache given wins.
func WithCache(c *Cache) llmprovider.Option {
	var v *cacheref.Value
	if c != nil {
		v = c.value()
	}
	return llmprovider.SharedOption("catalog.WithCache", v)
}

// store is one cache's listings and metadata documents, each under its
// own lock.
type store struct {
	listingMu sync.Mutex
	listing   map[listingKey]listingEntry
	// listingInFlight holds the background fetches running, under listingMu.
	listingInFlight map[listingFetchKey]bool
	// listingClaims holds, under listingMu, the listings being fetched by a
	// List, each with its result for every caller that waits on it
	// (0041-MADR D6; 0053-MADR D4).
	listingClaims map[listingKey]*listingCall
	// ollamaShows holds, under listingMu, Ollama's /api/show replies by
	// listing and model, each with its own age and digest (0053-MADR D5).
	ollamaShows map[listingKey]map[string]showReply
	// listingFetching counts the background fetches, so tests can wait.
	listingFetching sync.WaitGroup

	metadataMu sync.Mutex
	metadata   map[string]modelMetadataCacheEntry
	// metadataFetches holds each URL's fetch in flight, under metadataMu.
	metadataFetches map[string]*metadataFetch
	// metadataFetching counts the fetches in flight, so that tests can wait
	// for a background refresh.
	metadataFetching sync.WaitGroup

	// listingClosed and metadataClosed are set by close, each under its own
	// lock, where each start reads it. A start takes its count under the
	// same lock, so none is counted after close has set the flag
	// (0071-MADR D1).
	listingClosed  bool
	metadataClosed bool
	// life is cancelled by close and ends the background work in flight. The
	// process's store is never closed.
	life context.Context
	end  context.CancelFunc
}

func newStore() *store {
	life, end := context.WithCancel(context.Background())
	return &store{
		listing:         map[listingKey]listingEntry{},
		listingInFlight: map[listingFetchKey]bool{},
		listingClaims:   map[listingKey]*listingCall{},
		ollamaShows:     map[listingKey]map[string]showReply{},
		metadata:        map[string]modelMetadataCacheEntry{},
		metadataFetches: map[string]*metadataFetch{},
		life:            life,
		end:             end,
	}
}

// errCacheClosed refuses a call that would fetch through a closed Cache
// (0071-MADR D2, Q4).
var errCacheClosed = fmt.Errorf("catalog: the cache is closed: %w", llmprovider.ErrInvalidRequest)

// listingStartHook is nil outside tests. A test sets it to run between a
// start's closed check and its count, which both happen under listingMu
// (0071-PLAN Phase 1, TestCacheClose_CountUnderLock_0071).
var listingStartHook func()

// listingClaimHook is nil outside tests. A test sets it to run between a
// listing's freshness check and its claim, the gap a second leader came
// through (0071-PLAN Deviation 4, TestCachedOrFetched_NoSecondLeader_0071).
var listingClaimHook func()

// detached is the context for background work started from ctx. It keeps
// ctx's values but not its cancellation, is bounded by timeout, and ends
// when the store's life ends (0053-MADR D4; 0071-MADR D1). The caller
// defers the returned cancel.
func (c *store) detached(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	work, cancel := context.WithTimeout(context.WithoutCancel(ctx), timeout)
	stop := context.AfterFunc(c.life, cancel)
	return work, func() {
		stop()
		cancel()
	}
}

// refuseClosed is errCacheClosed once the store is closed, else nil.
func (c *store) refuseClosed() error {
	c.listingMu.Lock()
	defer c.listingMu.Unlock()
	if c.listingClosed {
		return errCacheClosed
	}
	return nil
}

// close stops the store from starting background work, then cancels the
// work in flight. Calling it again does nothing more.
func (c *store) close() {
	c.listingMu.Lock()
	c.listingClosed = true
	c.listingMu.Unlock()
	c.metadataMu.Lock()
	c.metadataClosed = true
	c.metadataMu.Unlock()
	c.end()
}

// wait returns once no background work the store started is in flight. It
// is called after close, so no count is taken while it waits.
func (c *store) wait() {
	c.listingFetching.Wait()
	c.metadataFetching.Wait()
}

// Close ends c's life (0071-MADR D1, D2). No call given c starts background
// work after Close begins: no listing fetch, models.dev refresh or probe
// run. Close cancels the work in flight, which ends at its next network
// wait, and waits for it. It returns nil once that work has ended, or
// ctx.Err() if ctx ends first. A Close that returned ctx.Err() can be called
// again to wait longer: closing happens once, and each call waits under its
// own ctx (0071-MADR Q5).
//
// After Close, CachedModel, CachedFacts and CachedMetadataWith given c
// answer from what c holds and start nothing. List, LookupModel and
// LookupMetadataWith given c refuse before the network with an error
// matching llmprovider.ErrInvalidRequest. A provider given c takes its own
// path for a failed listing. Ollama's and a chatcompat provider's
// ListModels return the error. Every other built-in provider's returns its
// static list. A probe through c starts no run. A request given c reads what
// c holds, as before. CachedModel still asks its
// source for a token, in the caller's goroutine. An auth.OAuthSession's
// refresh that a fetch started is the session's, not c's: call Close, then
// the session's Wait.
//
// A Cache that was never used closes at once, and stays closed when it is
// given to a call later. The process's cache, used without WithCache, has
// no Close. Close is safe to call concurrently with every other use of c.
func (c *Cache) Close(ctx context.Context) error {
	v := c.value()
	s := c.st
	s.close()
	v.Probes.Close()
	done := make(chan struct{})
	go func() {
		s.wait()
		v.Probes.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		// Work that ended as ctx did has ended.
		select {
		case <-done:
			return nil
		default:
			return ctx.Err()
		}
	}
}

// processStore is the process's store: every listing's and provider's
// without WithCache.
var processStore = newStore()

// storeIn is the store a listing's option values name, or the process's.
func storeIn(values []any) *store {
	if v := cacheref.From(values); v != nil {
		if s, ok := v.Catalog.(*store); ok {
			return s
		}
	}
	return processStore
}
