package llmprovider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
)

// generationCounter answers every listing (GET) with an empty or minimal
// catalog and counts generation requests (POST).
func generationCounter(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var posts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			posts.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		switch {
		case r.URL.Path == "/api.json":
			_, _ = w.Write([]byte(`{}`))
		case r.URL.Query().Get("client_version") != "":
			_, _ = w.Write([]byte(`{"models":[{"slug":"gpt-6-astra","visibility":"list"}]}`))
		default:
			_, _ = w.Write([]byte(`{"data":[{"id":"gpt-4.1-mini"}]}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &posts
}

// TestDiscoverModels_MeteredServicesDoNotProbe: DiscoverModels spends no
// generation on Kilo, OpenCode, Hugging Face or a ChatGPT session (MADR 0012
// §1.6); it returns the curated listing.
func TestDiscoverModels_MeteredServicesDoNotProbe(t *testing.T) {
	for name, build := range map[string]func(url string) (discoverer, error){
		"kilo": func(url string) (discoverer, error) { return NewKilo("k", "some/model", WithBaseURL(url)) },
		"opencode": func(url string) (discoverer, error) {
			return NewOpencode(ProviderOpencodeGo, "k", "glm-5.3-flash", WithBaseURL(url), WithModelMetadataURL(url+"/api.json"))
		},
		"huggingface": func(url string) (discoverer, error) {
			return NewHuggingFace("k", "org/model", WithBaseURL(url), WithModelMetadataURL(url+"/api.json"))
		},
	} {
		t.Run(name, func(t *testing.T) {
			srv, posts := generationCounter(t)
			p, err := build(srv.URL)
			if err != nil {
				t.Fatalf("construct: %v", err)
			}
			models, err := p.DiscoverModels(context.Background())
			if err != nil || len(models) == 0 {
				t.Fatalf("DiscoverModels = %v/%v, want the curated listing", models, err)
			}
			if n := posts.Load(); n != 0 {
				t.Fatalf("DiscoverModels made %d generation requests, want 0", n)
			}
		})
	}
}

// TestDiscoverModels_ProbesFollowDefaultOptionAndEnv (0016-MADR A5): the
// providers that can probe do so by default; WithModelProbes turns them off or
// on; LLMPROVIDER_PROBES takes effect only through ModelProbesFromEnv, and an
// explicit option passed after it wins.
func TestDiscoverModels_ProbesFollowDefaultOptionAndEnv(t *testing.T) {
	builders := map[string]func(url string, opts ...ProviderOption) (discoverer, error){
		"gemini": func(url string, opts ...ProviderOption) (discoverer, error) {
			return NewGemini(context.Background(), "k", "gemini-3.7-flash", append(opts, WithBaseURL(url))...)
		},
		"grok": func(url string, opts ...ProviderOption) (discoverer, error) {
			return NewGrok("k", "grok-4.5", append(opts, WithBaseURL(url))...)
		},
	}
	for _, tc := range []struct {
		name  string
		env   string // "" leaves LLMPROVIDER_PROBES unset
		opts  func() []ProviderOption
		probe bool
	}{
		{"default", "", func() []ProviderOption { return nil }, true},
		{"option off", "", func() []ProviderOption { return []ProviderOption{WithModelProbes(false)} }, false},
		{"option on", "", func() []ProviderOption { return []ProviderOption{WithModelProbes(true)} }, true},
		{"env false without the helper", "false", func() []ProviderOption { return nil }, true},
		{"env false through the helper", "false", func() []ProviderOption { return []ProviderOption{ModelProbesFromEnv()} }, false},
		{"env true through the helper", "true", func() []ProviderOption { return []ProviderOption{ModelProbesFromEnv()} }, true},
		{"env not a boolean", "sometimes", func() []ProviderOption { return []ProviderOption{ModelProbesFromEnv()} }, true},
		{"explicit option after the helper wins", "false",
			func() []ProviderOption { return []ProviderOption{ModelProbesFromEnv(), WithModelProbes(true)} }, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.env != "" {
				t.Setenv(envModelProbes, tc.env)
			} else {
				t.Setenv(envModelProbes, "")
				if err := os.Unsetenv(envModelProbes); err != nil {
					t.Fatal(err)
				}
			}
			for name, build := range builders {
				srv, posts := generationCounter(t)
				p, err := build(srv.URL, tc.opts()...)
				if err != nil {
					t.Fatalf("%s: construct: %v", name, err)
				}
				if _, err := p.DiscoverModels(context.Background()); err != nil {
					t.Fatalf("%s: DiscoverModels: %v", name, err)
				}
				switch n := posts.Load(); {
				case !tc.probe && n != 0:
					t.Errorf("%s: %d generation requests, want none", name, n)
				case tc.probe && (n == 0 || n > MaxListedModels):
					t.Errorf("%s: %d generation requests, want one per candidate", name, n)
				}
			}
		})
	}
}
