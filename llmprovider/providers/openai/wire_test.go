package openai

import (
	"flag"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
)

// updateWire rewrites this package's G-wire goldens, only for a difference a
// record explains (0015-PLAN S2).
var updateWire = flag.Bool("update", false, "rewrite testdata/wire golden files")

// wireCases are llmprovider's P7 openai and chatgpt cases, built through the
// new API. Their goldens moved here with the provider.
var wireCases = []wirecase.Case{
	{
		Name:    "openai",
		Listing: wirecase.ListingDataIDs("gpt-5.5", "gpt-5.4-mini", "text-embedding-3-small"),
		Build: func(u string, extra ...llmprovider.Option) (llmprovider.Provider, error) {
			return New(append([]llmprovider.Option{llmprovider.WithAPIKey("sk-wire"), llmprovider.WithModel("gpt-5.5")},
				wirecase.Opts(u, extra...)...)...)
		},
	},
	{
		// A stateless API-key request asks for encrypted reasoning
		// (0021-MADR D4).
		Name:    "openai-stateless",
		Listing: wirecase.ListingDataIDs("gpt-5.5", "gpt-5.4-mini", "text-embedding-3-small"),
		Build: func(u string, extra ...llmprovider.Option) (llmprovider.Provider, error) {
			return New(append([]llmprovider.Option{llmprovider.WithAPIKey("sk-wire"), llmprovider.WithModel("gpt-5.5"),
				WithStore(false)}, wirecase.Opts(u, extra...)...)...)
		},
	},
	{
		Name: "chatgpt",
		Listing: `{"models":[{"slug":"gpt-6-astra","visibility":"list","priority":1,"supported_in_api":true},` +
			`{"slug":"gpt-6-luna","visibility":"list","priority":2,"supported_in_api":true}]}`,
		SSE: true,
		Build: func(u string, extra ...llmprovider.Option) (llmprovider.Provider, error) {
			session := &auth.OAuthSession{Issuer: auth.DefaultOpenAIIssuer, Access: "chatgpt-access",
				Refresh: "chatgpt-refresh", Expiry: time.Now().Add(time.Hour), AccountID: "acct-wire"}
			return New(append([]llmprovider.Option{llmprovider.WithTokenSource(session), llmprovider.WithModel("gpt-6-astra")},
				wirecase.Opts(u, extra...)...)...)
		},
	},
	{
		// A ChatGPT plan session (0043-MADR D5): the platform, streamed, store
		// false, no output limit though one is given, none of the Codex
		// backend's headers though the session has an account, deferred tools
		// appended, and no continuation; the listing is its "models" array.
		Name: "chatgpt-plan",
		Listing: `{"models":[{"slug":"gpt-6.1-sol","display_name":"GPT-6.1 Sol","visibility":"list"},` +
			`{"slug":"gpt-6-hidden","visibility":"hide"},{"slug":"gpt-6-luna","visibility":"list"}]}`,
		SSE: true,
		Build: func(u string, extra ...llmprovider.Option) (llmprovider.Provider, error) {
			session := &auth.OAuthSession{Provider: llmprovider.ProviderOpenAI, Issuer: auth.DefaultOpenAIIssuer,
				Access: "plan-access", Refresh: "plan-refresh", Expiry: time.Now().Add(time.Hour), ClientID: "oaiapp_wire",
				AccountID: "acct-wire", Resource: PlatformBaseURL}
			return New(append([]llmprovider.Option{llmprovider.WithTokenSource(session), llmprovider.WithModel("gpt-6.1-sol"),
				llmprovider.WithMaxTokens(2048)}, wirecase.Opts(u, extra...)...)...)
		},
	},
}

// TestWireGoldens is G-wire (0015-MADR D12) for OpenAI, through the new API.
func TestWireGoldens(t *testing.T) {
	wirecase.Run(t, wireCases, *updateWire)
}
