package catalog

import (
	"context"
	"net/http"
	"slices"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0009-MADR's amendment "Together's ids measured not to serve (§3 item 12)",
// built by 0033-PLAN-refresh-the-curated-catalogs.md Phase 7c.

const togetherItem12Listing = `[
 {"id":"MiniMaxAI/MiniMax-M2.7","type":"chat"},
 {"id":"t/covered-a","type":"chat"},
 {"id":"t/covered-b","type":"chat"}
]`

// togetherItem12Metadata covers all three, M2.7 the cheapest, so that only
// item 12 can keep it out.
const togetherItem12Metadata = `{"togetherai":{"models":{
 "MiniMaxAI/MiniMax-M2.7":{"name":"M2.7","family":"minimax","reasoning":true,"tool_call":true,"release_date":"2026-09-01",
  "limit":{"context":196608,"output":8192},"cost":{"input":0.01,"output":0.02}},
 "t/covered-a":{"name":"A","family":"a","reasoning":true,"tool_call":true,"release_date":"2026-09-01",
  "limit":{"context":131072,"output":8192},"cost":{"input":0.1,"output":0.4}},
 "t/covered-b":{"name":"B","family":"b","reasoning":true,"tool_call":true,"release_date":"2026-08-01",
  "limit":{"context":131072,"output":8192},"cost":{"input":0.2,"output":0.8}}
}}}`

// TestTogetherMeasuredUnserved: on Together, an id measured not to answer is
// neither ranked nor filled, under either profile, and stays in Usable; on
// another provider the same id is ranked.
func TestTogetherMeasuredUnserved(t *testing.T) {
	pinRankingNow(t, refNow)
	enableModelMetadata(t)
	listing := serveBody(t, togetherItem12Listing)
	meta, _ := metadataServer(t, http.StatusOK, togetherItem12Metadata)
	const unserved = "MiniMaxAI/MiniMax-M2.7"
	for _, profile := range []Profile{ProfileUtility, ProfileCapable} {
		cat := listCatalog(context.Background(), t, llmprovider.ProviderTogether,
			llmprovider.WithBaseURL(listing.URL), llmprovider.WithModelMetadataURL(meta.URL), WithProfile(profile))
		if slices.Contains(cat.Recommended, unserved) {
			t.Errorf("profile %d: Recommended %v holds %s, which Together does not serve", profile, cat.Recommended, unserved)
		}
		if !slices.Contains(cat.Usable, unserved) {
			t.Errorf("profile %d: Usable %v lacks %s; item 12 keeps it searchable", profile, cat.Usable, unserved)
		}
	}
	cands := []rankCandidate{good(unserved, "minimaxai", 0.03, 10), good("x/other", "x", 0.5, 10)}
	if got := rankRecommended(ProfileUtility, llmprovider.ProviderHuggingFace, cands, nil, MaxListed); !slices.Contains(got, unserved) {
		t.Errorf("huggingface: %v, want %s ranked; item 12 is Together's alone", got, unserved)
	}
}
