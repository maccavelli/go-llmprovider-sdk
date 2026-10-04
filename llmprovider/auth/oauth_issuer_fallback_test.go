package auth

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// hostRecorder records every request's method, host and path, and answers
// none of them.
type hostRecorder struct {
	mu   sync.Mutex
	seen []string
}

func (h *hostRecorder) RoundTrip(r *http.Request) (*http.Response, error) {
	h.mu.Lock()
	h.seen = append(h.seen, r.Method+" "+r.URL.Host+r.URL.Path)
	h.mu.Unlock()
	return nil, errors.New("not dialled")
}

func (h *hostRecorder) requests() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.seen...)
}

// TestOAuthSession_UnknownIssuerNeverRefreshesAtXAI (0020-MADR F45): a session
// with no token URL refreshes at a derived endpoint only for OpenAI's and
// xAI's own issuers. Any other issuer is a sign-in-again error, and its
// refresh token goes nowhere, least of all to auth.x.ai.
func TestOAuthSession_UnknownIssuerNeverRefreshesAtXAI(t *testing.T) {
	for _, s := range []*OAuthSession{
		{Provider: llmprovider.ProviderOpenAI, Issuer: "https://staging-auth.example"},
		{Provider: llmprovider.ProviderGrok, Issuer: "https://custom-issuer.example"},
	} {
		rec := &hostRecorder{}
		s.Access, s.Refresh, s.Expiry, s.ClientID = "old", "refresh", time.Now().Add(-time.Minute), "client"
		s.HTTPClient = &http.Client{Transport: rec}
		_, err := s.Token(context.Background())
		if !errors.Is(err, llmprovider.ErrAuthFailure) {
			t.Errorf("%s at %s: Token = %v, want ErrAuthFailure", s.Provider, s.Issuer, err)
		}
		if got := rec.requests(); len(got) != 0 {
			t.Errorf("%s at %s: sent %v, want no request", s.Provider, s.Issuer, got)
		}
	}
}

// TestRevokeOAuthSession_OnlyItsOwnIssuer (0020-MADR F45): revocation goes to
// the session's own issuer, and a provider without revocation sends nothing.
func TestRevokeOAuthSession_OnlyItsOwnIssuer(t *testing.T) {
	rec := &hostRecorder{}
	kilo := &OAuthSession{Provider: llmprovider.ProviderKilo, Access: "kilo-token", HTTPClient: &http.Client{Transport: rec}}
	if err := RevokeOAuthSession(context.Background(), kilo); !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("revoking a Kilo session = %v, want errors.ErrUnsupported", err)
	}
	if got := rec.requests(); len(got) != 0 {
		t.Errorf("revoking a Kilo session sent %v, want no request", got)
	}

	rec = &hostRecorder{}
	staging := &OAuthSession{Provider: llmprovider.ProviderOpenAI, Issuer: "https://staging-auth.example",
		Refresh: "refresh", ClientID: "client", HTTPClient: &http.Client{Transport: rec}}
	_ = RevokeOAuthSession(context.Background(), staging)
	if got := rec.requests(); len(got) != 1 || got[0] != "POST staging-auth.example/oauth/revoke" {
		t.Errorf("revoking a staging OpenAI session sent %v, want one POST staging-auth.example/oauth/revoke", got)
	}
}
