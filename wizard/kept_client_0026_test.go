package wizard

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// TestConfigure_KeptSessionUsesCallersClient (0026-MADR F10): a kept session
// refreshes through Options.HTTPClient, the caller's proxy, TLS roots or test
// transport, as the listing does (0016-MADR D8; 0020-MADR F20).
func TestConfigure_KeptSessionUsesCallersClient(t *testing.T) {
	store := newMemoryTokenStore()
	client := &http.Client{}
	o := Options{
		Existing: storedExisting(t, store, Result{Provider: llmprovider.ProviderOpenAI, Kind: CredOAuth,
			TokenExpiry: time.Now().Add(-time.Minute), Issuer: auth.DefaultOpenAIIssuer,
			ClientID: auth.DefaultOpenAIClientID}, "kept-access-abcd", "kept-refresh"),
		TokenStore: store,
		HTTPClient: client,
	}
	f := newFake(t, fakePrompter{confirms: []bool{true}})
	d := llmprovider.Descriptor{ID: llmprovider.ProviderOpenAI, Label: "OpenAI"}
	session, kept, err := keepExistingOAuth(context.Background(), f, d, o)
	if err != nil || !kept {
		t.Fatalf("keepExistingOAuth = %v, %v; want the session kept", kept, err)
	}
	if session.HTTPClient != client {
		t.Fatal("the kept session does not refresh through Options.HTTPClient")
	}
}
