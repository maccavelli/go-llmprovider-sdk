package catalog

import (
	"context"
	"net/http"
	"slices"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0009-MADR's amendment "Together ranks only what models.dev lists (§3 item
// 10)", built by 0033-PLAN-refresh-the-curated-catalogs.md Phase 7a.

// togetherItem10Listing lists two models models.dev covers and two it does
// not, all chat models.
const togetherItem10Listing = `[
 {"id":"t/covered-a","type":"chat"},
 {"id":"t/covered-b","type":"chat"},
 {"id":"t/uncovered-a","type":"chat"},
 {"id":"t/uncovered-b","type":"chat"}
]`

// togetherItem10Metadata covers only the t/covered-* models.
const togetherItem10Metadata = `{"togetherai":{"models":{
 "t/covered-a":{"name":"A","family":"a","reasoning":true,"tool_call":true,"release_date":"2026-09-01",
  "limit":{"context":131072,"output":8192},"cost":{"input":0.1,"output":0.4}},
 "t/covered-b":{"name":"B","family":"b","reasoning":true,"tool_call":true,"release_date":"2026-08-01",
  "limit":{"context":131072,"output":8192},"cost":{"input":0.2,"output":0.8}}
}}}`

// TestTogetherRanksOnlyModelsDevIDs: on Together, an id models.dev's
// togetherai section does not list is neither ranked nor filled, under either
// profile, and stays in Usable.
func TestTogetherRanksOnlyModelsDevIDs(t *testing.T) {
	pinRankingNow(t, refNow)
	enableModelMetadata(t)
	listing := serveBody(t, togetherItem10Listing)
	meta, _ := metadataServer(t, http.StatusOK, togetherItem10Metadata)
	for _, profile := range []Profile{ProfileUtility, ProfileCapable} {
		cat := listCatalog(context.Background(), t, llmprovider.ProviderTogether,
			llmprovider.WithBaseURL(listing.URL), llmprovider.WithModelMetadataURL(meta.URL), WithProfile(profile))
		for _, id := range cat.Recommended {
			if id == "t/uncovered-a" || id == "t/uncovered-b" {
				t.Errorf("profile %d: Recommended %v holds %s, which models.dev does not list", profile, cat.Recommended, id)
			}
		}
		if !slices.Contains(cat.Recommended, "t/covered-a") || !slices.Contains(cat.Recommended, "t/covered-b") {
			t.Errorf("profile %d: Recommended %v, want both covered models", profile, cat.Recommended)
		}
		for _, id := range []string{"t/covered-a", "t/covered-b", "t/uncovered-a", "t/uncovered-b"} {
			if !slices.Contains(cat.Usable, id) {
				t.Errorf("profile %d: Usable %v lacks %s; item 10 keeps every id searchable", profile, cat.Usable, id)
			}
		}
	}
}

// TestTogetherWithoutMetadata: with the document disabled, item 10 does not
// apply, and Together's fallback curation is unchanged: it fills from the
// listing, uncovered ids included.
func TestTogetherWithoutMetadata(t *testing.T) {
	listing := serveBody(t, togetherItem10Listing)
	cat := listCatalog(context.Background(), t, llmprovider.ProviderTogether, llmprovider.WithBaseURL(listing.URL))
	if want := curateTogether(cat.Usable); !slices.Equal(cat.Recommended, want) {
		t.Errorf("Recommended %v, want the fallback curation %v", cat.Recommended, want)
	}
	if !slices.Contains(cat.Recommended, "t/uncovered-a") {
		t.Errorf("Recommended %v, want the fallback to fill from the whole listing", cat.Recommended)
	}
}
