package grok

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// Ported from llmprovider's probe_test.go, probe_scope_test.go and
// transport_defaults_test.go (0015-PLAN S7). DiscoverModels is ListModels.

func list(t *testing.T, p llmprovider.Provider) ([]string, error) {
	t.Helper()
	lister, ok := p.(llmprovider.ModelLister)
	if !ok {
		t.Fatal("the provider is not a ModelLister")
	}
	return lister.ListModels(context.Background())
}

// TestListModels was TestGrokProvider_DiscoverModels.
func TestListModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/models":
			_, _ = w.Write([]byte(`{"data":[{"id":"grok-3-mini-fast"}]}`))
		case "/responses":
			_, _ = w.Write([]byte(`{"id":"r1","output":[{"type":"message","content":[{"type":"output_text","text":"Hello"}]}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	models, err := list(t, build(t, apiKey(srv.URL, "grok-3-mini-fast")...))
	if err != nil {
		t.Fatalf("ListModels error: %v", err)
	}
	if len(models) == 0 {
		t.Fatal("expected discovered models")
	}
}

// generationCounter answers every listing (GET) with a minimal catalog and
// counts generation requests (POST), which it fails.
func generationCounter(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var posts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			posts.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"grok-4.5"}]}`))
	}))
	t.Cleanup(srv.Close)
	return srv, &posts
}

// TestListModels_ProbesFollowDefaultOptionAndEnv (0016-MADR A5): probes by
// default; WithModelProbes turns them off or on; LLMPROVIDER_PROBES takes
// effect only through ModelProbesFromEnv, and an explicit option after it
// wins. It was the grok row of TestDiscoverModels_ProbesFollowDefaultOptionAndEnv.
func TestListModels_ProbesFollowDefaultOptionAndEnv(t *testing.T) {
	for _, tc := range []struct {
		name  string
		env   string // "" leaves LLMPROVIDER_PROBES unset
		opts  func() []llmprovider.Option
		probe bool
	}{
		{"default", "", func() []llmprovider.Option { return nil }, true},
		{"option off", "", func() []llmprovider.Option { return []llmprovider.Option{llmprovider.WithModelProbes(false)} }, false},
		{"option on", "", func() []llmprovider.Option { return []llmprovider.Option{llmprovider.WithModelProbes(true)} }, true},
		{"env false without the helper", "false", func() []llmprovider.Option { return nil }, true},
		{"env false through the helper", "false", func() []llmprovider.Option {
			return []llmprovider.Option{llmprovider.ModelProbesFromEnv()}
		}, false},
		{"env true through the helper", "true", func() []llmprovider.Option {
			return []llmprovider.Option{llmprovider.ModelProbesFromEnv()}
		}, true},
		{"env not a boolean", "sometimes", func() []llmprovider.Option {
			return []llmprovider.Option{llmprovider.ModelProbesFromEnv()}
		}, true},
		{"explicit option after the helper wins", "false", func() []llmprovider.Option {
			return []llmprovider.Option{llmprovider.ModelProbesFromEnv(), llmprovider.WithModelProbes(true)}
		}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("LLMPROVIDER_PROBES", tc.env)
			if tc.env == "" {
				if err := os.Unsetenv("LLMPROVIDER_PROBES"); err != nil {
					t.Fatal(err)
				}
			}
			srv, posts := generationCounter(t)
			if _, err := list(t, build(t, apiKey(srv.URL, "grok-4.5", tc.opts()...)...)); err != nil {
				t.Fatalf("ListModels: %v", err)
			}
			switch n := posts.Load(); {
			case !tc.probe && n != 0:
				t.Errorf("%d generation requests, want none", n)
			case tc.probe && (n == 0 || n > catalog.MaxListed):
				t.Errorf("%d generation requests, want one per candidate", n)
			}
		})
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
// 10 s bound ListModelCatalogWithSource applies. Grok's old DiscoverModels
// listed through the same function; no test pinned it there.
func TestListModels_ListingBounded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	rec := &deadlineTransport{}
	p := build(t, apiKey(srv.URL, "m", llmprovider.WithHTTPClient(&http.Client{Transport: rec}), llmprovider.WithModelProbes(false))...)
	_, _ = list(t, p)
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
		agents = append(agents, r.Method+" "+r.UserAgent())
		mu.Unlock()
		_, _ = w.Write([]byte(`{"data":[{"id":"grok-4.5"}]}`))
	}))
	t.Cleanup(srv.Close)
	p := build(t, apiKey(srv.URL, "m", llmprovider.WithModelProbes(false), llmprovider.WithClientInfo("wire-app", "9.9.9"))...)
	if _, err := list(t, p); err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(agents) != 1 || !strings.HasPrefix(agents[0], "GET wire-app/9.9.9 (") {
		t.Fatalf("listing requests %q; want one GET naming wire-app/9.9.9", agents)
	}
}

