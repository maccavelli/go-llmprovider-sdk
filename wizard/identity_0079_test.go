package wizard

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// The wizard names its own requests and its logout with Options.Identity
// (0079-MADR D5; 0079-PLAN Phase 3, step 4).

// agents0079 records the User-Agent of each request a handler sees.
type agents0079 struct {
	mu   sync.Mutex
	seen []string
}

func (a *agents0079) add(r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.seen = append(a.seen, r.Header.Get("User-Agent"))
}

func (a *agents0079) all() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.seen...)
}

// wantAgents0079 fails unless there is a request, and each begins with prefix.
func wantAgents0079(t *testing.T, what string, got []string, prefix string) {
	t.Helper()
	if len(got) == 0 {
		t.Errorf("%s: no request was sent", what)
	}
	for _, agent := range got {
		if !strings.HasPrefix(agent, prefix) {
			t.Errorf("%s: User-Agent %q, want it to begin %q", what, agent, prefix)
		}
	}
}

// TestWizardOwnRequestsNamed_0079: with Options.Identity.Name set, the
// listing check, the Kilo profile and the Ollama check lead their User-Agent
// with the application, unless ProviderOptions name another.
func TestWizardOwnRequestsNamed_0079(t *testing.T) {
	probe := auth.Identity{Name: "probe-app", Version: "1.2.3"}
	for _, tc := range []struct {
		name     string
		identity auth.Identity
		extra    []llmprovider.Option
		prefix   string
	}{
		{"identity", probe, nil, "probe-app/1.2.3 ("},
		{"WithClientInfo in ProviderOptions wins", probe,
			[]llmprovider.Option{llmprovider.WithClientInfo("other-app", "2.0.0")}, "other-app/2.0.0 ("},
		{"no identity", auth.Identity{}, nil, "go-llmprovider-sdk/"},
	} {
		providerOptions := func(t *testing.T) []llmprovider.Option {
			return append([]llmprovider.Option{llmprovider.WithoutModelMetadata(), testCache(t)}, tc.extra...)
		}
		t.Run(tc.name+"/listing check", func(t *testing.T) {
			var seen agents0079
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				seen.add(r)
				_, _ = io.WriteString(w, zenListing(zenSearchIDs))
			}))
			t.Cleanup(srv.Close)
			f := newFake(t, fakePrompter{
				selects: []int{providerIdx(t, llmprovider.ProviderOpencodeZen), 0},
				inputs:  []string{srv.URL, "sonnet"}, secrets: []string{testKey},
			})
			if _, err := ConfigureLLM(context.Background(), f, Options{Discover: true, DiscoverLimit: 5 * time.Second,
				Identity: tc.identity, ProviderOptions: providerOptions(t)}); err != nil {
				t.Fatalf("ConfigureLLM: %v", err)
			}
			wantAgents0079(t, "the listing", seen.all(), tc.prefix)
		})
		t.Run(tc.name+"/Ollama check", func(t *testing.T) {
			var seen agents0079
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				seen.add(r)
				_, _ = io.WriteString(w, `{"version":"0.6.0","models":[]}`)
			}))
			t.Cleanup(srv.Close)
			f := newFake(t, fakePrompter{inputs: []string{srv.URL}})
			if _, _, err := ResolveCredential(context.Background(), f, Options{Identity: tc.identity,
				ProviderOptions: providerOptions(t)}, llmprovider.ProviderOllama); err != nil {
				t.Fatalf("ResolveCredential: %v", err)
			}
			wantAgents0079(t, "the endpoint check", seen.all(), tc.prefix)
		})
		t.Run(tc.name+"/ChatGPT session listing", func(t *testing.T) {
			var mu sync.Mutex
			var agents, originators []string
			client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				mu.Lock()
				agents = append(agents, r.Header.Get("User-Agent"))
				originators = append(originators, r.Header.Get("originator"))
				mu.Unlock()
				return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Request: r,
					Body: io.NopCloser(strings.NewReader(`{"models":[{"slug":"gpt-6-astra","visibility":"list"}]}`))}, nil
			})}
			store := newMemoryTokenStore()
			f := newFake(t, fakePrompter{blankSearches: true, selects: []int{providerIdx(t, llmprovider.ProviderOpenAI), 0}, confirms: []bool{true}})
			if _, err := ConfigureLLM(context.Background(), f, Options{
				Existing: storedExisting(t, store, Result{Provider: llmprovider.ProviderOpenAI, Kind: CredOAuth,
					TokenExpiry: time.Now().Add(time.Hour), Issuer: auth.DefaultOpenAIIssuer, ClientID: auth.DefaultOpenAIClientID},
					"existing-access-abcd", "existing-refresh"),
				TokenStore: store, Discover: true, HTTPClient: client, Identity: tc.identity, ProviderOptions: tc.extra,
			}); err != nil {
				t.Fatalf("ConfigureLLM: %v", err)
			}
			mu.Lock()
			defer mu.Unlock()
			wantAgents0079(t, "the ChatGPT session listing", agents, tc.prefix)
			wantOriginator := strings.TrimSuffix(strings.SplitN(tc.prefix, "/", 2)[0], " (")
			for _, got := range originators {
				if got != wantOriginator {
					t.Errorf("the ChatGPT session listing: originator %q, want %q", got, wantOriginator)
				}
			}
		})
		t.Run(tc.name+"/Kilo profile", func(t *testing.T) {
			stubKilo(t, auth.KiloAccount{HasPersonalAccount: true}, nil)
			var profileOpts []llmprovider.Option
			kiloProfile = func(_ context.Context, _ string, opts ...llmprovider.Option) (auth.KiloAccount, error) {
				profileOpts = opts
				return auth.KiloAccount{HasPersonalAccount: true}, nil
			}
			f := newFake(t, fakePrompter{selects: []int{providerIdx(t, llmprovider.ProviderKilo), 1, 0},
				inputs: []string{acceptDefault}, blankSearches: true})
			if _, err := ConfigureLLM(context.Background(), f, Options{TokenStore: newMemoryTokenStore(),
				Identity: tc.identity, ProviderOptions: tc.extra}); err != nil {
				t.Fatalf("ConfigureLLM: %v", err)
			}
			st, err := llmprovider.ResolveOptions(llmprovider.ProviderKilo, profileOpts)
			if err != nil {
				t.Fatal(err)
			}
			wantAgents0079(t, "the Kilo profile", []string{st.UserAgent()}, tc.prefix)
		})
	}
}

