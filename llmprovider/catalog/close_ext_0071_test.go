package catalog_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
)

// TestCacheClose_ProviderListModels_0071: a provider given a closed cache
// takes its own path for a failed listing, and sends nothing. Together's
// ListModels returns its static list. Ollama's returns the closed-cache
// refusal (0071-MADR, amendment "a provider's ListModels takes its own
// failure path"; 0071-PLAN Deviation 3).
func TestCacheClose_ProviderListModels_0071(t *testing.T) {
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		http.NotFound(w, nil)
	}))
	t.Cleanup(srv.Close)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var c catalog.Cache
	if err := c.Close(ctx); err != nil {
		t.Fatal(err)
	}
	together, err := providers.New(llmprovider.ProviderTogether, llmprovider.WithAPIKey("key-0071-together"),
		llmprovider.WithBaseURL(srv.URL+"/v1"), llmprovider.WithoutModelMetadata(), catalog.WithCache(&c))
	if err != nil {
		t.Fatal(err)
	}
	listed, err := together.(llmprovider.ModelLister).ListModels(ctx)
	if err != nil || !slices.Equal(listed, catalog.Static(llmprovider.ProviderTogether)) {
		t.Errorf("Together's ListModels after Close = %v, %v; want its static list and nil", listed, err)
	}
	ollama, err := providers.New(llmprovider.ProviderOllama, llmprovider.WithBaseURL(srv.URL),
		llmprovider.WithoutModelMetadata(), catalog.WithCache(&c))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ollama.(llmprovider.ModelLister).ListModels(ctx); !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Errorf("Ollama's ListModels after Close = %v; want the closed-cache refusal", err)
	}
	if n := requests.Load(); n != 0 {
		t.Errorf("%d request(s) after Close; want 0", n)
	}
}
