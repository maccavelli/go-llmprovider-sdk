//go:build unix

package auth

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestFileTokenStore_TempIs0600BeforeWrite (0016-PLAN T2 step 1): the temp
// file is 0600 before its first byte is written.
func TestFileTokenStore_TempIs0600BeforeWrite(t *testing.T) {
	store, err := NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var mode os.FileMode
	var size int64 = -1
	tokenStoreBeforeWrite = func(name string) {
		if info, err := os.Stat(name); err == nil {
			mode, size = info.Mode().Perm(), info.Size()
		}
	}
	t.Cleanup(func() { tokenStoreBeforeWrite = func(string) {} })
	if err := store.Save(context.Background(), "openai", &OAuthSession{Access: "at", Refresh: "rt"}); err != nil {
		t.Fatal(err)
	}
	if mode != 0o600 || size != 0 {
		t.Errorf("before the first write: mode %o, size %d; want 0600 and empty", mode, size)
	}
}

func TestFileTokenStore_SaveMode0600(t *testing.T) {
	dir := t.TempDir()
	fs, err := NewFileTokenStore(dir)
	if err != nil {
		t.Fatalf("NewFileTokenStore: %v", err)
	}
	sess := &OAuthSession{
		Provider: "openai",
		Access:   "at",
		Refresh:  "rt",
		Expiry:   time.Now().Add(time.Hour),
		Issuer:   "https://auth.openai.com",
		ClientID: "app_test",
	}
	if err := fs.Save(context.Background(), "openai", sess); err != nil {
		t.Fatalf("Save: %v", err)
	}
	info, err := os.Stat(filepath.Join(dir, "openai.json"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("mode = %o, want 0600", perm)
	}
}
