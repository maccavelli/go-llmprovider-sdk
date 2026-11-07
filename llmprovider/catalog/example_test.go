package catalog_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// ExampleStatic reads a provider's built-in model list, which needs no
// network and no credential.
func ExampleStatic() {
	models := catalog.Static(llmprovider.ProviderClaude)
	fmt.Println(len(models) > 0)
	// Output: true
}

// ExampleSortByRank orders a Grok list. Beside grok-4.3 and grok-4.7,
// grok-4.20 reads as 4.2, so it sorts last; no dates are needed.
func ExampleSortByRank() {
	ids := []string{"grok-4.20-0309-reasoning", "grok-4.7", "grok-4.3"}
	fmt.Println(catalog.SortByRank(llmprovider.ProviderGrok, ids, nil))
	// Output: [grok-4.7 grok-4.3 grok-4.20-0309-reasoning]
}

// ExampleLabelName reads a curated label's two parts, for a picker that shows
// the name and the note apart.
func ExampleLabelName() {
	fmt.Println(catalog.LabelName(llmprovider.ProviderGrok, "grok-4.7"))
	fmt.Println(catalog.LabelNote(llmprovider.ProviderGrok, "grok-4.7"))
	// Output:
	// Grok 4.7
	// frontier model
}

// ExampleWithCache keeps two caches' listings apart: a listing is fetched
// once per cache. The httptest server stands in for Together's listing, so
// that the example runs offline.
func ExampleWithCache() {
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id": "org/model-a", "type": "chat", "context_length": 131072}]`))
	}))
	defer srv.Close()

	var a, b catalog.Cache
	list := func(c *catalog.Cache) {
		_, err := catalog.List(context.Background(), llmprovider.ProviderTogether, llmprovider.NewStaticToken("your-key"),
			llmprovider.WithBaseURL(srv.URL+"/v1"), llmprovider.WithoutModelMetadata(), catalog.WithCache(c))
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println("listing requests:", requests.Load())
	}
	list(&a)
	list(&a) // answered from a
	list(&b)
	// Output:
	// listing requests: 1
	// listing requests: 1
	// listing requests: 2
}

// ExampleCache_Close ends a cache's life. CachedModel starts the listing's
// fetch in the background, and Close waits for it. After Close, the cache
// starts nothing, and List refuses. The httptest server stands in for
// Together's listing, so that the example runs offline.
func ExampleCache_Close() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id": "org/model-a", "type": "chat", "context_length": 131072}]`))
	}))
	defer srv.Close()

	var cache catalog.Cache
	key := llmprovider.NewStaticToken("your-key")
	opts := []llmprovider.Option{llmprovider.WithBaseURL(srv.URL + "/v1"), llmprovider.WithoutModelMetadata(),
		catalog.WithCache(&cache)}
	catalog.CachedModel(context.Background(), llmprovider.ProviderTogether, key, "org/model-a", opts...)
	fmt.Println(cache.Close(context.Background()))
	_, err := catalog.List(context.Background(), llmprovider.ProviderTogether, key, opts...)
	fmt.Println(errors.Is(err, llmprovider.ErrInvalidRequest))
	// Output:
	// <nil>
	// true
}
