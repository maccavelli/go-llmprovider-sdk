package openai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// 0028-MADR D-A3: a second listing within ten minutes sends no probes; a
// different credential or base URL probes again.

// probeServer lists three models and answers every probe "Hello", counting
// the probes.
func probeServer(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	models := catalog.Static(llmprovider.ProviderOpenAI)[:3]
	var posts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			posts.Add(1)
			_, _ = w.Write([]byte(`{"id":"r1","output":[{"type":"message","content":[{"type":"output_text","text":"Hello"}]}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"` + strings.Join(models, `"},{"id":"`) + `"}]}`))
	}))
	t.Cleanup(srv.Close)
	return srv, &posts
}

// listProbes lists through a provider with key and base URL, and returns
// the models and the probes the listing sent.
func listProbes(t *testing.T, key, url string, posts *atomic.Int32) ([]string, int32) {
	t.Helper()
	before := posts.Load()
	p := build(t, llmprovider.WithAPIKey(key), llmprovider.WithBaseURL(url), llmprovider.WithModel("gpt-4.1-mini"),
		llmprovider.WithModelProbes(true))
	models, err := list(t, p)
	if err != nil || len(models) == 0 {
		t.Fatalf("ListModels = %v, %v; want models", models, err)
	}
	return models, posts.Load() - before
}

// TestListModels_SecondListingSendsNoProbes: the first listing probes each
// listed model; the second, through another provider with the same key and
// base URL, sends none and returns the same models.
func TestListModels_SecondListingSendsNoProbes(t *testing.T) {
	srv, posts := probeServer(t)
	first, n := listProbes(t, "k-second-listing", srv.URL, posts)
	if n != 3 {
		t.Fatalf("first listing: %d probes; want 3", n)
	}
	second, n := listProbes(t, "k-second-listing", srv.URL, posts)
	if n != 0 {
		t.Errorf("second listing: %d probes; want 0", n)
	}
	if strings.Join(first, ",") != strings.Join(second, ",") {
		t.Errorf("second listing = %v; want the first's %v", second, first)
	}
}

// TestProbeCache_KeyedByCredentialAndBaseURL: another key, or another base
// URL, probes again.
func TestProbeCache_KeyedByCredentialAndBaseURL(t *testing.T) {
	srv, posts := probeServer(t)
	other, otherPosts := probeServer(t)
	if _, n := listProbes(t, "k-one", srv.URL, posts); n != 3 {
		t.Fatalf("first listing: %d probes; want 3", n)
	}
	if _, n := listProbes(t, "k-two", srv.URL, posts); n != 3 {
		t.Errorf("another key: %d probes; want 3", n)
	}
	if _, n := listProbes(t, "k-one", other.URL, otherPosts); n != 3 {
		t.Errorf("another base URL: %d probes; want 3", n)
	}
}

// TestProbeCache_ProviderKeyIsFingerprinted: the provider caches under the
// key's fingerprint, never the key: the cache answers a lookup by the
// fingerprint, and not one by the key itself.
func TestProbeCache_ProviderKeyIsFingerprinted(t *testing.T) {
	srv, posts := probeServer(t)
	const key = "k-fingerprint"
	models, _ := listProbes(t, key, srv.URL, posts)
	var probed atomic.Int32
	probe := func(context.Context, string) (string, error) { probed.Add(1); return "hello", nil }
	byFingerprint := transport.ProbeKey{Provider: string(llmprovider.ProviderOpenAI), BaseURL: srv.URL, Credential: transport.Fingerprint(key)}
	if _ = transport.ProbeGenerateHealthCached(context.Background(), byFingerprint, models, catalog.MaxListed, probe); probed.Load() != 0 {
		t.Errorf("by the fingerprint: %d probes; want the cached entry", probed.Load())
	}
	byKey := transport.ProbeKey{Provider: string(llmprovider.ProviderOpenAI), BaseURL: srv.URL, Credential: key}
	if _ = transport.ProbeGenerateHealthCached(context.Background(), byKey, models, catalog.MaxListed, probe); probed.Load() == 0 {
		t.Error("by the key itself: answered from the cache; want no entry under the raw key")
	}
}
