package wizard

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// TestConfigureLLM_ProviderOptionsReachTheListing (0015-PLAN S10):
// Options.ProviderOptions are added to the wizard's model listing, here a
// metadata URL the listing then fetches.
func TestConfigureLLM_ProviderOptionsReachTheListing(t *testing.T) {
	var metaHits atomic.Int32
	meta := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		metaHits.Add(1)
		_, _ = io.WriteString(w, `{"opencode":{"models":{}}}`)
	}))
	t.Cleanup(meta.Close)
	srv := zenServer(t, http.StatusOK, zenListing(zenSearchIDs))
	f := &fakePrompter{
		t: t, selects: []int{providerIdx(t, llmprovider.ProviderOpencodeZen), 0},
		inputs: []string{srv.URL, "sonnet"}, secrets: []string{testKey},
	}
	if _, err := ConfigureLLM(context.Background(), f, Options{Discover: true, DiscoverLimit: 5 * time.Second,
		ProviderOptions: []llmprovider.Option{llmprovider.WithModelMetadataURL(meta.URL)}}); err != nil {
		t.Fatalf("ConfigureLLM: %v", err)
	}
	if metaHits.Load() == 0 {
		t.Error("the listing never fetched the metadata URL ProviderOptions named")
	}
}

// TestConfigureLLM_ProviderOptionsReachTheChatGPTProvider: they are added to
// the openai provider the wizard builds to list a ChatGPT session, here a
// client identity its listing then sends.
func TestConfigureLLM_ProviderOptionsReachTheChatGPTProvider(t *testing.T) {
	var mu sync.Mutex
	var agents []string
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		mu.Lock()
		agents = append(agents, r.Header.Get("User-Agent"))
		mu.Unlock()
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Request: r,
			Body: io.NopCloser(strings.NewReader(`{"models":[{"slug":"gpt-6-astra","visibility":"list"}]}`))}, nil
	})}
	store := newMemoryTokenStore()
	f := &fakePrompter{t: t, blankSearches: true, selects: []int{providerIdx(t, llmprovider.ProviderOpenAI), 0}, confirms: []bool{true}}
	if _, err := ConfigureLLM(context.Background(), f, Options{
		Existing: storedExisting(t, store, Result{Provider: llmprovider.ProviderOpenAI, Kind: CredOAuth,
			TokenExpiry: time.Now().Add(time.Hour), Issuer: auth.DefaultOpenAIIssuer, ClientID: auth.DefaultOpenAIClientID},
			"existing-access-abcd", "existing-refresh"),
		TokenStore: store, Discover: true, HTTPClient: client,
		ProviderOptions: []llmprovider.Option{llmprovider.WithClientInfo("my-app", "1.2.3")},
	}); err != nil {
		t.Fatalf("ConfigureLLM: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(agents) != 1 || !strings.HasPrefix(agents[0], "my-app/1.2.3 ") {
		t.Errorf("listing User-Agents = %q, want one naming my-app", agents)
	}
}
