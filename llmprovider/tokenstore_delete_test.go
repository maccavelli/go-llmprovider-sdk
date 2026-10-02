package llmprovider

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestFileTokenStore_Delete: a saved session is gone after Delete, deleting an
// absent one succeeds, a bad id is refused, and a failure to remove is
// returned (0015-PLAN S8, commit 2).
func TestFileTokenStore_Delete(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := NewFileTokenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	session := &OAuthSession{Provider: ProviderOpenAI, Access: "a", Refresh: "r", Expiry: time.Now().Add(time.Hour)}
	if err := store.Save(ctx, ProviderOpenAI, session); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := store.Delete(ctx, ProviderOpenAI); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got, err := store.Load(ctx, ProviderOpenAI); err != nil || got != nil {
		t.Fatalf("Load after Delete = %v, %v; want nothing", got, err)
	}
	if err := store.Delete(ctx, ProviderOpenAI); err != nil {
		t.Errorf("Delete of an absent session: %v, want nil", err)
	}
	if err := store.Delete(ctx, "../escape"); err == nil {
		t.Error("Delete with a bad provider id: want an error")
	}
	// A directory at the session's path, with something in it, cannot be
	// removed.
	blocker := filepath.Join(dir, string(ProviderGrok)+".json")
	if err := os.MkdirAll(filepath.Join(blocker, "child"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, ProviderGrok); err == nil || errors.Is(err, os.ErrNotExist) {
		t.Errorf("Delete that cannot remove: err = %v, want it returned", err)
	}
}
