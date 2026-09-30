package llmprovider

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func testSession(access, refresh string, expiry time.Time) *OAuthSession {
	return &OAuthSession{Provider: ProviderOpenAI, Access: access, Refresh: refresh, Expiry: expiry,
		Issuer: DefaultOpenAIIssuer, ClientID: "client-test"}
}

// TestFileTokenStore_RenameFailureKeepsPrevious (0016-PLAN T2 step 1): a
// failed rename leaves the previous session intact and no temp file behind.
func TestFileTokenStore_RenameFailureKeepsPrevious(t *testing.T) {
	store, err := NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := store.Save(ctx, ProviderOpenAI, testSession("a-1", "rt-1", time.Now().Add(time.Hour))); err != nil {
		t.Fatal(err)
	}
	planted := errors.New("planted rename failure")
	tokenStoreRename = func(string, string) error { return planted }
	t.Cleanup(func() { tokenStoreRename = os.Rename })

	if err := store.Save(ctx, ProviderOpenAI, testSession("a-2", "rt-2", time.Now().Add(time.Hour))); !errors.Is(err, planted) {
		t.Fatalf("Save = %v, want the planted failure", err)
	}
	got, err := store.Load(ctx, ProviderOpenAI)
	if err != nil || got == nil || got.Refresh != "rt-1" {
		t.Fatalf("Load = %+v, %v; want the previous session rt-1", got, err)
	}
	entries, err := os.ReadDir(store.Dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".tok-") {
			t.Errorf("temp file %s left behind", e.Name())
		}
	}
}

// TestFileTokenStore_LoadRefusesOversized: a session file over 64 KiB is
// refused rather than read.
func TestFileTokenStore_LoadRefusesOversized(t *testing.T) {
	store, err := NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	big := make([]byte, maxTokenFileBytes+1)
	for i := range big {
		big[i] = ' '
	}
	if err := os.WriteFile(filepath.Join(store.Dir, "openai.json"), big, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(context.Background(), ProviderOpenAI); err == nil || !strings.Contains(err.Error(), "larger than") {
		t.Errorf("Load = %v, want a size refusal", err)
	}
}

// sharedStoreSessions saves one expired session to a new FileTokenStore and
// loads it twice: two sessions standing in for two processes.
func sharedStoreSessions(t *testing.T, tokenURL string) (*FileTokenStore, *OAuthSession, *OAuthSession) {
	t.Helper()
	store, err := NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	expired := testSession("a-old", "rt-old", time.Now().Add(-time.Minute))
	expired.TokenURL = tokenURL
	if err := store.Save(context.Background(), ProviderOpenAI, expired); err != nil {
		t.Fatal(err)
	}
	load := func() *OAuthSession {
		s, err := store.Load(context.Background(), ProviderOpenAI)
		if err != nil || s == nil {
			t.Fatalf("Load = %v, %v", s, err)
		}
		return s
	}
	return store, load(), load()
}

// TestFileTokenStore_RefreshLock_OnceAcrossSessions (0016-MADR amendment A2):
// two sessions sharing a store, both due, refresh once in total; the second
// adopts what the first saved instead of spending the same refresh token.
func TestFileTokenStore_RefreshLock_OnceAcrossSessions(t *testing.T) {
	srv, calls := refreshServer(t, func(_ int32, w http.ResponseWriter, _ *http.Request) {
		time.Sleep(150 * time.Millisecond)
		refreshOK(w, "a-refreshed")
	})
	_, first, second := sharedStoreSessions(t, srv.URL)

	var wg sync.WaitGroup
	tokens := make([]string, 2)
	errs := make([]error, 2)
	for i, s := range []*OAuthSession{first, second} {
		wg.Go(func() {
			tok, err := s.Token(context.Background())
			tokens[i], errs[i] = tok.Value, err
		})
	}
	wg.Wait()
	if n := calls.Load(); n != 1 {
		t.Errorf("%d refresh requests, want 1: the same refresh token was spent twice", n)
	}
	for i := range tokens {
		if errs[i] != nil || tokens[i] != "a-refreshed" {
			t.Errorf("session %d: Token = %q, %v; want a-refreshed", i, tokens[i], errs[i])
		}
	}
}

// TestFileTokenStore_RefreshLock_StaleTakenOver: a lock its holder abandoned
// is taken over, and the refresh proceeds.
func TestFileTokenStore_RefreshLock_StaleTakenOver(t *testing.T) {
	srv, calls := refreshServer(t, func(_ int32, w http.ResponseWriter, _ *http.Request) { refreshOK(w, "a-refreshed") })
	store, session, _ := sharedStoreSessions(t, srv.URL)
	lock := filepath.Join(store.Dir, ProviderOpenAI+".lock")
	if err := os.WriteFile(lock, []byte("999999 dead\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * lockStaleAfter)
	if err := os.Chtimes(lock, old, old); err != nil {
		t.Fatal(err)
	}
	tok, err := session.Token(context.Background())
	if err != nil || tok.Value != "a-refreshed" || calls.Load() != 1 {
		t.Fatalf("Token = %q, %v after %d refreshes; want the stale lock taken over", tok.Value, err, calls.Load())
	}
	if _, err := os.Stat(lock); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("lock file left after the refresh: %v", err)
	}
}

// TestFileTokenStore_RefreshLock_HeldTimesOutRetryable: while another holder
// keeps the lock, a waiter gives up with a retryable error and never
// refreshes unlocked.
func TestFileTokenStore_RefreshLock_HeldTimesOutRetryable(t *testing.T) {
	srv, calls := refreshServer(t, func(_ int32, w http.ResponseWriter, _ *http.Request) { refreshOK(w, "a-refreshed") })
	store, session, _ := sharedStoreSessions(t, srv.URL)
	store.wait, store.staleAfter, store.heartbeat = 300*time.Millisecond, 200*time.Millisecond, 20*time.Millisecond
	session.Store = store

	holder := &FileTokenStore{Dir: store.Dir, staleAfter: store.staleAfter, heartbeat: store.heartbeat}
	unlock, err := holder.LockRefresh(context.Background(), ProviderOpenAI)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()

	start := time.Now()
	_, err = session.Token(context.Background())
	if !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("Token = %v, want a retryable ErrProviderUnavailable while the lock is held", err)
	}
	if calls.Load() != 0 {
		t.Errorf("%d refresh requests while locked, want 0", calls.Load())
	}
	// The heartbeat kept the lock fresh past staleAfter, so it was not taken over.
	if waited := time.Since(start); waited < store.staleAfter {
		t.Errorf("gave up after %v, before the stale window %v", waited, store.staleAfter)
	}
}
