package catalog_test

import (
	"errors"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// TestWithCache_AnyProvider_0061 (0061-MADR, amendment "Phase 4's options"):
// WithCache is a shared option, which every provider's New takes, a third
// party's too; WithLimit stays scoped to the built-ins (0038-MADR D2).
func TestWithCache_AnyProvider_0061(t *testing.T) {
	for _, id := range []llmprovider.ProviderID{llmprovider.ProviderClaude, "third-party"} {
		if _, err := llmprovider.ResolveOptions(id, []llmprovider.Option{catalog.WithCache(new(catalog.Cache))}); err != nil {
			t.Errorf("%s resolved WithCache with %v; want it taken", id, err)
		}
	}
	if _, err := llmprovider.ResolveOptions("third-party", []llmprovider.Option{catalog.WithLimit(2)}); !errors.Is(err,
		llmprovider.ErrInvalidRequest) {
		t.Errorf("a third party resolved WithLimit with %v; want ErrInvalidRequest", err)
	}
}
