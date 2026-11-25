package wizard

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// TestConfigureLLM_ProviderOptionsReachTheChatGPTProvider: Options.ProviderOptions
// are added to the openai provider the wizard builds to list a ChatGPT session, here a
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
	f := newFake(t, fakePrompter{blankSearches: true, selects: []int{providerIdx(t, llmprovider.ProviderOpenAI), 0}, confirms: []bool{true}})
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
