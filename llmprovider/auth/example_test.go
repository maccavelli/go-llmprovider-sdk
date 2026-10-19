package auth_test

import (
	"context"
	"fmt"
	"os"

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
