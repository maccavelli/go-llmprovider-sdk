//go:build live_gateways

package catalog

import (
	"cmp"
	"slices"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0038-PLAN-full-curated-catalog-by-default.md Phase 4: each live listing's
// full list starts with its six, holds no id measured not to answer, and is
// a subset of Usable. Listings only, with the metadata document on as at
// runtime; no generation is sent. Together's two listings are billed, so
// they need LLMPROVIDER_LIVE_TOGETHER=1, as liveEnvKey says:
//
//	go test -tags live_gateways ./llmprovider/catalog -run TestLive_FullCatalog -v

// TestLive_FullCatalog lists every provider whose key is set, and the public
// listings of OpenCode Zen and Go, Kilo and Hugging Face, with no option and
// with WithLimit(MaxListed).
func TestLive_FullCatalog(t *testing.T) {
	type listing struct {
		name     string
		provider llmprovider.ProviderID
		key      func(t *testing.T) string
	}
	var listings []listing
	for provider, name := range llmprovider.ProviderEnvVars() {
		listings = append(listings, listing{string(provider) + " (keyed)", provider, func(t *testing.T) string { return liveEnvKey(t, name) }})
	}
	for _, provider := range []llmprovider.ProviderID{llmprovider.ProviderOpencodeZen, llmprovider.ProviderOpencodeGo,
		llmprovider.ProviderKilo, llmprovider.ProviderHuggingFace} {
		listings = append(listings, listing{string(provider) + " (public)", provider, func(*testing.T) string { return "" }})
	}
	slices.SortFunc(listings, func(a, b listing) int { return cmp.Compare(a.name, b.name) })
	for _, l := range listings {
		t.Run(l.name, func(t *testing.T) {
			src := llmprovider.NewStaticToken(l.key(t))
			ctx, cancel := liveCtx(t)
			defer cancel()
			full, err := List(ctx, l.provider, src)
			if err != nil || !full.Live {
				t.Skipf("listing unavailable: live=%t err=%v catalog err=%v", full.Live, err, full.Err)
			}
			six, err := List(ctx, l.provider, src, WithLimit(MaxListed))
			if err != nil || !six.Live {
				t.Fatalf("the limited listing failed: live=%t err=%v catalog err=%v", six.Live, err, six.Err)
			}
			if len(six.Recommended) > MaxListed || len(full.Recommended) < len(six.Recommended) ||
				!slices.Equal(full.Recommended[:len(six.Recommended)], six.Recommended) {
				t.Errorf("the six %v are not the full list's first %d: %v", six.Recommended, len(six.Recommended),
					full.Recommended[:min(len(full.Recommended), MaxListed)])
			}
			for _, id := range full.Recommended {
				if excludedAtRequest(l.provider, id) {
					t.Errorf("the full list holds %q, measured not to answer", id)
				}
				if !slices.Contains(full.Usable, id) {
					t.Errorf("the full list holds %q, which Usable does not", id)
				}
			}
			t.Logf("usable %d, full %d, six %v", len(full.Usable), len(full.Recommended), six.Recommended)
		})
	}
}
