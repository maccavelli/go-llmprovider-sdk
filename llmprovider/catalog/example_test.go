package catalog_test

import (
	"context"
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
