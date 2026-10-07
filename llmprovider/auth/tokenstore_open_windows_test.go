//go:build windows

package auth

import (
	"context"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestFileTokenStore_SaveWhileOpenForRead: a reader holding the token file
// open, as a Load in another process does, does not make the next Save's
// rename fail (0026-MADR F42). With os.Open, which does not share delete,
// the rename is refused.
func TestFileTokenStore_SaveWhileOpenForRead(t *testing.T) {
	store, err := NewFileTokenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	session := &OAuthSession{Provider: llmprovider.ProviderOpenAI, Access: "a1", Refresh: "r1", Expiry: time.Now().Add(time.Hour)}
	if err := store.Save(context.Background(), llmprovider.ProviderOpenAI, session); err != nil {
		t.Fatal(err)
	}
	reader, err := openShared(store.path(llmprovider.ProviderOpenAI))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()
	session.Access, session.Refresh = "a2", "r2"
	if err := store.Save(context.Background(), llmprovider.ProviderOpenAI, session); err != nil {
		t.Fatalf("Save while a reader holds the file: %v", err)
	}
	loaded, err := store.Load(context.Background(), llmprovider.ProviderOpenAI)
	if err != nil || loaded == nil || loaded.Refresh != "r2" {
		t.Fatalf("Load after the save = %v, %v; want r2", loaded, err)
	}
}
