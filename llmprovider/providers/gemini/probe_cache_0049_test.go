package gemini

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// TestProbeCache_ProviderProbesIntoItsCache_0049 (0049-MADR D4, F5): a
// provider built WithCache keeps its probes in that cache: a second listing
// through it sends none, and one through another cache, or the process's,
// probes again. The server answers each probe, so its result is kept.
func TestProbeCache_ProviderProbesIntoItsCache_0049(t *testing.T) {
	var posts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			posts.Add(1)
			_, _ = w.Write([]byte(`{"id":"v1_probe","status":"completed","steps":[{"type":"model_output","content":[{"type":"text","text":"Hello"}]}]}`))
			return
		}
		_, _ = w.Write([]byte(geminiListing))
	}))
	t.Cleanup(srv.Close)
	var a, b catalog.Cache
	for i, step := range []struct {
		cache  *catalog.Cache
		probes bool
	}{{&a, true}, {&a, false}, {&b, true}, {nil, true}} {
		before := posts.Load()
		p := build(t, apiKey(srv.URL, "gemini-3.7-flash", llmprovider.WithModelProbes(true), catalog.WithCache(step.cache))...)
		if models, err := list(t, p); err != nil || len(models) == 0 {
			t.Fatalf("step %d: ListModels = %v, %v; want models", i, models, err)
		}
		if n := posts.Load() - before; (n > 0) != step.probes {
			t.Fatalf("step %d: %d probes; want probes %v", i, n, step.probes)
		}
	}
}
