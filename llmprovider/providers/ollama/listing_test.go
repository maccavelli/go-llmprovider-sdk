package ollama

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Ported from the Ollama row of llmprovider's discovery_wiring_test.go
// (0015-PLAN S7). DiscoverModels is ListModels.

// instance answers GET /api/tags with tags, and counts the generation
// requests (POST), answering each with fxOllamaChat or failing it.
func instance(t *testing.T, tags string, fail bool) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var posts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/tags":
			_, _ = w.Write([]byte(tags))
		case r.Method == http.MethodPost:
			posts.Add(1)
			if fail {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"Hello"}}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &posts
}

// TestListModels_ProbesFollowTheOption (0016-MADR A5): every installed model
// is probed by default, and none with WithModelProbes(false).
func TestListModels_ProbesFollowTheOption(t *testing.T) {
	for _, tc := range []struct {
		name  string
		opts  []llmprovider.Option
		posts int32
	}{
		{"default", nil, 2},
		{"option off", []llmprovider.Option{llmprovider.WithModelProbes(false)}, 0},
		{"option on", []llmprovider.Option{llmprovider.WithModelProbes(true)}, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv, posts := instance(t, ollamaTags("llama3.3:latest", "qwen3:8b"), false)
			models, err := list(t, build(t, local(srv.URL, "m", tc.opts...)...))
			if err != nil || strings.Join(models, ",") != "llama3.3:latest,qwen3:8b" {
				t.Fatalf("ListModels = %v/%v, want the installed models", models, err)
			}
			if n := posts.Load(); n != tc.posts {
				t.Errorf("%d generation requests, want %d", n, tc.posts)
			}
		})
	}
}

// TestListModels_KeepsTheListingWhenNoProbeAnswers: a probe failure is not a
// listing failure.
func TestListModels_KeepsTheListingWhenNoProbeAnswers(t *testing.T) {
	srv, _ := instance(t, ollamaTags("llama3.3:latest"), true)
	models, err := list(t, build(t, local(srv.URL, "m")...))
	if err != nil || strings.Join(models, ",") != "llama3.3:latest" {
		t.Fatalf("ListModels = %v/%v, want the listing", models, err)
	}
}

// TestListModels_ProbeIsTheOldRequest: a probe sends the old probe provider's
// request, whatever the caller's output limit and reasoning.
func TestListModels_ProbeIsTheOldRequest(t *testing.T) {
	var mu sync.Mutex
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(ollamaTags("llama3.3:latest")))
			return
		}
		raw, _ := io.ReadAll(r.Body)
		mu.Lock()
		bodies = append(bodies, string(raw))
		mu.Unlock()
		_, _ = w.Write([]byte(fxOllamaChat))
	}))
	t.Cleanup(srv.Close)
	p := build(t, local(srv.URL, "m", llmprovider.WithMaxTokens(500),
		llmprovider.WithReasoning(&llmprovider.Reasoning{Effort: llmprovider.EffortHigh}))...)
	if _, err := list(t, p); err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	want := `{"max_tokens":8192,"messages":[{"content":"Respond with ONLY the word Hello","role":"user"}],"model":"llama3.3:latest"}`
	if len(bodies) != 1 || bodies[0] != want {
		t.Errorf("probes sent %q, want one %s", bodies, want)
	}
}

// TestListModels_ReturnsTheListingError: there is no static catalog, so an
// unreachable listing is an error and an empty install an empty listing.
func TestListModels_ReturnsTheListingError(t *testing.T) {
	notFound := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(notFound.Close)
	if models, err := list(t, build(t, local(notFound.URL, "m")...)); err == nil || models != nil {
		t.Errorf("404: ListModels = %v/%v, want an error", models, err)
	}
	empty, posts := instance(t, ollamaTags(), false)
	if models, err := list(t, build(t, local(empty.URL, "m")...)); err != nil || len(models) != 0 || posts.Load() != 0 {
		t.Errorf("empty: ListModels = %v/%v with %d probes, want nothing and no error", models, err, posts.Load())
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
// 10 s bound. It was the ollama row of TestDiscoverModels_ListingBounded.
func TestListModels_ListingBounded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	rec := &deadlineTransport{}
	_, _ = list(t, build(t, local(srv.URL, "m", llmprovider.WithHTTPClient(&http.Client{Transport: rec}))...))
	rec.mu.Lock()
	defer rec.mu.Unlock()
	left, ok := rec.left["GET /api/tags"]
	switch {
	case !ok:
		t.Fatal("no GET /api/tags request was made")
	case left < 0:
		t.Error("GET /api/tags ran without a deadline, want one within 10s")
	case left > 10*time.Second:
		t.Errorf("GET /api/tags ran with %v left, want at most 10s", left.Round(time.Second))
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
		_, _ = w.Write([]byte(ollamaTags("llama3.3:latest")))
	}))
	t.Cleanup(srv.Close)
	if _, err := list(t, build(t, local(srv.URL, "m", llmprovider.WithClientInfo("wire-app", "9.9.9"),
		llmprovider.WithModelProbes(false))...)); err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(agents) != 1 || !strings.HasPrefix(agents[0], "GET /api/tags wire-app/9.9.9 (") {
		t.Fatalf("listing requests %q; want one GET /api/tags naming wire-app/9.9.9", agents)
	}
}
