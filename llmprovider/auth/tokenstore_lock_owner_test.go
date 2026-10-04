package auth

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// lockStore is a FileTokenStore with short lock timing and a stale lock
// file for openai already in place.
func lockStore(t *testing.T) (*FileTokenStore, string) {
	t.Helper()
	store, err := NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	store.staleAfter, store.heartbeat, store.wait = 10*time.Second, time.Hour, 300*time.Millisecond
	path := filepath.Join(store.Dir, "openai.lock")
	if err := os.WriteFile(path, []byte("1 dead-holder\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-time.Minute)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	return store, path
}

// TestFileTokenStore_StaleLockHasOneTaker (0020-MADR F12): two waiters see
// the same stale lock. The first takes it over; the second, which judged it
// stale before that, must not take the first one's fresh lock as well.
func TestFileTokenStore_StaleLockHasOneTaker(t *testing.T) {
	store, path := lockStore(t)
	ctx := context.Background()
	var first func()
	nested := false
	t.Cleanup(func() { lockBeforeTakeover = func(string) {} })
	lockBeforeTakeover = func(string) {
		if nested {
			return
		}
		nested = true
		// The second waiter pauses here; the first takes the lock meanwhile.
		unlock, err := store.LockRefresh(ctx, "openai")
		if err != nil {
			t.Errorf("first waiter: %v", err)
			return
		}
		first = unlock
	}
	second, err := store.LockRefresh(ctx, "openai")
	if err == nil {
		second()
		t.Fatal("the second waiter also holds the lock: two holders")
	}
	if !errors.Is(err, llmprovider.ErrProviderUnavailable) {
		t.Errorf("second waiter = %v, want ErrProviderUnavailable while the first holds the lock", err)
	}
	if first == nil {
		t.Fatal("the first waiter never took the lock")
	}
	if _, statErr := os.Stat(path); statErr != nil {
		t.Errorf("the first holder's lock file is gone: %v", statErr)
	}
	first()
}

// TestFileTokenStore_UnlockKeepsSuccessorsLock (0020-MADR F12): a holder that
// stalled past staleAfter, and was taken over, must not remove its
// successor's lock when it finally unlocks.
func TestFileTokenStore_UnlockKeepsSuccessorsLock(t *testing.T) {
	store, path := lockStore(t)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	stalled, err := store.LockRefresh(ctx, "openai")
	if err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-time.Minute)
	if err := os.Chtimes(path, old, old); err != nil { // the holder stalls
		t.Fatal(err)
	}
	successor, err := store.LockRefresh(ctx, "openai")
	if err != nil {
		t.Fatalf("successor: %v", err)
	}
	stalled()
	if _, statErr := os.Stat(path); statErr != nil {
		t.Errorf("the stalled holder's unlock removed its successor's lock: %v", statErr)
	}
	successor()
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("the successor's unlock left its lock: %v", statErr)
	}
}
