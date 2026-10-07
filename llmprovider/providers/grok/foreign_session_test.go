package grok

import (
	"errors"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// TestNew_RefusesAForeignSession: another provider's sign-in or CLI login is
// refused, so its token never reaches xAI (R16; 0026-MADR F2). A session with
// no Provider is judged by its issuer, and one with neither is refused
// (0026-MADR amendment of 2026-10-06).
func TestNew_RefusesAForeignSession(t *testing.T) {
	for name, src := range map[string]llmprovider.TokenSource{
		"chatgpt oauth":             &auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Issuer: auth.DefaultOpenAIIssuer, Access: "CHATGPT-ACCESS"},
		"kilo oauth":                &auth.OAuthSession{Provider: llmprovider.ProviderKilo, Access: "KILO-ACCESS"},
		"chatgpt issuer, no owner":  &auth.OAuthSession{Issuer: auth.DefaultOpenAIIssuer, Access: "CHATGPT-ACCESS"},
		"no provider and no issuer": &auth.OAuthSession{Access: "ANON-ACCESS"},
		"codex cli":                 &auth.VendorCLISession{Provider: llmprovider.ProviderOpenAI, Path: "unused"},
	} {
		p, err := New(llmprovider.WithTokenSource(src))
		if !errors.Is(err, llmprovider.ErrUnsupported) || p != nil {
			t.Errorf("%s: New err = %v, provider built = %t; want ErrUnsupported and none", name, err, p != nil)
		}
	}
}

// TestNew_AcceptsItsOwnSessionWithoutProvider: a hand-built session with xAI's
// issuer and no Provider is Grok's (0026-MADR amendment of 2026-10-06).
func TestNew_AcceptsItsOwnSessionWithoutProvider(t *testing.T) {
	src := &auth.OAuthSession{Issuer: auth.DefaultGrokOAuthIssuer, Access: "XAI-ACCESS"}
	if _, err := New(llmprovider.WithTokenSource(src)); err != nil {
		t.Fatalf("New with an xAI-issued session: %v", err)
	}
}
