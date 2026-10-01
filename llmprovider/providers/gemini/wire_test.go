package gemini

import (
	"flag"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
)

// updateWire rewrites this package's G-wire goldens, only for a difference a
// record explains (0015-PLAN S2).
var updateWire = flag.Bool("update", false, "rewrite testdata/wire golden files")

// wireCases is llmprovider's P7 gemini case, built through the new API. Its
// goldens moved here with the provider.
var wireCases = []wirecase.Case{{
	Name: "gemini",
	Listing: `{"models":[{"name":"models/gemini-3.7-flash","supportedGenerationMethods":["generateContent"]},` +
		`{"name":"models/text-embedding-005","supportedGenerationMethods":["embedContent"]}]}`,
	ContinueOpts: []llmprovider.Option{WithStore(true)},
	Build: func(u string, extra ...llmprovider.Option) (llmprovider.Provider, error) {
		return New(append([]llmprovider.Option{llmprovider.WithAPIKey("gemini-wire-key"), llmprovider.WithModel("gemini-3.7-flash")},
			wirecase.Opts(u, extra...)...)...)
	},
}}

// TestWireGoldens is G-wire (0015-MADR D12) for Gemini, through the new API.
func TestWireGoldens(t *testing.T) {
	wirecase.Run(t, wireCases, *updateWire)
}
