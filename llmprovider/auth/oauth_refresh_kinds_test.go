package auth

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestOAuthToken_NoRefreshTokenIsAuthFailure: a lapsed session that cannot
// refresh must sign in again, so its error is an ErrAuthFailure (0015-MADR
// D7, R25; 0015-PLAN deviation of 2026-09-30, found by llmtest).
func TestOAuthToken_NoRefreshTokenIsAuthFailure(t *testing.T) {
	session := &OAuthSession{Provider: llmprovider.ProviderOpenAI, Issuer: DefaultOpenAIIssuer, Access: "a-old",
		Expiry: time.Now().Add(-time.Minute)}
	_, err := session.Token(context.Background())
	if !errors.Is(err, llmprovider.ErrAuthFailure) {
		t.Fatalf("Token = %v, want an error matching ErrAuthFailure", err)
	}
}

// TestOAuthRefresh_FailureKinds: every refresh the token endpoint rejects
// maps to one kind. A 401 or a terminal code is a dead login; a rate limit
// and an outage keep their kinds; any other 4xx is a malformed request, and
// not an auth failure (TestOAuthRefresh_BadRequestIsNotRetried).
func TestOAuthRefresh_FailureKinds(t *testing.T) {
	for _, c := range []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"401", http.StatusUnauthorized, `{"error":"unauthorized"}`, llmprovider.ErrAuthFailure},
		{"terminal code", http.StatusBadRequest, `{"error":"invalid_grant"}`, llmprovider.ErrAuthFailure},
		{"other 400", http.StatusBadRequest, `{"error":"invalid_request"}`, llmprovider.ErrInvalidRequest},
		{"403", http.StatusForbidden, `{"error":"forbidden"}`, llmprovider.ErrInvalidRequest},
		{"429", http.StatusTooManyRequests, `{"error":"slow_down"}`, llmprovider.ErrRateLimited},
		{"503", http.StatusServiceUnavailable, ``, llmprovider.ErrProviderUnavailable},
	} {
		t.Run(c.name, func(t *testing.T) {
			srv, _ := refreshServer(t, func(_ int32, w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(c.status)
				_, _ = io.WriteString(w, c.body)
			})
			_, err := expiredSession(srv, DefaultOpenAIIssuer).Token(context.Background())
			if !errors.Is(err, c.want) {
				t.Fatalf("Token = %v, want an error matching %v", err, c.want)
			}
			if !errors.Is(c.want, llmprovider.ErrAuthFailure) && errors.Is(err, llmprovider.ErrAuthFailure) {
				t.Fatalf("Token = %v also matches ErrAuthFailure", err)
			}
		})
	}
}
