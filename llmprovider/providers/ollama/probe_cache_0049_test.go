package ollama

import (
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// TestProbeCache_ProviderProbesIntoItsCache_0049 (0049-MADR D4, F5): a
// provider built WithCache keeps its probes in that cache: a second listing
// through it sends none, and one through another cache, or the process's,
// probes again. The instance answers each probe, so its result is kept.
func TestProbeCache_ProviderProbesIntoItsCache_0049(t *testing.T) {
	srv, posts := instance(t, ollamaTags("llama3.3:latest", "qwen3:8b"), false)
	var a, b catalog.Cache
	for i, step := range []struct {
		cache  *catalog.Cache
		probes bool
	}{{&a, true}, {&a, false}, {&b, true}, {nil, true}} {
		before := posts.Load()
		p := build(t, local(srv.URL, "m", llmprovider.WithModelProbes(true), catalog.WithCache(step.cache))...)
		if models, err := list(t, p); err != nil || len(models) == 0 {
			t.Fatalf("step %d: ListModels = %v, %v; want models", i, models, err)
		}
		if n := posts.Load() - before; (n > 0) != step.probes {
			t.Fatalf("step %d: %d probes; want probes %v", i, n, step.probes)
		}
	}
}
