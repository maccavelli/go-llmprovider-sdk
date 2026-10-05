package catalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestList_TogetherRanksWithItsMetadata (0020-MADR F4): Together ranks from
// models.dev's togetherai section, so a free model is not recommended first.
func TestList_TogetherRanksWithItsMetadata(t *testing.T) {
	enableModelMetadata(t)
	listing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"id":"a/free","type":"chat"},{"id":"b/paid","type":"chat"}]`))
	}))
	t.Cleanup(listing.Close)
	recent := time.Now().AddDate(0, -2, 0).Format("2006-01-02")
	meta, _ := metadataServer(t, http.StatusOK, `{"togetherai":{"models":{`+
		`"a/free":{"id":"a/free","reasoning":true,"cost":{"input":0,"output":0},"release_date":"`+recent+`","limit":{"context":131072}},`+
		`"b/paid":{"id":"b/paid","reasoning":true,"cost":{"input":0.2,"output":0.6},"release_date":"`+recent+`","limit":{"context":131072}}}}}`)
	cat, err := List(context.Background(), llmprovider.ProviderTogether, llmprovider.NewStaticToken("k"),
		llmprovider.WithBaseURL(listing.URL), llmprovider.WithModelMetadataURL(meta.URL))
	if err != nil || len(cat.Recommended) == 0 || cat.Recommended[0] != "b/paid" {
		t.Errorf("Recommended = %v, %v; want the paid model first: the free one is not eligible", cat.Recommended, err)
	}
}

// TestList_FailedListingLeavesMetadataHealthy (0020-MADR F5): a listing that
// fails abandons its metadata fetch; that is not a metadata failure, and the
// next lookup finds the document.
func TestList_FailedListingLeavesMetadataHealthy(t *testing.T) {
	enableModelMetadata(t)
	listing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(listing.Close)
	meta := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(300 * time.Millisecond)
		_, _ = w.Write([]byte(smallMetadataDoc))
	}))
	t.Cleanup(meta.Close)
	if _, err := List(context.Background(), llmprovider.ProviderTogether, llmprovider.NewStaticToken("k"),
		llmprovider.WithBaseURL(listing.URL), llmprovider.WithModelMetadataURL(meta.URL)); err != nil {
		t.Fatalf("List: %v", err)
	}
	// The abandoned fetch settles first, as it does before any later lookup.
	time.Sleep(200 * time.Millisecond)
	if _, err := LookupMetadata(context.Background(), meta.URL, nil); err != nil {
		t.Errorf("LookupMetadata after a failed listing = %v; the metadata host is healthy", err)
	}
}

// TestLoadModelMetadata_LateFailureKeepsNewerDocument (0020-MADR F15): a
// failed fetch never displaces a document already cached. With one fetch per
// URL (0021-MADR C2, amendment "F15 and A6 under the shared fetch"), a
// second load joins the fetch in flight rather than racing it; when that
// fetch, a refresh, fails, the cached document is kept.
func TestLoadModelMetadata_LateFailureKeepsNewerDocument(t *testing.T) {
	enableModelMetadata(t)
	release := make(chan struct{})
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			_, _ = w.Write([]byte(smallMetadataDoc))
			return
		}
		<-release
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	cfg := testConfig(t, llmprovider.WithModelMetadataURL(srv.URL))
	if _, err := loadModelMetadata(context.Background(), cfg); err != nil {
		t.Fatalf("first load: %v", err)
	}
	later := time.Now().Add(modelMetadataTTL + time.Minute)
	metadataNow = func() time.Time { return later }
	t.Cleanup(func() { metadataNow = time.Now })

	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			if doc, err := loadModelMetadata(context.Background(), cfg); err != nil || doc == nil {
				t.Errorf("load during the refresh = %v, %v; want the cached document", doc, err)
			}
		})
	}
	wg.Wait()
	for calls.Load() < 2 {
		time.Sleep(5 * time.Millisecond)
	}
	close(release)
	waitMetadataFetches()
	if n := calls.Load(); n != 2 {
		t.Errorf("requests = %d, want 2: the first fetch and one shared refresh", n)
	}
	if doc, err := loadModelMetadata(context.Background(), cfg); err != nil || doc == nil {
		t.Errorf("load after the failed refresh = %v, %v; want the cached document", doc, err)
	}
}

// TestList_OllamaRecommendedIsItsOwnSlice (0020-MADR F34).
func TestList_OllamaRecommendedIsItsOwnSlice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"models":[{"name":"a"},{"name":"b"}]}`))
	}))
	t.Cleanup(srv.Close)
	cat, err := listT(context.Background(), llmprovider.ProviderOllama, llmprovider.NewStaticToken(""), llmprovider.WithBaseURL(srv.URL))
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

// TestLoadModelMetadata_SendsTheCallersIdentity (0020-MADR F37): the metadata
// fetch names the application the caller named, as the listing does (MADR
// 0012 §1.4).
func TestLoadModelMetadata_SendsTheCallersIdentity(t *testing.T) {
	enableModelMetadata(t)
	var agent atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		agent.Store(r.UserAgent())
		_, _ = w.Write([]byte(smallMetadataDoc))
	}))
	t.Cleanup(srv.Close)
	cfg := testConfig(t, llmprovider.WithModelMetadataURL(srv.URL), llmprovider.WithClientInfo("wire-app", "9.9.9"))
	if _, err := loadModelMetadata(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	if got, _ := agent.Load().(string); !strings.HasPrefix(got, "wire-app/9.9.9 ") {
		t.Errorf("User-Agent = %q, want it to start wire-app/9.9.9", got)
	}
}
