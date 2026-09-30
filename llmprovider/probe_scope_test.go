package llmprovider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
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
		"chatgpt": func(url string) (discoverer, error) {
			session := &OAuthSession{Issuer: DefaultOpenAIIssuer, Access: "a", Expiry: time.Now().Add(time.Hour)}
			return NewOpenAIWithSource(session, "gpt-6-astra", WithBaseURL(url))
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

// TestDiscoverModels_ProbesOnlyWhenEnabled (0016-MADR D9, 0016-PLAN T3 step 2,
// replacing the test that pinned probing by default): the providers that can
// probe send no generation while listing unless WithModelProbes(true), and
// then one per candidate, at most MaxListedModels.
func TestDiscoverModels_ProbesOnlyWhenEnabled(t *testing.T) {
	for name, build := range map[string]func(url string, opts ...ProviderOption) (discoverer, error){
		"openai": func(url string, opts ...ProviderOption) (discoverer, error) {
			return NewOpenAI("k", "gpt-4.1-mini", append(opts, WithBaseURL(url))...)
		},
		"claude": func(url string, opts ...ProviderOption) (discoverer, error) {
			return NewClaude("k", "claude-haiku-4-5", append(opts, WithBaseURL(url))...)
		},
		"gemini": func(url string, opts ...ProviderOption) (discoverer, error) {
			return NewGemini(context.Background(), "k", "gemini-3.7-flash", append(opts, WithBaseURL(url))...)
		},
		"grok": func(url string, opts ...ProviderOption) (discoverer, error) {
			return NewGrok("k", "grok-4.5", append(opts, WithBaseURL(url))...)
		},
	} {
		t.Run(name, func(t *testing.T) {
			for _, probe := range []bool{false, true} {
				srv, posts := generationCounter(t)
				p, err := build(srv.URL, WithModelProbes(probe))
				if err != nil {
					t.Fatalf("construct: %v", err)
				}
				if _, err := p.DiscoverModels(context.Background()); err != nil {
					t.Fatalf("DiscoverModels: %v", err)
				}
				switch n := posts.Load(); {
				case !probe && n != 0:
					t.Errorf("by default: %d generation requests, want none", n)
				case probe && (n == 0 || n > MaxListedModels):
					t.Errorf("with WithModelProbes(true): %d generation requests, want one per candidate", n)
				}
			}
		})
	}
}
