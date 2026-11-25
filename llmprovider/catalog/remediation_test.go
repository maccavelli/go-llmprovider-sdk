package catalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestList_TogetherRanksWithItsMetadata (0020-MADR F4): Together ranks from
// models.dev's togetherai section, so a free model is not recommended first.
func TestList_TogetherRanksWithItsMetadata(t *testing.T) {
	clearProcessStore(t)
	listing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"id":"a/free","type":"chat"},{"id":"b/paid","type":"chat"}]`))
	}))
	t.Cleanup(listing.Close)
	recent := time.Now().AddDate(0, -2, 0).Format("2006-01-02")
	meta, _ := metadataServer(t, http.StatusOK, `{"togetherai":{"models":{`+
		`"a/free":{"id":"a/free","reasoning":true,"cost":{"input":0,"output":0},"release_date":"`+recent+`","limit":{"context":131072}},`+
		`"b/paid":{"id":"b/paid","reasoning":true,"cost":{"input":0.2,"output":0.6},"release_date":"`+recent+`","limit":{"context":131072}}}}}`)
	cat, err := List(context.Background(), llmprovider.ProviderTogether, llmprovider.NewStaticToken("k"),
		llmprovider.WithBaseURL(listing.URL), servedDoc(t, meta.URL))
	if err != nil || len(cat.Recommended) == 0 || cat.Recommended[0] != "b/paid" {
		t.Errorf("Recommended = %v, %v; want the paid model first: the free one is not eligible", cat.Recommended, err)
	}
}

// TestList_OllamaRecommendedIsItsOwnSlice (0020-MADR F34).
func TestList_OllamaRecommendedIsItsOwnSlice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"models":[{"name":"a"},{"name":"b"}]}`))
	}))
	t.Cleanup(srv.Close)
	cat, err := listT(context.Background(), llmprovider.ProviderOllama, llmprovider.NewStaticToken(""), llmprovider.WithBaseURL(srv.URL), testCache(t))
	if err != nil || len(cat.Recommended) == 0 {
		t.Fatalf("List = %+v, %v", cat, err)
	}
	cat.Recommended[0] = "changed"
	if cat.Usable[0] != "a" {
		t.Errorf("Usable = %v after changing Recommended; want them independent", cat.Usable)
	}
}

// TestList_MixedCaseIDTakesItsOptions (0020-MADR F35): List accepts "Kilo",
// so it accepts Kilo's own options for it.
func TestList_MixedCaseIDTakesItsOptions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	if _, err := listT(context.Background(), "Kilo", llmprovider.NewStaticToken("k"),
		WithProfile(ProfileCapable), llmprovider.WithBaseURL(srv.URL)); err != nil {
		t.Errorf("List(Kilo, WithProfile) = %v, want nil: the id is Kilo's", err)
	}
}

// TestSortByRankDesc_KeepsListingOrderOnTies (0020-MADR F36).
func TestSortByRankDesc_KeepsListingOrderOnTies(t *testing.T) {
	ids := []string{"a", "b", "c"}
	sortByRankDesc(ids, func(id string) int {
		if id == "c" {
			return 2
		}
		return 0
	})
	if !slices.Equal(ids, []string{"c", "a", "b"}) {
		t.Errorf("sorted = %v, want [c a b]", ids)
	}
}
