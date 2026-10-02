package together

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// The listing through ListModels (0015-PLAN S7): DiscoverModels is
// ListModels.

// TestListModels_NeverProbes: listing spends no generation on this metered
// service (0016-MADR D9), whatever WithModelProbes says.
func TestListModels_NeverProbes(t *testing.T) {
	for _, opts := range [][]llmprovider.Option{nil, {llmprovider.WithModelProbes(true)}} {
		var posts atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				posts.Add(1)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			_, _ = w.Write([]byte(`[{"id":"openai/gpt-oss-120b","type":"chat"}]`))
		}))
		t.Cleanup(srv.Close)
		models, err := list(t, build(t, apiKey(srv.URL, "m", opts...)...))
		if err != nil || len(models) == 0 {
			t.Fatalf("ListModels = %v/%v, want the curated listing", models, err)
		}
		if n := posts.Load(); n != 0 {
			t.Fatalf("ListModels made %d generation requests, want 0", n)
		}
	}
}

// failingSource fails every Token call.
type failingSource struct{}

func (failingSource) Token(context.Context) (llmprovider.Token, error) {
	return llmprovider.Token{}, errors.New("no key today")
}

// TestListModels_FallsBackWhenTheKeyFails: the shared listing already falls
// back for a bad body; a token failure, which it returns, falls back to the
// static catalog too, as DiscoverModels never failed.
func TestListModels_FallsBackWhenTheKeyFails(t *testing.T) {
	got, err := list(t, build(t, llmprovider.WithTokenSource(failingSource{}), llmprovider.WithModel("m")))
	if want := catalog.Static(llmprovider.ProviderTogether); err != nil || strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("ListModels = %v, %v; want the static catalog %v", got, err, want)
	}
}

// deadlineTransport records how long the first listing request's context had
// left (-1 without a deadline).
type deadlineTransport struct {
	mu   sync.Mutex
	left map[string]time.Duration
}

func (d *deadlineTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	left := time.Duration(-1)
	if dl, ok := r.Context().Deadline(); ok {
		left = time.Until(dl)
	}
	key := r.Method + " " + r.URL.Path
	d.mu.Lock()
	if d.left == nil {
		d.left = map[string]time.Duration{}
	}
	if _, ok := d.left[key]; !ok {
		d.left[key] = left
	}
	d.mu.Unlock()
	return http.DefaultTransport.RoundTrip(r)
}

// TestListModels_ListingBounded pins MADR 0013 A5: the listing runs under the
// 10 s bound.
func TestListModels_ListingBounded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	rec := &deadlineTransport{}
	_, _ = list(t, build(t, apiKey(srv.URL, "m", llmprovider.WithHTTPClient(&http.Client{Transport: rec}))...))
	rec.mu.Lock()
	defer rec.mu.Unlock()
	left, ok := rec.left["GET /models"]
	switch {
	case !ok:
		t.Fatal("no GET /models request was made")
	case left < 0:
		t.Error("GET /models ran without a deadline, want one within 10s")
	case left > 10*time.Second:
		t.Errorf("GET /models ran with %v left, want at most 10s", left.Round(time.Second))
	}
}

// TestListModels_HonoursTheMetadataURLOption (MADR 0013 A4), less a profile:
// ListModels returns what the catalog recommends for the same listing and
// metadata. The environment's URL is never read: internal/ambientcheck
// holds that (0015-PLAN S10). Its profile half is in catalog's ranking_options_test.go (0015-PLAN S8,
// commit 2).
func TestListModels_HonoursTheMetadataURLOption(t *testing.T) {
	enableMetadata(t)
	listing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`[{"id":"a/large","type":"chat"},{"id":"c/flash","type":"chat"}]`))
	}))
	t.Cleanup(listing.Close)
	meta, metaHits := metadataServer(t, http.StatusOK, `{"togetherai":{"models":{`+
		`"a/large":{"id":"a/large","reasoning":true},"c/flash":{"id":"c/flash","reasoning":true,"tool_call":true}}}}`)
	opts := []llmprovider.Option{llmprovider.WithBaseURL(listing.URL), llmprovider.WithModelMetadataURL(meta.URL)}
	wantCat, err := catalog.List(context.Background(), llmprovider.ProviderTogether, llmprovider.NewStaticToken("k"), opts...)
	want := wantCat.Recommended
	if err != nil {
		t.Fatalf("ListAvailableModels: %v", err)
	}
	got, err := list(t, build(t, append([]llmprovider.Option{llmprovider.WithAPIKey("k"), llmprovider.WithModel("m")}, opts...)...))
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("ListModels = %v, want the catalog's %v", got, want)
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
		_, _ = w.Write([]byte(`[{"id":"openai/gpt-oss-120b","type":"chat"}]`))
	}))
	t.Cleanup(srv.Close)
	if _, err := list(t, build(t, apiKey(srv.URL, "m", llmprovider.WithClientInfo("wire-app", "9.9.9"))...)); err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(agents) != 1 || !strings.HasPrefix(agents[0], "GET /models wire-app/9.9.9 (") {
		t.Fatalf("listing requests %q; want one GET /models naming wire-app/9.9.9", agents)
	}
}
