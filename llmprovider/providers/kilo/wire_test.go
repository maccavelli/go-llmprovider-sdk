package kilo

import (
	"flag"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
)

// updateWire rewrites this package's G-wire goldens, only for a difference a
// record explains (0015-PLAN S2).
var updateWire = flag.Bool("update", false, "rewrite testdata/wire golden files")

// wireCases is llmprovider's P7 kilo case, built through the new API. Its
// goldens moved here with the provider. The P7 type had no Continue.
var wireCases = []wirecase.Case{{
	Name: "kilo",
	Listing: `{"data":[{"id":"anthropic/claude-sonnet-5","name":"Claude Sonnet 5","created":1780000000,` +
		`"architecture":{"input_modalities":["text"],"output_modalities":["text"]},` +
		`"pricing":{"prompt":"0.000003","completion":"0.000015"},"context_length":200000,` +
		`"supported_parameters":["tools","tool_choice","reasoning","reasoning_effort"]}]}`,
	NoContinuation: true,
	Build: func(u string, extra ...llmprovider.Option) (llmprovider.Provider, error) {
		return New(append([]llmprovider.Option{llmprovider.WithAPIKey("kilo-wire-key"), llmprovider.WithModel("anthropic/claude-sonnet-5")},
			wirecase.Opts(u, extra...)...)...)
	},
}}

// TestWireGoldens is G-wire (0015-MADR D12) for Kilo, through the new API.
func TestWireGoldens(t *testing.T) {
	wirecase.Run(t, wireCases, *updateWire)
}
