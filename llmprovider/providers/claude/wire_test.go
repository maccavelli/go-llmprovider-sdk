package claude

import (
	"flag"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
)

// updateWire rewrites this package's G-wire goldens, only for a difference a
// record explains (0015-PLAN S2).
var updateWire = flag.Bool("update", false, "rewrite testdata/wire golden files")

// wireCases is llmprovider's P7 claude case, built through the new API. Its
// goldens moved here with the provider. The P7 type had no Continue.
var wireCases = []wirecase.Case{{
	Name:           "claude",
	Listing:        `{"data":[{"id":"claude-sonnet-5","type":"model"},{"id":"claude-haiku-4-5","type":"model"}],"has_more":false}`,
	NoContinuation: true,
	Build: func(u string, extra ...llmprovider.Option) (llmprovider.Provider, error) {
		return New(append([]llmprovider.Option{llmprovider.WithAPIKey("sk-ant-wire"), llmprovider.WithModel("claude-sonnet-5")},
			wirecase.Opts(u, extra...)...)...)
	},
}}

// TestWireGoldens is G-wire (0015-MADR D12) for Claude, through the new API.
func TestWireGoldens(t *testing.T) {
	wirecase.Run(t, wireCases, *updateWire)
}
