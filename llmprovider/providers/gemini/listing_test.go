package gemini

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// Ported from llmprovider's probe_test.go, probe_scope_test.go and
// discovery_wiring_test.go (0015-PLAN S7). DiscoverModels is ListModels.

const geminiListing = `{"models":[{"name":"models/gemini-3.7-flash","supportedGenerationMethods":["generateContent"]}]}`

func list(t *testing.T, p llmprovider.Provider) ([]string, error) {
	t.Helper()
	lister, ok := p.(llmprovider.ModelLister)
	if !ok {
		t.Fatal("the provider is not a ModelLister")
	}
	return lister.ListModels(context.Background())
}

// TestListModels was TestGeminiProvider_DiscoverModels.
func TestListModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/models":
			_, _ = w.Write([]byte(geminiListing))
		default:
			_, _ = w.Write([]byte(`{"id":"v1_p","status":"completed","steps":[{"type":"model_output","content":[{"type":"text","text":"Hello"}]}]}`))
		}
	}))
	defer srv.Close()
	models, err := list(t, build(t, apiKey(srv.URL, "gemini-3.7-flash")...))
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
		_, _ = w.Write([]byte(geminiListing))
	}))
	t.Cleanup(srv.Close)
	return srv, &posts
}

// TestListModels_ProbesFollowDefaultOptionAndEnv (0016-MADR A5): probes by
// default; WithModelProbes turns them off or on; LLMPROVIDER_PROBES takes
// effect only through ModelProbesFromEnv, and an explicit option after it wins.
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
			p := build(t, apiKey(srv.URL, "gemini-3.7-flash", tc.opts()...)...)
			if _, err := list(t, p); err != nil {
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

// TestListModels_ListingBounded pins MADR 0013 A5: the listing runs under
// the 10 s bound, so a slow listing host cannot hold discovery past it. It
// was the gemini row of TestDiscoverModels_ListingBounded.
func TestListModels_ListingBounded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	rec := &deadlineTransport{}
	p := build(t, apiKey(srv.URL, "m", llmprovider.WithHTTPClient(&http.Client{Transport: rec}))...)
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
		_, _ = w.Write([]byte(geminiListing))
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
// probe provider sent: the provider's output limit, store false and no
// thinking, whatever WithStore and WithReasoning say (0015-MADR D1).
func TestListModels_ProbeBodyIsTheOldProbes(t *testing.T) {
	var mu sync.Mutex
	var probes []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(geminiListing))
			return
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		probes = append(probes, body)
		mu.Unlock()
		_, _ = w.Write([]byte(`{"id":"v1_p","status":"completed","steps":[{"type":"model_output","content":[{"type":"text","text":"Hello"}]}]}`))
	}))
	t.Cleanup(srv.Close)
	p := build(t, apiKey(srv.URL, "gemini-base", llmprovider.WithMaxTokens(77), WithStore(true),
		llmprovider.WithReasoning(&llmprovider.Reasoning{Effort: llmprovider.EffortHigh}))...)
	if _, err := list(t, p); err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(probes) != 1 {
		t.Fatalf("%d probes, want 1", len(probes))
	}
	gen := generationConfig(probes[0])
	_, summaries := gen["thinking_summaries"]
	if probes[0]["model"] != "gemini-3.7-flash" || gen["max_output_tokens"] != float64(77) || probes[0]["store"] != false || summaries {
		t.Fatalf("probe body %v; want the listed model, max_output_tokens 77, store false, no thinking", probes[0])
	}
}
