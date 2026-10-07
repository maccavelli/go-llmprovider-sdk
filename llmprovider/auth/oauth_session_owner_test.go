package auth

import (
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestOAuthSession_Owner: Provider when set, else the provider whose issuer
// the session names, else none (0026-MADR F2 and its amendment of
// 2026-10-06).
func TestOAuthSession_Owner(t *testing.T) {
	cases := []struct {
		name string
		s    *OAuthSession
		want llmprovider.ProviderID
	}{
		{"provider set", &OAuthSession{Provider: llmprovider.ProviderKilo, Issuer: DefaultOpenAIIssuer}, llmprovider.ProviderKilo},
		{"openai issuer", &OAuthSession{Issuer: DefaultOpenAIIssuer + "/"}, llmprovider.ProviderOpenAI},
		{"xai issuer", &OAuthSession{Issuer: DefaultGrokOAuthIssuer}, llmprovider.ProviderGrok},
		{"other issuer", &OAuthSession{Issuer: "https://login.example"}, ""},
		{"nothing", &OAuthSession{}, ""},
	}
	for _, c := range cases {
		if got := c.s.Owner(); got != c.want {
			t.Errorf("%s: Owner() = %q, want %q", c.name, got, c.want)
		}
	}
}

// TestOAuthSession_ChatGPTWithoutIssuer: an OpenAI session with no issuer is
// an older ChatGPT sign-in, as its refresh treats it (0026-MADR F2).
func TestOAuthSession_ChatGPTWithoutIssuer(t *testing.T) {
	if !(&OAuthSession{Provider: llmprovider.ProviderOpenAI}).ChatGPT() {
		t.Error("an OpenAI session with no issuer is not ChatGPT")
	}
	if (&OAuthSession{Provider: llmprovider.ProviderGrok}).ChatGPT() {
		t.Error("a Grok session with no issuer is ChatGPT")
	}
}
