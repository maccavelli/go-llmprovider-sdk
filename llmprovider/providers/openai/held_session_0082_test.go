package openai

// 0082-PLAN Phase 1's tests in providers/openai: a provider never sends the
// token of a sign-in its held session refused to adopt (0082-MADR S4, S5).
// Every server is in process.

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

type heldStore0082 struct {
	mu  sync.Mutex
	rec *auth.OAuthSession
}

func (m *heldStore0082) Load(context.Context, llmprovider.ProviderID) (*auth.OAuthSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.rec == nil {
		return nil, nil
	}
	r := m.rec
	return &auth.OAuthSession{Provider: r.Provider, Access: r.Access, Refresh: r.Refresh, Expiry: r.Expiry,
		Issuer: r.Issuer, ClientID: r.ClientID, AccountID: r.AccountID, TokenURL: r.TokenURL,
		Resource: r.Resource, Scope: r.Scope, HostID: r.HostID, Kind: r.Kind, Consents: r.Consents}, nil
}

func (m *heldStore0082) Save(context.Context, llmprovider.ProviderID, *auth.OAuthSession) error {
	return nil
}

func (m *heldStore0082) Delete(context.Context, llmprovider.ProviderID) error { return nil }

func (m *heldStore0082) set(rec *auth.OAuthSession) {
	m.mu.Lock()
	m.rec = rec
	m.mu.Unlock()
}

// bearers0082 is a server that refuses every token with a 401 and records
// the bearer tokens it was sent.
func bearers0082(t *testing.T) (*httptest.Server, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"expired","type":"invalid_request_error","code":"invalid_api_key"}}`))
	}))
	t.Cleanup(srv.Close)
	return srv, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), seen...)
	}
}

func list0082(t *testing.T, p llmprovider.Provider) error {
	t.Helper()
	lister, ok := p.(llmprovider.ModelLister)
	if !ok {
		t.Fatal("the provider lists no models")
	}
	_, err := lister.ListModels(context.Background())
	return err
}

// T7 (MADR S4): a provider built over a plan session never sends another
// application's ChatGPT token that its store now holds.
func TestPlanProviderNeverSendsAnotherSignInsToken_0082(t *testing.T) {
	srv, seen := bearers0082(t)
	store := &heldStore0082{}
	held := &auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Kind: auth.SessionChatGPTPlan,
		Access: "held-plan-access", Refresh: "held-plan-refresh", Expiry: time.Now().Add(time.Hour),
		Issuer: auth.DefaultOpenAIIssuer, ClientID: "oaiapp_held", TokenURL: srv.URL + "/token",
		Resource: "https://api.openai.com/v1", HostID: "host-held", Store: store}
	p := build(t, llmprovider.WithTokenSource(held), llmprovider.WithBaseURL(srv.URL))
	store.set(&auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Kind: auth.SessionChatGPT,
		Access: "other-access", Refresh: "other-refresh", Expiry: time.Now().Add(time.Hour),
		Issuer: auth.DefaultOpenAIIssuer, ClientID: "app_other", AccountID: "acct-other", TokenURL: srv.URL + "/token"})
	err := list0082(t, p)
	for _, bearer := range seen() {
		if bearer == "other-access" {
			t.Errorf("the other sign-in's token was sent: requests %q", seen())
		}
	}
	// The server answers 401 to everything, so an auth failure alone could be
	// its answer: the text must be the held session's refusal.
	if !errors.Is(err, llmprovider.ErrAuthFailure) || !strings.Contains(err.Error(), "another sign-in") {
		t.Errorf("ListModels err = %v; want the held session's refusal, an auth failure naming another sign-in", err)
	}
}

// T8 (MADR S5): a provider built over a ChatGPT login never sends a plan
// token its store now holds to the Codex backend.
func TestChatGPTProviderNeverSendsAPlanToken_0082(t *testing.T) {
	srv, seen := bearers0082(t)
	store := &heldStore0082{}
	held := &auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Kind: auth.SessionChatGPT,
		Access: "held-chatgpt-access", Refresh: "held-chatgpt-refresh", Expiry: time.Now().Add(time.Hour),
		Issuer: auth.DefaultOpenAIIssuer, ClientID: "app_held", AccountID: "acct-held", TokenURL: srv.URL + "/token",
		Store: store}
	p := build(t, llmprovider.WithTokenSource(held), llmprovider.WithBaseURL(srv.URL))
	store.set(&auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Kind: auth.SessionChatGPTPlan,
		Access: "plan-access", Refresh: "plan-refresh", Expiry: time.Now().Add(time.Hour),
		Issuer: auth.DefaultOpenAIIssuer, ClientID: "oaiapp_other", TokenURL: srv.URL + "/token",
		Resource: "https://api.openai.com/v1", HostID: "host-other"})
	err := list0082(t, p)
	for _, bearer := range seen() {
		if bearer == "plan-access" {
			t.Errorf("the plan token was sent to the Codex route: requests %q", seen())
		}
	}
	// The server answers 401 to everything, so an auth failure alone could be
	// its answer: the text must be the held session's refusal.
	if !errors.Is(err, llmprovider.ErrAuthFailure) || !strings.Contains(err.Error(), "another sign-in") {
		t.Errorf("ListModels err = %v; want the held session's refusal, an auth failure naming another sign-in", err)
	}
}