// TestWizardLogoutNamed_0079: Logout gives a stored Grok session the
// Options.Identity before it revokes it, so the discovery and the revocation
// carry the name.
func TestWizardLogoutNamed_0079(t *testing.T) {
	for _, tc := range []struct {
		name     string
		identity auth.Identity
		prefix   string
	}{
		{"identity", auth.Identity{Name: "probe-app", Version: "1.2.3"}, "probe-app/1.2.3 ("},
		{"no identity", auth.Identity{}, "go-llmprovider-sdk/"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			var requests []string
			var agents []string
			client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				mu.Lock()
				requests = append(requests, r.Method+" "+r.URL.Path)
				agents = append(agents, r.Header.Get("User-Agent"))
				mu.Unlock()
				body := ""
				if r.Method == http.MethodGet {
					body = `{"revocation_endpoint":"https://auth.x.ai/oauth2/revoke"}`
				}
				return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Request: r,
					Body: io.NopCloser(strings.NewReader(body))}, nil
			})}
			store := newMemoryTokenStore()
			if err := store.Save(context.Background(), llmprovider.ProviderGrok, testOAuthSession(llmprovider.ProviderGrok)); err != nil {
				t.Fatal(err)
			}
			f := newFake(t, fakePrompter{confirms: []bool{true}})
			if err := Logout(context.Background(), f, Options{TokenStore: store, HTTPClient: client, Identity: tc.identity},
				llmprovider.ProviderGrok); err != nil {
				t.Fatalf("Logout() error = %v", err)
			}
			mu.Lock()
			defer mu.Unlock()
			if len(requests) != 2 || requests[0] != "GET /.well-known/openid-configuration" || requests[1] != "POST /oauth2/revoke" {
				t.Fatalf("requests %q, want the discovery, then the revocation", requests)
			}
			wantAgents0079(t, "the discovery and the revocation", agents, tc.prefix)
			if tc.identity.Name != "" && !strings.Contains(agents[1], "go-llmprovider-sdk/") {
				t.Errorf("revocation User-Agent %q, want it to name this module after the application", agents[1])
			}
		})
	}
}
