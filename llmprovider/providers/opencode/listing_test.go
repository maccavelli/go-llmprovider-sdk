package opencode

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Ported from the OpenCode rows of llmprovider's discovery_wiring_test.go and
// probe_scope_test.go (0015-PLAN S7). DiscoverModels is ListModels.

func list(t *testing.T, p llmprovider.Provider) ([]string, error) {
	t.Helper()
	lister, ok := p.(llmprovider.ModelLister)
	if !ok {
		t.Fatal("the provider is not a ModelLister")
	}
	return lister.ListModels(context.Background())
}

// TestListModels_NeverProbes: listing spends no generation on these metered
// gateways (MADR 0012 §1.6), whatever WithModelProbes says; it returns the
// curated listing. It was the opencode row of
// TestDiscoverModels_MeteredServicesDoNotProbe.
func TestListModels_NeverProbes(t *testing.T) {
	for _, opts := range [][]llmprovider.Option{nil, {llmprovider.WithModelProbes(true)}} {
		var posts atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				posts.Add(1)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			_, _ = w.Write([]byte(`{"data":[{"id":"glm-5.3-flash"}]}`))
		}))
		t.Cleanup(srv.Close)
		p := build(t, llmprovider.ProviderOpencodeGo, apiKey(srv.URL, "glm-5.3-flash",
			append(opts, llmprovider.WithModelMetadataURL(srv.URL+"/api.json"))...)...)
		models, err := list(t, p)
		if err != nil || len(models) == 0 {
			t.Fatalf("ListModels = %v/%v, want the curated listing", models, err)
		}
		if n := posts.Load(); n != 0 {
			t.Fatalf("ListModels made %d generation requests, want 0", n)
		}
	}
}

// TestListModels_ListingBounded pins MADR 0013 A5: the listing runs under the
// 10 s bound, so a slow listing host cannot hold discovery past it. It was
// the opencode row of TestDiscoverModels_ListingBounded.
func TestListModels_ListingBounded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	rec := &deadlineTransport{}
	p := build(t, llmprovider.ProviderOpencodeZen, apiKey(srv.URL, "glm-5.3-flash", llmprovider.WithHTTPClient(&http.Client{Transport: rec}))...)
	_, _ = list(t, p)
	left, ok := rec.left("GET /models")
	switch {
	case !ok:
		t.Fatal("no GET /models request was made")
	case left < 0:
		t.Error("GET /models ran without a deadline, want one within 10s")
	case left > 10*time.Second:
		t.Errorf("GET /models ran with %v left, want at most 10s", left.Round(time.Second))
	}
}

// TestListModels_HonoursTheMetadataURLOption is the opencode row of
// TestDiscoverModels_HonoursRankingOptions (MADR 0013 A4), less its profile
// half: ListModels returns what the catalog recommends for the same listing and
// metadata, and never reads the environment's metadata URL when the option
// names one. A profile cannot be chosen on the new API until 0015-PLAN S8b
// (0015-MADR, amendment "the OpenCode family").
func TestListModels_HonoursTheMetadataURLOption(t *testing.T) {
	enableMetadata(t)
	envMeta, envHits := metadataServer(t, http.StatusInternalServerError, "")
	t.Setenv("LLMPROVIDER_MODELS_METADATA_URL", envMeta.URL)
	listing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"glm-5.3-flash"},{"id":"qwen3.8-flash"},{"id":"kimi-k2.6"},{"id":"gpt-6-luna"}]}`))
	}))
	t.Cleanup(listing.Close)
	meta, metaHits := metadataServer(t, http.StatusOK, `{"opencode":{"models":{`+
		`"glm-5.3-flash":{"id":"glm-5.3-flash","reasoning":true},"kimi-k2.6":{"id":"kimi-k2.6","reasoning":true}}}}`)
	opts := []llmprovider.Option{llmprovider.WithBaseURL(listing.URL), llmprovider.WithModelMetadataURL(meta.URL)}
	want, err := llmprovider.ListAvailableModels(context.Background(), llmprovider.ProviderOpencodeZen, "k", opts...)
	if err != nil {
		t.Fatalf("ListAvailableModels: %v", err)
	}
	got, err := list(t, build(t, llmprovider.ProviderOpencodeZen, append([]llmprovider.Option{llmprovider.WithAPIKey("k"),
		llmprovider.WithModel("glm-5.3-flash")}, opts...)...))
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("ListModels = %v, want the catalog's %v", got, want)
	}
	if n := envHits.Load(); n != 0 {
		t.Errorf("the environment's metadata URL was fetched %d times, want 0", n)
	}
	if metaHits.Load() == 0 {
		t.Error("the option's metadata URL was never fetched")
	}
}

// TestListModels_CarriesTheCallersIdentity: the listing names the application
// the caller named with WithClientInfo, as generation does (MADR 0012 §1.4).
func TestListModels_CarriesTheCallersIdentity(t *testing.T) {
	var mu sync.Mutex
	var agents []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		agents = append(agents, r.Method+" "+r.URL.Path+" "+r.UserAgent())
		mu.Unlock()
		_, _ = w.Write([]byte(`{"data":[{"id":"glm-5.3-flash"}]}`))
	}))
	t.Cleanup(srv.Close)
	p := build(t, llmprovider.ProviderOpencodeGo, apiKey(srv.URL, "m", llmprovider.WithClientInfo("wire-app", "9.9.9"))...)
	if _, err := list(t, p); err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(agents) != 1 || !strings.HasPrefix(agents[0], "GET /models wire-app/9.9.9 (") {
		t.Fatalf("listing requests %q; want one GET /models naming wire-app/9.9.9", agents)
	}
}
