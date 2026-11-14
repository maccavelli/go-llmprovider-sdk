package wizard

import (
	"context"
	"errors"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// TestOpenURLErrorWarns_0074 (0074-MADR D4, F6): an Options.OpenURL error
// gives one warning notice and the sign-in still completes by the loopback.
func TestOpenURLErrorWarns_0074(t *testing.T) {
	stubBrowserLogin(t, func(_ context.Context, provider llmprovider.ProviderID, opts auth.OAuthFlowOptions) (*auth.OAuthSession, error) {
		_ = opts.OpenURL("https://authorize.test") // the error is the wizard's to show
		return testOAuthSession(provider), nil     // the loopback returned
	})
	f := newFake(t, fakePrompter{selects: []int{grokBrowserIdx}})
	res, _, err := ResolveCredential(context.Background(), f, Options{TokenStore: newMemoryTokenStore(), NoEndpointPrompt: true,
		NoPastePrompt: true, OpenURL: func(string) error { return errors.New("no display") }}, llmprovider.ProviderGrok)
	if err != nil || res.Kind != CredOAuth {
		t.Fatalf("kind %q, error %v; want the session", res.Kind, err)
	}
	const want = "could not open the browser (no display); open the URL above"
	if n := countContaining(f.seenNotify, want); n != 1 {
		t.Errorf("notices %q; want %q once", f.seenNotify, want)
	}
}
