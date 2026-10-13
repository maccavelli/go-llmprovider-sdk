package catalog

import (
	"slices"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0009-MADR's amendment "no small dense model in the utility six (§3 item
// 11)", built by 0033-PLAN-refresh-the-curated-catalogs.md Phase 7b.

// TestRankRecommended_SmallDenseUtility: under the utility profile, an id that
// names a dense model of 14B or less is neither ranked nor filled, on any
// provider; a larger one, a mixture of experts (one of 14B or less in total
// among them) and a size-less id stay; under the capable profile the small
// dense model is ranked.
func TestRankRecommended_SmallDenseUtility(t *testing.T) {
	cands := []rankCandidate{
		good("a/model-9b", "a", 0.1, 10),
		good("b/model-14B", "b", 0.1, 10),
		good("c/model-15b", "c", 0.2, 10),
		good("d/moe-30b-a3b", "d", 0.2, 10),
		good("h/moe-7b-a1b", "h", 0.25, 10),
		good("e/model-flash", "e", 0.3, 10),
	}
	fill := []string{"f/fill-7b", "g/fill-ok"}
	for _, provider := range []llmprovider.ProviderID{llmprovider.ProviderTogether, llmprovider.ProviderKilo} {
		assertRanked(t, rankRecommended(ProfileUtility, provider, cands, fill, MaxListed),
			[]string{"c/model-15b", "d/moe-30b-a3b", "h/moe-7b-a1b", "e/model-flash", "g/fill-ok"})
		if got := rankRecommended(ProfileCapable, provider, cands, fill, MaxListed); !slices.Contains(got, "a/model-9b") {
			t.Errorf("%s capable: %v, want a/model-9b ranked", provider, got)
		}
	}
}
