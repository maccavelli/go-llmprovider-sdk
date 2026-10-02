package openai

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
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
)

// Ported from llmprovider's probe_test.go, probe_scope_test.go and
// transport_defaults_test.go (0015-PLAN S7). DiscoverModels is ListModels
// (0015-MADR amendment of 2026-09-30, D3).

func list(t *testing.T, p llmprovider.Provider) ([]string, error) {
	t.Helper()
	lister, ok := p.(llmprovider.ModelLister)
	if !ok {
		t.Fatal("the provider is not a ModelLister")
	}
	return lister.ListModels(context.Background())
}

func TestListModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/models":
			_, _ = w.Write([]byte(`{"data":[{"id":"gpt-4.1-mini"}]}`))
		case "/responses":
			_, _ = w.Write([]byte(`{"id":"r1","output":[{"type":"message","content":[{"type":"output_text","text":"Hello"}]}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	p := build(t, apiKey(srv.URL, llmprovider.WithModel("gpt-4.1-mini"))...)
	models, err := list(t, p)
	if err != nil {
		t.Fatalf("ListModels error: %v", err)
	}
	if len(models) == 0 {
		t.Fatal("expected discovered models")
	}
}

func TestListModels_ChatGPTListingFailureIsError(t *testing.T) {
	var hosts []string
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		hosts = append(hosts, r.URL.Host+r.URL.Path)
		return httpResponse(r, http.StatusBadGateway, ""), nil
	})}
	session := &auth.OAuthSession{Issuer: auth.DefaultOpenAIIssuer, Access: "session-access",
		Expiry: time.Now().Add(time.Hour)}
	p := sessionProvider(t, session, "chatgpt-model", llmprovider.WithHTTPClient(client))
	models, err := list(t, p)
	if err == nil || models != nil {
		t.Fatalf("ListModels() = %v/%v, want nil/listing error", models, err)
	}
	if len(hosts) != 1 || strings.Contains(hosts[0], "api.openai.com") || !strings.HasSuffix(hosts[0], "/models") {
		t.Fatalf("requests = %v, want one Codex /models listing", hosts)
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
		if r.URL.Query().Get("client_version") != "" {
			_, _ = w.Write([]byte(`{"models":[{"slug":"gpt-6-astra","visibility":"list"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"gpt-4.1-mini"}]}`))
	}))
	t.Cleanup(srv.Close)
	return srv, &posts
}

// TestListModels_ChatGPTDoesNotProbe: a ChatGPT session's listing spends no
// generation (MADR 0012 §1.6); it returns the curated listing.
func TestListModels_ChatGPTDoesNotProbe(t *testing.T) {
	srv, posts := generationCounter(t)
	session := &auth.OAuthSession{Issuer: auth.DefaultOpenAIIssuer, Access: "a", Expiry: time.Now().Add(time.Hour)}
	p := sessionProvider(t, session, "gpt-6-astra", llmprovider.WithBaseURL(srv.URL))
	models, err := list(t, p)
	if err != nil || len(models) == 0 {
		t.Fatalf("ListModels = %v/%v, want the curated listing", models, err)
	}
	if n := posts.Load(); n != 0 {
		t.Fatalf("ListModels made %d generation requests, want 0", n)
	}
}

// TestListModels_ProbesFollowDefaultOptionAndEnv (0016-MADR A5): an API key
// probes by default; WithModelProbes turns it off or on; LLMPROVIDER_PROBES
// takes effect only through ModelProbesFromEnv, and an explicit option passed
// after it wins.
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
			p := build(t, apiKey(srv.URL, append(tc.opts(), llmprovider.WithModel("gpt-4.1-mini"))...)...)
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
// ChatGPT provider built without WithHTTPClient sends its generation, its
// listing and its OAuth session's refresh through one *http.Client. It was
// the chatgpt row of llmprovider's test of that name.
func TestProviderClient_SharedWithListingAndRefresh(t *testing.T) {
	wire := wirecase.Case{Name: "chatgpt", SSE: true,
		Listing: `{"models":[{"slug":"gpt-6-astra","visibility":"list","priority":1,"supported_in_api":true}]}`}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "a-refreshed", "refresh_token": "rt-new", "expires_in": 3600})
			return
		}
		reply := wire.Reply(r)
		w.WriteHeader(max(reply.Status, http.StatusOK))
		_, _ = w.Write([]byte(reply.Body))
	}))
	t.Cleanup(srv.Close)
	session := &auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Issuer: auth.DefaultOpenAIIssuer,
		Access: "a-old", Refresh: "rt-old", Expiry: time.Now().Add(-time.Minute), ClientID: "client-test",
		TokenURL: srv.URL + "/oauth/token", AccountID: "acct-test"}
	p := sessionProvider(t, session, "gpt-6-astra", llmprovider.WithBaseURL(srv.URL))
	client := p.(*provider).client
	recorder := &pathRecorder{base: client.Transport}
	client.Transport = recorder

	if _, err := llmprovider.GenerateText(context.Background(), p, text(wirecase.Prompt)); err != nil {
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

// TestListModels_CarriesTheCallersIdentity: the listing names the application
// the caller named with WithClientInfo, as generation does (MADR 0012 §1.4).
func TestListModels_CarriesTheCallersIdentity(t *testing.T) {
	var mu sync.Mutex
	var agents []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		agents = append(agents, r.Method+" "+r.UserAgent())
		mu.Unlock()
		_, _ = w.Write([]byte(`{"data":[{"id":"gpt-4.1-mini"}]}`))
	}))
	t.Cleanup(srv.Close)
	p := build(t, apiKey(srv.URL, llmprovider.WithModel("gpt-base"), llmprovider.WithModelProbes(false),
		llmprovider.WithClientInfo("wire-app", "9.9.9"))...)
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
// probe provider sent, which carried only the identity, client and base URL:
// the default output limit, no store and no reasoning, whatever the
// provider's own options say (0015-MADR D1).
func TestListModels_ProbeBodyIsTheOldProbes(t *testing.T) {
	var mu sync.Mutex
	var probes []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"data":[{"id":"gpt-4.1-mini"}]}`))
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
	p := build(t, apiKey(srv.URL, llmprovider.WithModel("gpt-base"), llmprovider.WithMaxTokens(77), WithStore(false),
		llmprovider.WithReasoning(&llmprovider.Reasoning{Effort: llmprovider.EffortHigh}))...)
	if _, err := list(t, p); err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(probes) != 1 {
		t.Fatalf("%d probes, want 1", len(probes))
	}
	body := probes[0]
	_, store := body["store"]
	_, reasoning := body["reasoning"]
	if body["model"] != "gpt-4.1-mini" || body["max_output_tokens"] != float64(probeMaxOutputTokens) || store || reasoning {
		t.Fatalf("probe body %v; want the listed model, max_output_tokens %d, no store, no reasoning", body, probeMaxOutputTokens)
	}
}
