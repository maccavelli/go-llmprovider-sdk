package opencode

import (
	"flag"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
)

// updateWire rewrites this package's G-wire goldens, only for a difference a
// record explains (0015-PLAN S2).
var updateWire = flag.Bool("update", false, "rewrite testdata/wire golden files")

// wireCase is llmprovider's P7 opencode case for one gateway route, built
// through the new API. The P7 type had no Continue.
func wireCase(gateway llmprovider.ProviderID, route, model, listing string) wirecase.Case {
	return wirecase.Case{
		Name:           string(gateway) + "-" + route,
		Listing:        listing,
		NoContinuation: true,
		Build: func(u string, extra ...llmprovider.Option) (llmprovider.Provider, error) {
			return newFor(gateway)(append([]llmprovider.Option{llmprovider.WithAPIKey("opencode-wire-key"), llmprovider.WithModel(model)},
				wirecase.Opts(u, extra...)...)...)
		},
	}
}

var (
	zenListing = wirecase.ListingDataIDs("gpt-5.5", "claude-sonnet-5", "gemini-3.8-flash", "glm-5.3")
	goListing  = wirecase.ListingDataIDs("grok-4.7", "qwen3.8-flash", "kimi-k3")
)

// wireCases are the seven P7 opencode cases. Their goldens moved here with
// the provider.
var wireCases = []wirecase.Case{
	wireCase(llmprovider.ProviderOpencodeZen, "responses", "gpt-5.5", zenListing),
	wireCase(llmprovider.ProviderOpencodeZen, "messages", "claude-sonnet-5", zenListing),
	wireCase(llmprovider.ProviderOpencodeZen, "google", "gemini-3.8-flash", zenListing),
	wireCase(llmprovider.ProviderOpencodeZen, "chat", "glm-5.3", zenListing),
	wireCase(llmprovider.ProviderOpencodeGo, "responses", "grok-4.7", goListing),
	wireCase(llmprovider.ProviderOpencodeGo, "messages", "qwen3.8-flash", goListing),
	wireCase(llmprovider.ProviderOpencodeGo, "chat", "kimi-k3", goListing),
}

// TestWireGoldens is G-wire (0015-MADR D12) for OpenCode, through the new API.
func TestWireGoldens(t *testing.T) {
	wirecase.Run(t, wireCases, *updateWire)
}
