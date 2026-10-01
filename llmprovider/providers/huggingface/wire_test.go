package huggingface

import (
	"flag"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
)

// updateWire rewrites this package's G-wire goldens, only for a difference a
// record explains (0015-PLAN S2).
var updateWire = flag.Bool("update", false, "rewrite testdata/wire golden files")

// wireCases is llmprovider's P7 huggingface case, built through the new API.
// Its goldens moved here with the provider. The P7 type had no Continue.
var wireCases = []wirecase.Case{{
	Name:           "huggingface",
	Listing:        hfListing,
	NoContinuation: true,
	Build: func(u string, extra ...llmprovider.Option) (llmprovider.Provider, error) {
		return New(append([]llmprovider.Option{llmprovider.WithAPIKey("hf_wire"), llmprovider.WithModel("openai/gpt-oss-120b")},
			wirecase.Opts(u, extra...)...)...)
	},
}}

// TestWireGoldens is G-wire (0015-MADR D12) for Hugging Face, through the new
// API.
func TestWireGoldens(t *testing.T) {
	wirecase.Run(t, wireCases, *updateWire)
}
