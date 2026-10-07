package openai

import (
	"errors"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// TestNew_RefusesAForeignSession: another provider's sign-in or CLI login is
// refused, so its token never reaches OpenAI (R16; 0026-MADR F2). A session
// with no Provider is judged by its issuer, and one with neither is refused
// (0026-MADR amendment of 2026-10-06).
func TestNew_RefusesAForeignSession(t *testing.T) {
	for name, src := range map[string]llmprovider.TokenSource{
		"grok oauth":                &auth.OAuthSession{Provider: llmprovider.ProviderGrok, Issuer: auth.DefaultGrokOAuthIssuer, Access: "XAI-ACCESS"},
		"kilo oauth":                &auth.OAuthSession{Provider: llmprovider.ProviderKilo, Access: "KILO-ACCESS"},
		"xai issuer, no owner":      &auth.OAuthSession{Issuer: auth.DefaultGrokOAuthIssuer, Access: "XAI-ACCESS"},
		"no provider and no issuer": &auth.OAuthSession{Access: "ANON-ACCESS"},
		"grok cli":                  &auth.VendorCLISession{Provider: llmprovider.ProviderGrok, Path: "unused"},
	} {
		p, err := New(llmprovider.WithTokenSource(src))
		if !errors.Is(err, llmprovider.ErrUnsupported) || p != nil {
			t.Errorf("%s: New err = %v, provider built = %t; want ErrUnsupported and none", name, err, p != nil)
		}
	}
}

// TestIsChatGPTSession_NoIssuer: an OpenAI session with no issuer is an older
// ChatGPT sign-in, as its refresh treats it, so it goes to the ChatGPT backend,
// never to the platform API as a key (0026-MADR F2).
func TestIsChatGPTSession_NoIssuer(t *testing.T) {
	cases := []struct {
		name string
		src  llmprovider.TokenSource
		want bool
	}{
		{"chatgpt issuer", &auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Issuer: auth.DefaultOpenAIIssuer}, true},
		{"chatgpt issuer, no provider", &auth.OAuthSession{Issuer: auth.DefaultOpenAIIssuer}, true},
		{"no issuer", &auth.OAuthSession{Provider: llmprovider.ProviderOpenAI}, true},
		{"other issuer", &auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Issuer: "https://login.example"}, false},
		{"codex cli", &auth.VendorCLISession{Provider: llmprovider.ProviderOpenAI, Path: "unused"}, true},
		{"api key", llmprovider.NewStaticToken("sk-x"), false},
	}
	for _, c := range cases {
		if got := isChatGPTSession(c.src); got != c.want {
			t.Errorf("%s: isChatGPTSession = %v, want %v", c.name, got, c.want)
		}
	}
}
