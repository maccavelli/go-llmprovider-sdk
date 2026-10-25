package catalog_test

import (
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
)

// noLayer is a layer that knows no model.
type noLayer struct{}

func (noLayer) Name() string { return "none" }

func (noLayer) Model(llmprovider.ProviderID, string) (catalog.ModelInfo, bool) {
	return catalog.ModelInfo{}, false
}

// TestWithModelLayer_Scoped_0033 (R18; 0033-PLAN-public-model-metadata.md
// Phase 3, as 0061-MADR's amendment "E2 and E4's menu, made exact before
// Phase 4c", X8, restates it): WithModelLayer is accepted by a built-in
// provider's New, and, a shared option, taken by any other id's options,
// which hold its layer for a provider that reads it.
func TestWithModelLayer_Scoped_0033(t *testing.T) {
	if _, err := providers.New(llmprovider.ProviderClaude, llmprovider.WithAPIKey("k"), llmprovider.WithModel("m"),
		catalog.WithModelLayer(noLayer{})); err != nil {
		t.Errorf("a built-in provider refused WithModelLayer: %v", err)
	}
	st, err := llmprovider.ResolveOptions("custom-provider", []llmprovider.Option{catalog.WithModelLayer(noLayer{})})
	if err != nil || len(st.Values()) != 1 {
		t.Errorf("another id: %v, %d value(s); want WithModelLayer taken, its layer one value", err, len(st.Values()))
	}
}
