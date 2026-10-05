package auth

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
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
	store.staleAfter, store.heartbeat, store.wait = 100*time.Millisecond, time.Hour, 300*time.Millisecond
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
	// The first waiter's heartbeat keeps its lock fresh, so the second never
	// judges it stale (0021-MADR T9).
	store.heartbeat = 20 * time.Millisecond
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

// TestFileTokenStore_SkewedHeartbeatKeepsLock (0021-MADR T9): a holder whose
// clock runs 40 s behind keeps touching its lock. Every mtime it writes is in
// the waiter's past, but it changes, so the waiter times out rather than take
// the lock over.
func TestFileTokenStore_SkewedHeartbeatKeepsLock(t *testing.T) {
	store, path := lockStore(t)
	store.staleAfter, store.wait = 200*time.Millisecond, 300*time.Millisecond
	stop, stopped := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(stopped)
		base := time.Now().Add(-40 * time.Second)
		for i := 1; ; i++ {
			select {
			case <-stop:
				return
			case <-time.After(20 * time.Millisecond):
			}
			at := base.Add(time.Duration(i) * 20 * time.Millisecond)
			_ = os.Chtimes(path, at, at)
		}
	}()
	defer func() { close(stop); <-stopped }()

	unlock, err := store.LockRefresh(context.Background(), "openai")
	if err == nil {
		unlock()
		t.Fatal("the waiter took over a lock its holder kept touching")
	}
	if !errors.Is(err, llmprovider.ErrProviderUnavailable) {
		t.Errorf("LockRefresh = %v, want ErrProviderUnavailable", err)
	}
	if got, _ := os.ReadFile(path); string(got) != "1 dead-holder\n" {
		t.Errorf("lock file holds %q, want the holder's token", got)
	}
}

// TestFileTokenStore_HeartbeatSurvivesReadError (0021-MADR T9): one failed
// read of the lock skips a heartbeat; the next ones still touch it.
func TestFileTokenStore_HeartbeatSurvivesReadError(t *testing.T) {
	store, path := lockStore(t)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	store.heartbeat = 20 * time.Millisecond
	var failed atomic.Bool
	lockRead = func(p string) (string, error) {
		if failed.CompareAndSwap(false, true) {
			return "", errors.New("planted read failure")
		}
		return readLockToken(p)
	}
	t.Cleanup(func() { lockRead = readLockToken })
	unlock, err := store.LockRefresh(context.Background(), "openai")
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	old := time.Now().Add(-time.Minute)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(time.Second); ; time.Sleep(10 * time.Millisecond) {
		if info, err := os.Stat(path); err == nil && time.Since(info.ModTime()) < 10*time.Second {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the heartbeat stopped touching the lock after one failed read")
		}
	}
	if !failed.Load() {
		t.Error("the planted read failure never happened")
	}
}

// TestFileTokenStore_TakeoverRenameRetried (0021-MADR T9): a takeover whose
// rename fails once is tried again, and the waiter gets the lock.
func TestFileTokenStore_TakeoverRenameRetried(t *testing.T) {
	store, _ := lockStore(t)
	store.staleAfter = 50 * time.Millisecond
	var failed atomic.Bool
	lockRename = func(from, to string) error {
		if failed.CompareAndSwap(false, true) {
			return errors.New("planted rename failure")
		}
		return os.Rename(from, to)
	}
	t.Cleanup(func() { lockRename = os.Rename })
	unlock, err := store.LockRefresh(context.Background(), "openai")
	if err != nil {
		t.Fatalf("LockRefresh = %v, want the takeover tried again", err)
	}
	unlock()
	if !failed.Load() {
		t.Error("the planted rename failure never happened")
	}
}
