package auth

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestFileTokenStore_KeepsFedRAMP: the flag survives a save and a load, so a
// reloaded session still sends the header.
func TestFileTokenStore_KeepsFedRAMP(t *testing.T) {
	store, err := NewFileTokenStore(filepath.Join(t.TempDir(), "tokens"))
	if err != nil {
		t.Fatal(err)
	}
	in := &OAuthSession{Provider: llmprovider.ProviderOpenAI, Access: "a", Refresh: "r", Expiry: time.Now().Add(time.Hour),
		Issuer: DefaultOpenAIIssuer, ClientID: DefaultOpenAIClientID, FedRAMP: true}
	if err := store.Save(context.Background(), llmprovider.ProviderOpenAI, in); err != nil {
		t.Fatal(err)
	}
	out, err := store.Load(context.Background(), llmprovider.ProviderOpenAI)
	if err != nil || !out.FedRAMP {
		t.Fatalf("Load = %+v, %v; want FedRAMP", out, err)
	}
}
