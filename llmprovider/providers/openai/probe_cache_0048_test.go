package openai

import (
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// TestProbeCache_ProviderProbesIntoItsCache_0048 (0048-MADR D1): a provider
// built WithCache keeps its probes in that cache: a second listing through it
// sends none, and one through another cache, or the process's, probes again.
func TestProbeCache_ProviderProbesIntoItsCache_0048(t *testing.T) {
	srv, posts := probeServer(t)
	var a, b catalog.Cache
	for i, step := range []struct {
		cache *catalog.Cache
		want  int32
	}{{&a, 3}, {&a, 0}, {&b, 3}, {nil, 3}} {
		before := posts.Load()
		p := build(t, llmprovider.WithAPIKey("k-0048"), llmprovider.WithBaseURL(srv.URL), llmprovider.WithModel("gpt-4.1-mini"),
			llmprovider.WithModelProbes(true), catalog.WithCache(step.cache))
		if models, err := list(t, p); err != nil || len(models) == 0 {
			t.Fatalf("step %d: ListModels = %v, %v; want models", i, models, err)
		}
		if n := posts.Load() - before; n != step.want {
			t.Fatalf("step %d: %d probes; want %d", i, n, step.want)
		}
	}
}
