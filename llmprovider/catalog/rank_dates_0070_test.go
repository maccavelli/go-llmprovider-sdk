package catalog

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// grokGeneration matches the grok-N.M generations of xAI's listing, and not
// grok-build-0.1, which names no generation. It is written here, not read
// from parseVersion, so that the test builds at b5ce51d and is seen red there
// (0070-PLAN Phase 1).
var grokGeneration = regexp.MustCompile(`^grok-\d\.\d+`)

// TestRankGrok_FollowsListingDates_0070 (0070-MADR D1, D3): on xAI's recorded
// listing, a grok generation created earlier never ranks above one created
// later. At b5ce51d the grok-4.20 ids, created 2026-03-09, scored 93 against
// grok-4.3's 45.
func TestRankGrok_FollowsListingDates_0070(t *testing.T) {
	var page struct {
		Data []struct {
			ID      string `json:"id"`
			Created int64  `json:"created"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(firstPartyPage(t, "grok.json")), &page); err != nil {
		t.Fatal(err)
	}
	type dated struct {
		id      string
		created int64
	}
	var models []dated
	for _, m := range page.Data {
		if isUsableGrokModel(m.ID) && grokGeneration.MatchString(m.ID) {
			models = append(models, dated{m.ID, m.Created})
		}
	}
	if len(models) != 7 {
		t.Fatalf("the recorded listing has %d grok-N.M ids, want the 7 of 0070-MADR C17 to C19", len(models))
	}
	for _, a := range models {
		for _, b := range models {
			if a.created < b.created && Rank(llmprovider.ProviderGrok, a.id) > Rank(llmprovider.ProviderGrok, b.id) {
				t.Errorf("%s, created first, ranks %d, above %s at %d", a.id, Rank(llmprovider.ProviderGrok, a.id),
					b.id, Rank(llmprovider.ProviderGrok, b.id))
			}
		}
	}
}
