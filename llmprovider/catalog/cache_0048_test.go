package catalog_test

import (
	"context"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
)

// togetherListing0048 is one Together model, whose context window the
// listing states, so that a cache holding it answers CachedFacts.
const togetherListing0048 = `[{"id": "org/together-x", "type": "chat", "context_length": 1048575,
 "pricing": {"input": 0.3, "output": 1.2, "cached_input": 0.006, "hourly": 0}}]`

// listingServer serves Together's listing and counts its requests.
func listingServer(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/models") {
			http.NotFound(w, r)
			return
		}
		n.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(togetherListing0048))
	}))
	t.Cleanup(srv.Close)
	return srv, &n
}

// keys0048 holds each running test's key, by *testing.T.
var keys0048 sync.Map

// key0048 is the test's own key, one of this run's, so that no other test,
// nor this one in another -count run, shares the process cache's entry: the
// listing is keyed by its base URL, and httptest reuses ports (0058-MADR D1).
func key0048(t *testing.T) string {
	k, loaded := keys0048.LoadOrStore(t, "key-0048-"+rand.Text())
	if !loaded {
		t.Cleanup(func() { keys0048.Delete(t) })
	}
	return k.(string)
}

// listOpts are a listing's options for srv, with the test's key.
func listOpts(t *testing.T, srv *httptest.Server, more ...llmprovider.Option) []llmprovider.Option {
	return append([]llmprovider.Option{llmprovider.WithAPIKey(key0048(t)),
		llmprovider.WithBaseURL(srv.URL + "/v1"), llmprovider.WithoutModelMetadata()}, more...)
}

func list(t *testing.T, opts []llmprovider.Option) {
	t.Helper()
	if _, err := catalog.List(context.Background(), llmprovider.ProviderTogether, llmprovider.NewStaticToken(key0048(t)), opts...); err != nil {
		t.Fatalf("List: %v", err)
	}
}

// TestCache_KeepsListingsApart_0048: a listing is fetched once per cache; two
// caches, and the process's, do not share it (0048-MADR D1).
func TestCache_KeepsListingsApart_0048(t *testing.T) {
	srv, n := listingServer(t)
	var a, b catalog.Cache // zero values, ready to use
	for i, step := range []struct {
		opts []llmprovider.Option
		want int32
	}{
		{listOpts(t, srv, catalog.WithCache(&a)), 1},
		{listOpts(t, srv, catalog.WithCache(&a)), 1},
		{listOpts(t, srv, catalog.WithCache(&b)), 2},
		{listOpts(t, srv), 3},
		{listOpts(t, srv, catalog.WithCache(nil)), 3},
		{listOpts(t, srv, catalog.WithCache(&b), catalog.WithCache(&a)), 3},
	} {
		list(t, step.opts)
		if got := n.Load(); got != step.want {
			t.Fatalf("step %d: %d listing request(s), want %d", i, got, step.want)
		}
	}
}

// TestCache_ProviderListsIntoItsCache_0048: a provider built with WithCache
// lists into that cache, and CachedFacts finds the model there and not in
// another one.
func TestCache_ProviderListsIntoItsCache_0048(t *testing.T) {
	srv, _ := listingServer(t)
	var a, b catalog.Cache
	p, err := providers.New(llmprovider.ProviderTogether, listOpts(t, srv, catalog.WithCache(&a))...)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.(llmprovider.ModelLister).ListModels(context.Background()); err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	info, ok := catalog.CachedFacts(context.Background(), llmprovider.ProviderTogether, "org/together-x",
		listOpts(t, srv, catalog.WithCache(&a))...)
	if !ok || info.ContextWindow != 1048575 {
		t.Fatalf("CachedFacts from the provider's cache: %v, context window %d; want true, 1048575", ok, info.ContextWindow)
	}
	if _, ok := catalog.CachedFacts(context.Background(), llmprovider.ProviderTogether, "org/together-x",
		listOpts(t, srv, catalog.WithCache(&b))...); ok {
		t.Fatal("CachedFacts found the model in a cache the provider was not given")
	}
}

// TestCache_ConcurrentUse_0048: 32 listings through one cache at once make
// one request; run under -race, the cache is safe for concurrent use.
func TestCache_ConcurrentUse_0048(t *testing.T) {
	srv, n := listingServer(t)
	var c catalog.Cache
	var wg sync.WaitGroup
	errs := make(chan error, 32)
	for range 32 {
		wg.Go(func() {
			_, err := catalog.List(context.Background(), llmprovider.ProviderTogether, llmprovider.NewStaticToken(key0048(t)),
				listOpts(t, srv, catalog.WithCache(&c))...)
			errs <- err
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("List: %v", err)
		}
	}
	if got := n.Load(); got != 1 {
		t.Errorf("%d listing request(s) for 32 concurrent listings of one cache, want 1", got)
	}
}
