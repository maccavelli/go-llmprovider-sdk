package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// staleStore holds one session. Its first Save fails, so after a rotation it
// still holds the session that rotation spent; later saves succeed.
type staleStore struct {
	mu     sync.Mutex
	held   *OAuthSession
	failed bool
}

func (s *staleStore) Load(context.Context, llmprovider.ProviderID) (*OAuthSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.held == nil {
		return nil, nil
	}
	return s.held.persistable(), nil
}

func (s *staleStore) Save(_ context.Context, _ llmprovider.ProviderID, session *OAuthSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.failed {
		s.failed = true
		return errors.New("save failed")
	}
	s.held = session.persistable()
	return nil
}

func (s *staleStore) Delete(context.Context, llmprovider.ProviderID) error { return nil }

// TestOAuthSession_FailedSaveNeverReusesSpentRefresh (0020-MADR F1): after a
// rotation whose save failed, the store still holds the spent refresh token.
// The next refresh must send the rotated token, not adopt the stored one as a
// sibling's rotation: an issuer that rejects reuse would revoke the family.
func TestOAuthSession_FailedSaveNeverReusesSpentRefresh(t *testing.T) {
	var mu sync.Mutex
	var sent []string
	used := map[string]bool{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("ParseForm: %v", err)
		}
		mu.Lock()
		defer mu.Unlock()
		refresh := r.Form.Get("refresh_token")
		sent = append(sent, refresh)
		if used[refresh] {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"refresh_token_reused"}`))
			return
		}
		used[refresh] = true
		n := len(sent)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": fmt.Sprintf("access-%d", n), "refresh_token": fmt.Sprintf("refresh-%d", n), "expires_in": 1800,
		})
	}))
	t.Cleanup(srv.Close)

	stale := &OAuthSession{Provider: "openai", Access: "old-access", Refresh: "old-refresh",
		Expiry: time.Now().Add(-time.Minute), ClientID: "client-test", TokenURL: srv.URL}
	store := &staleStore{held: stale.persistable()}
	session := stale.persistable()
	session.Store, session.HTTPClient = store, srv.Client()

	if tok, err := session.Token(context.Background()); err != nil || tok.Value != "access-1" {
		t.Fatalf("first Token = %q, %v; want access-1 despite the failed save", tok.Value, err)
	}
	session.Invalidate()
	tok, err := session.Token(context.Background())
	if err != nil || tok.Value != "access-2" {
		t.Errorf("second Token = %q, %v; want access-2 from the rotated refresh token", tok.Value, err)
	}
	mu.Lock()
	defer mu.Unlock()
	if want := []string{"old-refresh", "refresh-1"}; fmt.Sprint(sent) != fmt.Sprint(want) {
		t.Errorf("refresh tokens sent = %v, want %v: the spent token must never be sent again", sent, want)
	}
}
