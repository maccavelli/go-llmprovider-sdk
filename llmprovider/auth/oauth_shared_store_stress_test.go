package auth

import (
	"context"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestOAuthSession_SharedStoreStress (0026-PLAN V3): 64 goroutines over four
// sessions loaded from one FileTokenStore, as four processes would load
// them, call Token and invalidate the token they got, against an issuer that
// rotates every refresh and rejects a refresh token sent twice. No refresh
// token is sent twice, and every Token succeeds.
func TestOAuthSession_SharedStoreStress(t *testing.T) {
	srv, issuer := newRotatingIssuer(t, false)
	store, err := NewFileTokenStore(filepath.Join(t.TempDir(), "tokens"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(context.Background(), llmprovider.ProviderOpenAI, expiredStoredSession(srv, nil)); err != nil {
		t.Fatal(err)
	}
	sessions := make([]*OAuthSession, 4)
	for i := range sessions {
		s, err := store.Load(context.Background(), llmprovider.ProviderOpenAI)
		if err != nil || s == nil {
			t.Fatalf("Load: %v, %v", s, err)
		}
		s.HTTPClient = srv.Client()
		sessions[i] = s
	}
	var wg sync.WaitGroup
	for g := range 64 {
		wg.Go(func() {
			s := sessions[g%len(sessions)]
			for range 4 {
				tok, err := s.Token(context.Background())
				if err != nil {
					t.Errorf("goroutine %d: Token: %v", g, err)
					return
				}
				if g%3 == 0 {
					s.InvalidateToken(tok)
				}
				time.Sleep(time.Millisecond)
			}
		})
	}
	wg.Wait()
	sent := issuer.tokens()
	seen := map[string]bool{}
	for _, refresh := range sent {
		if seen[refresh] {
			t.Fatalf("refresh token %q was sent twice; sent %v", refresh, sent)
		}
		seen[refresh] = true
	}
	if len(sent) == 0 || !slices.Contains(sent, "old-refresh") {
		t.Fatalf("refresh tokens sent = %v; want the stored one first", sent)
	}
}
