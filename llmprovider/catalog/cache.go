package catalog

import (
	"sync"

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
}

// value is c's option value, made on first use.
func (c *Cache) value() *cacheref.Value {
	c.once.Do(func() {
		c.ref = cacheref.New(newStore())
	})
	return c.ref
}

// WithCache makes a listing (List, ListRaw, the lookups) or a built-in
// provider's New use c in place of the process's cache. A provider given it
// uses it for its listings, its model facts, its probes and the refusals it
// learns. A nil c is the
// process's cache, as no option is. The last WithCache given wins.
func WithCache(c *Cache) llmprovider.Option {
	var v *cacheref.Value
	if c != nil {
		v = c.value()
	}
	return llmprovider.ScopedOptionFor(builtinIDs, "catalog.WithCache", v)
}

// store is one cache's listings and metadata documents, each under its
// own lock.
type store struct {
	listingMu sync.Mutex
	listing   map[listingKey]listingEntry
	// listingInFlight holds the background fetches running, under listingMu.
	listingInFlight map[listingFetchKey]bool
	// listingClaims holds, under listingMu, the listings being fetched by a
	// List, each closed when its fetch ends (0041-MADR D6).
	listingClaims map[listingKey]chan struct{}
	// listingFetching counts the background fetches, so tests can wait.
	listingFetching sync.WaitGroup

	metadataMu sync.Mutex
	metadata   map[string]modelMetadataCacheEntry
	// metadataFetches holds each URL's fetch in flight, under metadataMu.
	metadataFetches map[string]*metadataFetch
	// metadataFetching counts the fetches in flight, so that tests can wait
	// for a background refresh.
	metadataFetching sync.WaitGroup
}

func newStore() *store {
	return &store{
		listing:         map[listingKey]listingEntry{},
		listingInFlight: map[listingFetchKey]bool{},
		listingClaims:   map[listingKey]chan struct{}{},
		metadata:        map[string]modelMetadataCacheEntry{},
		metadataFetches: map[string]*metadataFetch{},
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
