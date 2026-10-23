package providers

import (
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// caches holds each running test's catalog cache, by *testing.T.
var caches sync.Map

// testCache is t's own catalog cache, the same for every call in t. httptest
// reuses ports, so nothing a test caches under its server's URL may reach the
// process's cache, where a later test's server could read it (0058-MADR D1).
// A test's servers stay open until it ends, so none of them shares a port
// with another within it; a server closed sooner gets a cache of its own.
func testCache(t *testing.T) llmprovider.Option {
	t.Helper()
	c, loaded := caches.LoadOrStore(t, new(catalog.Cache))
	if !loaded {
		t.Cleanup(func() { caches.Delete(t) })
	}
	return catalog.WithCache(c.(*catalog.Cache))
}
