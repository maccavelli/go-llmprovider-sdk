package catalog_test

import (
	"errors"
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
// Phase 3): WithModelLayer is accepted by a built-in provider's New, and
// refused for any other id.
func TestWithModelLayer_Scoped_0033(t *testing.T) {
	if _, err := providers.New(llmprovider.ProviderClaude, llmprovider.WithAPIKey("k"), llmprovider.WithModel("m"),
		catalog.WithModelLayer(noLayer{})); err != nil {
		t.Errorf("a built-in provider refused WithModelLayer: %v", err)
	}
	if _, err := llmprovider.ResolveOptions("custom-provider", []llmprovider.Option{catalog.WithModelLayer(noLayer{})}); !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Errorf("another id: %v, want WithModelLayer refused with ErrInvalidRequest", err)
	}
}