// TestListModels_ProbeBodyIsTheOldProbes: a probe sends what the old API's
// probe provider sent: the default output limit, no store and no reasoning,
// whatever the provider's own options say (0015-MADR D1).
func TestListModels_ProbeBodyIsTheOldProbes(t *testing.T) {
	var mu sync.Mutex
	var probes []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"data":[{"id":"grok-4.5"}]}`))
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		probes = append(probes, body)
		mu.Unlock()
		_, _ = w.Write([]byte(`{"id":"r","output":[{"type":"message","content":[{"type":"output_text","text":"Hello"}]}]}`))
	}))
	t.Cleanup(srv.Close)
	p := build(t, apiKey(srv.URL, "grok-base", llmprovider.WithMaxTokens(77), WithStore(false),
		llmprovider.WithReasoning(&llmprovider.Reasoning{Effort: llmprovider.EffortHigh}))...)
	if _, err := list(t, p); err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(probes) != 1 {
		t.Fatalf("%d probes, want 1", len(probes))
	}
	_, store := probes[0]["store"]
	_, reasoning := probes[0]["reasoning"]
	if probes[0]["model"] != "grok-4.5" || probes[0]["max_output_tokens"] != float64(probeMaxOutputTokens) || store || reasoning {
		t.Fatalf("probe body %v; want the listed model, max_output_tokens %d, no store, no reasoning", probes[0], probeMaxOutputTokens)
	}
}

// pathRecorder records every request's method and path.
type pathRecorder struct {
	base  http.RoundTripper
	mu    sync.Mutex
	paths []string
}

func (r *pathRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
	r.mu.Lock()
	r.paths = append(r.paths, req.Method+" "+req.URL.Path)
	r.mu.Unlock()
	return r.base.RoundTrip(req)
}

// TestProviderClient_SharedWithListingAndRefresh (0016-PLAN T1 step 3): a
// provider built without WithHTTPClient sends its generation, its listing and
// its OAuth session's refresh through one *http.Client. It was the grok row
// of llmprovider's test of the same name.
func TestProviderClient_SharedWithListingAndRefresh(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/oauth/token":
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "a-refreshed", "expires_in": 3600})
		case r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"data":[{"id":"grok-4.5"}]}`))
		default:
			_, _ = w.Write([]byte(okResponse))
		}
	}))
	t.Cleanup(srv.Close)
	session := &auth.OAuthSession{Provider: llmprovider.ProviderGrok, Access: "a-old", Refresh: "rt-old",
		Expiry: time.Now().Add(-time.Minute), ClientID: "client-test", TokenURL: srv.URL + "/oauth/token"}
	p := build(t, llmprovider.WithTokenSource(session), llmprovider.WithModel("grok-4.5"), llmprovider.WithBaseURL(srv.URL))
	client := p.(*provider).client
	recorder := &pathRecorder{base: client.Transport}
	client.Transport = recorder

	if _, err := p.Generate(context.Background(), text("hi")); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if _, err := list(t, p); err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	for _, want := range []string{"POST /oauth/token", "POST /responses", "GET /models"} {
		if !slices.Contains(recorder.paths, want) {
			t.Errorf("%s did not go through the provider's client (it carried %v)", want, recorder.paths)
		}
	}
}
