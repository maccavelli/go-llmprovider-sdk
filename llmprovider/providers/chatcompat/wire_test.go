package chatcompat

import (
	"flag"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
)

// updateWire rewrites this package's G-wire goldens, only for a difference a
// record explains (0015-PLAN S2).
var updateWire = flag.Bool("update", false, "rewrite testdata/wire golden files")

// wireCases is chatcompat's G-wire case (0061-PLAN Phase 4): a cache of each
// scenario's own, as every case (0058-MADR D1).
var wireCases = []wirecase.Case{{
	Name:           "chatcompat",
	Listing:        wirecase.ListingDataIDs("acme-model", "acme-small"),
	NoContinuation: true,
	Build: func(u string, extra ...llmprovider.Option) (llmprovider.Provider, error) {
		return New(append([]llmprovider.Option{WithID(testID), llmprovider.WithAPIKey("sk-wire"),
			llmprovider.WithModel("acme-model"), catalog.WithCache(new(catalog.Cache))}, wirecase.Opts(u, extra...)...)...)
	},
}}

// TestWireGoldens is G-wire (0015-MADR D12) for chatcompat.
func TestWireGoldens(t *testing.T) {
	wirecase.Run(t, wireCases, *updateWire)
}
