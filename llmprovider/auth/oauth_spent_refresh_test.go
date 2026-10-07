package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
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

// countingLocker is a FileTokenStore that counts its LockRefresh calls.
type countingLocker struct {
	*FileTokenStore
	attempts atomic.Int32
}

func (c *countingLocker) LockRefresh(ctx context.Context, provider llmprovider.ProviderID) (func(), error) {
	c.attempts.Add(1)
	return c.FileTokenStore.LockRefresh(ctx, provider)
}

// TestOAuthSession_PendingResaveDoesNotBlock (0021-MADR T10): while another
// process holds the refresh lock, a session with a valid token and an unsaved
// rotation does not wait for the lock on each call. Five calls take under
// 100 ms in all, with at most one lock attempt.
func TestOAuthSession_PendingResaveDoesNotBlock(t *testing.T) {
	file, err := NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	file.wait = 300 * time.Millisecond
	holder := &FileTokenStore{Dir: file.Dir}
	unlock, err := holder.LockRefresh(context.Background(), llmprovider.ProviderOpenAI)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()

	store := &countingLocker{FileTokenStore: file}
	session := testSession("a-current", "rt-rotated", time.Now().Add(time.Hour))
	session.Store, session.storedRefresh = store, "rt-spent" // the store holds the token the rotation spent
	start := time.Now()
	for call := 1; call <= 5; call++ {
		if tok, err := session.Token(context.Background()); err != nil || tok.Value != "a-current" {
			t.Fatalf("call %d: Token = %q, %v; want the current token", call, tok.Value, err)
		}
	}
	if elapsed, n := time.Since(start), store.attempts.Load(); elapsed > 100*time.Millisecond || n > 1 {
		t.Errorf("5 calls took %v with %d lock attempts; want under 100 ms and at most 1", elapsed, n)
	}
}
