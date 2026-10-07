package wizard

import (
	"context"
	"slices"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// TestChatGPTCatalog_Capped (0026-MADR F47): a ChatGPT session's listing
// recommends at most catalog.MaxListed models, as every other listing does,
// and search still covers every listed model.
func TestChatGPTCatalog_Capped(t *testing.T) {
	var built []*thirdParty
	listed := listedIDs(14)
	o := Options{Registry: thirdPartyRegistry(t, thirdParty{models: listed}, true, &built)}
	d := llmprovider.Descriptor{ID: thirdPartyID, Label: "Acme"}
	cat, err := chatGPTCatalog(context.Background(), o, d, Result{}, llmprovider.NewStaticToken("k"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cat.Recommended) > catalog.MaxListed || !slices.Equal(cat.Usable, listed) {
		t.Fatalf("Recommended has %d models, Usable %d; want at most %d recommended, and all %d usable",
			len(cat.Recommended), len(cat.Usable), catalog.MaxListed, len(listed))
	}
}
