package kilo

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

// Ported from the Kilo rows of llmprovider's discovery_wiring_test.go and
// probe_scope_test.go (0015-PLAN S7). DiscoverModels is ListModels. The kilo
// row of TestDiscoverModels_HonoursRankingOptions is in catalog's
// ranking_options_test.go (0015-PLAN S8, commit 2).

const kiloListing = `{"data":[{"id":"deepseek/deepseek-v4.1-flash","architecture":{"input_modalities":["text"],` +
	`"output_modalities":["text"]},"supported_parameters":["tools"],"pricing":{"completion":"0.1"}}]}`

func list(t *testing.T, p llmprovider.Provider) ([]string, error) {
	t.Helper()
	lister, ok := p.(llmprovider.ModelLister)
	if !ok {
		t.Fatal("the provider is not a ModelLister")
	}
	return lister.ListModels(context.Background())
}

// TestListModels_NeverProbes: listing spends no generation on this metered
// gateway (MADR 0012 §1.6), whatever WithModelProbes says. It was the kilo row
// of TestDiscoverModels_MeteredServicesDoNotProbe.
func TestListModels_NeverProbes(t *testing.T) {
	for _, opts := range [][]llmprovider.Option{nil, {llmprovider.WithModelProbes(true)}} {
		var posts atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				posts.Add(1)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			_, _ = w.Write([]byte(kiloListing))
		}))
		t.Cleanup(srv.Close)
		models, err := list(t, build(t, apiKey(srv.URL, "some/model", opts...)...))
		if err != nil || len(models) == 0 {
			t.Fatalf("ListModels = %v/%v, want the curated listing", models, err)
		}
		if n := posts.Load(); n != 0 {
			t.Fatalf("ListModels made %d generation requests, want 0", n)
		}
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
// 10 s bound. It was the kilo row of TestDiscoverModels_ListingBounded.
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

// TestListModels_CarriesTheCallersIdentity: the listing names the application
// the caller named with WithClientInfo, as generation does (MADR 0012 §1.4).
func TestListModels_CarriesTheCallersIdentity(t *testing.T) {
	var mu sync.Mutex
	var agents []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		agents = append(agents, r.Method+" "+r.URL.Path+" "+r.UserAgent())
		mu.Unlock()
		_, _ = w.Write([]byte(kiloListing))
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

// TestListModels_ListsTheOrganization: kilo.WithOrganization reaches the
// listing through catalog.WithKiloOrganization, which asks for the
// organization's catalog with its header (MADR 0012 §3.3; 0015-PLAN S8,
// commit 2).
func TestListModels_ListsTheOrganization(t *testing.T) {
	var mu sync.Mutex
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.URL.Path+" org="+r.Header.Get("X-KILOCODE-ORGANIZATIONID"))
		mu.Unlock()
		_, _ = w.Write([]byte(kiloListing))
	}))
	t.Cleanup(srv.Close)
	for _, org := range []llmprovider.Option{
		WithOrganization("org-1"),
		// The same, through an overlay in a list other providers share
		// (0015-MADR D5 step 3).
		llmprovider.For(llmprovider.ProviderKilo, WithOrganization("org-1")),
	} {
		mu.Lock()
		seen = nil
		mu.Unlock()
		p := build(t, llmprovider.WithAPIKey("k"), llmprovider.WithModel("m"), llmprovider.WithBaseURL(srv.URL), org)
		if _, err := list(t, p); err != nil {
			t.Fatalf("ListModels: %v", err)
		}
		mu.Lock()
		got := append([]string(nil), seen...)
		mu.Unlock()
		if len(got) != 1 || got[0] != "/api/organizations/org-1/models org=org-1" {
			t.Fatalf("listing requests = %q, want the organization's catalog with its header", got)
		}
	}
}
