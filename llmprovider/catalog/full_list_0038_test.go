package catalog

import (
	"slices"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0038-MADR amendment A1: the full list is built as the six, then the rest.
// The six are still computed on their own (0041-MADR D3, menuOf);
// fullList, which appended the uncapped ranking, gave way to the menu in
// 0041-PLAN Phase 4, and its test with it.

// TestRankRecommended_UncappedFillDiffers pins A1's finding: once the
// fallback's first six are used up, today's six fill from the listing's
// order, and an uncapped fill from the fallback's. So the six cannot be read
// off the uncapped ranking.
func TestRankRecommended_UncappedFillDiffers(t *testing.T) {
	// No candidate is eligible, so both fill. The fallback's first six hold
	// four small dense models the utility profile leaves out.
	fallback := []string{"s/one", "d/a-7b", "d/b-8b", "d/c-9b", "d/d-3b", "s/two", "r/first", "r/second"}
	usable := []string{"r/second", "r/first", "s/one", "s/two"}
	six := rankRecommended(ProfileUtility, llmprovider.ProviderKilo, nil, slices.Concat(fallback[:MaxListed], usable), MaxListed)
	all := rankRecommended(ProfileUtility, llmprovider.ProviderKilo, nil, slices.Concat(fallback, usable), 0)
	if want := []string{"s/one", "s/two", "r/second", "r/first"}; !slices.Equal(six, want) {
		t.Errorf("capped = %v; want %v", six, want)
	}
	if want := []string{"s/one", "s/two", "r/first", "r/second"}; !slices.Equal(all, want) {
		t.Errorf("uncapped = %v; want %v", all, want)
	}
}
