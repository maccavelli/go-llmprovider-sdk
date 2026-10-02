package catalog

import (
	"context"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Helpers for the tests moved from llmprovider (0015-PLAN S8, commit 2).

// testConfig is a listing's configuration for opts, as llmprovider's
// ApplyOptions built one for these tests. It resolves for kilo, the one id
// both of catalog's own options are scoped to.
func testConfig(t *testing.T, opts ...llmprovider.Option) config {
	t.Helper()
	cfg, err := configFor(llmprovider.ProviderKilo, opts)
	if err != nil {
		t.Fatalf("configFor: %v", err)
	}
	return cfg
}

// listRecommended is List's recommendation, as llmprovider's
// ListAvailableModels returned it: nil on an error.
func listRecommended(ctx context.Context, id llmprovider.ProviderID, src llmprovider.TokenSource, opts ...llmprovider.Option) ([]string, error) {
	cat, err := listT(ctx, id, src, opts...)
	if err != nil {
		return nil, err
	}
	return cat.Recommended, nil
}
