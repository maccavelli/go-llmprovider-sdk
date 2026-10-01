package together

import (
	"flag"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
)

// updateWire rewrites this package's G-wire goldens, only for a difference a
// record explains (0015-PLAN S2).
var updateWire = flag.Bool("update", false, "rewrite testdata/wire golden files")

// wireCases is llmprovider's together case (0017-PLAN U1), built through the
// new API. Its goldens moved here with the provider. The old type had no
// Continue.
var wireCases = []wirecase.Case{{
	Name: "together",
	Listing: `[{"id":"openai/gpt-oss-120b","object":"model","type":"chat","context_length":131072},` +
		`{"id":"BAAI/bge-large-en-v1.5","object":"model","type":"embedding"},` +
		`{"id":"zai-org/GLM-5.3","object":"model","type":"chat","context_length":202752}]`,
	NoContinuation: true,
	Build: func(u string, extra ...llmprovider.Option) (llmprovider.Provider, error) {
		return New(append([]llmprovider.Option{llmprovider.WithAPIKey("together-wire-key"), llmprovider.WithModel("openai/gpt-oss-120b")},
			wirecase.Opts(u, extra...)...)...)
	},
}}

// TestWireGoldens is G-wire (0015-MADR D12) for Together, through the new API.
func TestWireGoldens(t *testing.T) {
	wirecase.Run(t, wireCases, *updateWire)
}
