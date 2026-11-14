package auth

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestOAuthSession_UseStore_0074: UseStore binds the store, the last call
// wins, and a nil store changes nothing.
func TestOAuthSession_UseStore_0074(t *testing.T) {
	first, err := NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	session := &OAuthSession{}
	session.UseStore(first)
	if session.Store != first {
		t.Fatalf("Store = %v after UseStore(first); want first", session.Store)
	}
	session.UseStore(nil)
	if session.Store != first {
		t.Errorf("Store = %v after UseStore(nil); want first, unchanged", session.Store)
	}
	session.UseStore(second)
	if session.Store != second {
		t.Errorf("Store = %v after UseStore(second); want second: the last call wins", session.Store)
	}
}

// TestFileTokenStoreSave_RefreshSaveCompletes_0074 (0074-PLAN Phase 7 step
// 6): FileTokenStore.Save reads the session under its lock, and the
// session's own refresh saves a private copy without holding that lock, so a
// refresh that saves through the store completes: it neither deadlocks nor
// races. A direct Save of the live session, while another goroutine
// refreshes it, completes too. The wait is bounded.
func TestFileTokenStoreSave_RefreshSaveCompletes_0074(t *testing.T) {
	srv, _ := refreshServer(t, func(_ int32, w http.ResponseWriter, _ *http.Request) { refreshOK(w, "a-refreshed") })
	store, err := NewFileTokenStore(filepath.Join(t.TempDir(), "tokens"))
	if err != nil {
		t.Fatal(err)
	}
	session := expiredSession(srv, DefaultOpenAIIssuer)
	session.Store = store
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	done := make(chan error, 2)
	go func() { _, err := session.Token(ctx); done <- err }()
	go func() {
		for range 20 {
			if err := store.Save(ctx, llmprovider.ProviderOpenAI, session); err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()
	for range 2 {
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-ctx.Done():
			t.Fatal("a refresh that saves through the store did not complete: deadlock")
		}
	}
	saved, err := store.Load(ctx, llmprovider.ProviderOpenAI)
	if err != nil || saved == nil || saved.Access == "" {
		t.Fatalf("Load = %v, %v; want the saved session", saved, err)
	}
}
