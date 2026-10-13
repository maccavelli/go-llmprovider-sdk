package catalog

import (
	"slices"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0038-MADR amendment A1: the full list is built as the six, then the rest.

// TestFullList_SixFirst: the full list is the six computed as before, then
// the uncapped ranking's other ids, even where the uncapped ranking's first
// six differ (A1).
func TestFullList_SixFirst(t *testing.T) {
	curate := func(_ []string, max int) []string {
		if max == MaxListed {
			return []string{"a", "b", "c", "d", "e", "f"}
		}
		return []string{"b", "a", "c", "d", "e", "g", "f", "h"}
	}
	if got, want := fullList(nil, curate), []string{"a", "b", "c", "d", "e", "f", "g", "h"}; !slices.Equal(got, want) {
		t.Errorf("fullList = %v; want %v", got, want)
	}
}

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
