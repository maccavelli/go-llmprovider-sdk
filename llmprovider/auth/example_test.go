package auth_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// ExampleNewFileTokenStore saves a provider's session in a directory private
// to the current user, and loads it back. A real session comes from a
// sign-in, such as LoginDeviceOAuth.
func ExampleNewFileTokenStore() {
	dir, err := os.MkdirTemp("", "tokens")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer os.RemoveAll(dir)

	store, err := auth.NewFileTokenStore(dir)
	if err != nil {
		fmt.Println(err)
		return
	}
	ctx := context.Background()
	session := &auth.OAuthSession{Provider: llmprovider.ProviderGrok, Access: "access-token", Refresh: "refresh-token"}
	if err := store.Save(ctx, llmprovider.ProviderGrok, session); err != nil {
		fmt.Println(err)
		return
	}
	loaded, err := store.Load(ctx, llmprovider.ProviderGrok)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(loaded.Provider)
	// Output: grok
}

// ExampleOAuthSession_Wait waits for a refresh that its caller gave up on.
// The refresh runs on, detached, so that the issuer's rotation is kept; Wait
// returns once it is done, and the next Token gets its access token. The
// httptest server stands in for the issuer, and answers slowly.
func ExampleOAuthSession_Wait() {
	issuer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "fresh-access", "refresh_token": "fresh-refresh", "expires_in": 3600})
	}))
	defer issuer.Close()

	session := &auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Access: "old-access", Refresh: "old-refresh",
		Expiry: time.Now().Add(-time.Minute), ClientID: "your-client", TokenURL: issuer.URL, HTTPClient: issuer.Client()}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := session.Token(ctx)
	fmt.Println(err)
	fmt.Println(session.Wait(context.Background()))
	token, err := session.Token(context.Background())
	fmt.Println(token.Value, err)
	// Output:
	// context deadline exceeded
	// <nil>
	// fresh-access <nil>
}
