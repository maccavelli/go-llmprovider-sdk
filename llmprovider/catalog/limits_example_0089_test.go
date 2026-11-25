package catalog_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// ExampleLookupLimits reads a Claude model's limits from its listing, here
// an httptest server, before a session is set up with them.
func ExampleLookupLimits() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"id":"claude-opus-4-6","max_tokens":128000,"max_input_tokens":1000000}],"has_more":false}`))
	}))
	defer srv.Close()
	cache := new(catalog.Cache)
	defer func() { _ = cache.Close(context.Background()) }()

	limits, err := catalog.LookupLimits(context.Background(), llmprovider.ProviderClaude, llmprovider.NewStaticToken("key"),
		"claude-opus-4-6", llmprovider.WithBaseURL(srv.URL), catalog.WithCache(cache))
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(limits.Output.Tokens, limits.Output.Evidence[0].Source, limits.Required)
	// Output: 128000 listing:claude true
}
