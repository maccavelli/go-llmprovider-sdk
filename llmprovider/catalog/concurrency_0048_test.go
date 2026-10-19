package catalog

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"slices"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0048-MADR D3: what the docs of Catalog and Metadata say of concurrent use,
// run under -race.

// TestCatalog_SharesNothing_0048: catalogs listed at once from one cached
// listing share nothing: each goroutine changes its own while the others read
// theirs, and the next listing is unchanged.
func TestCatalog_SharesNothing_0048(t *testing.T) {
	enableModelMetadata(t)
	listing := serveBody(t, togetherItem10Listing)
	meta, _ := metadataServer(t, http.StatusOK, togetherItem10Metadata)
	opts := []llmprovider.Option{llmprovider.WithBaseURL(listing.URL), llmprovider.WithModelMetadataURL(meta.URL)}
	first := listCatalog(context.Background(), t, llmprovider.ProviderTogether, opts...)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			cat, err := listT(context.Background(), llmprovider.ProviderTogether, llmprovider.NewStaticToken(""), opts...)
			if err != nil {
				t.Errorf("List: %v", err)
				return
			}
			<-start
			_ = slices.Concat(cat.Recommended, cat.Usable)
			for i := range cat.Recommended {
				cat.Recommended[i] = "changed"
			}
			for i := range cat.Usable {
				cat.Usable[i] = "changed"
			}
			for i := range cat.Menu {
				cat.Menu[i].ID = "changed"
				cat.Menu[i].Caveats = append(cat.Menu[i].Caveats[:0], CaveatNoTools)
			}
		})
	}
	close(start)
	wg.Wait()
	again := listCatalog(context.Background(), t, llmprovider.ProviderTogether, opts...)
	if !slices.Equal(again.Recommended, first.Recommended) || !slices.Equal(again.Usable, first.Usable) ||
		len(again.Menu) != len(first.Menu) || again.Menu[0].ID != first.Menu[0].ID {
		t.Errorf("after the catalogs were changed, a listing gives %v; want %v", again.Recommended, first.Recommended)
	}
}

// TestMetadata_ConcurrentReads_0048: one document, read at once by 16
// goroutines through every exported method, gives each the same answers: its
// lazy decode runs once.
func TestMetadata_ConcurrentReads_0048(t *testing.T) {
	m := fixtureMetadata0033(t)
	var want bytes.Buffer
	if err := fixtureMetadata0033(t).Encode(&want); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			info, ok := m.Model(llmprovider.ProviderClaude, "claude-x")
			if !ok || info.ID != "claude-x" {
				t.Errorf("Model = %v, %v; want claude-x", info.ID, ok)
			}
			for _, p := range []llmprovider.ProviderID{llmprovider.ProviderOpenAI, llmprovider.ProviderGemini,
				llmprovider.ProviderGrok, llmprovider.ProviderTogether} {
				if len(m.Models(p)) != 1 {
					t.Errorf("Models(%s): %d; want the fixture's one", p, len(m.Models(p)))
				}
			}
			_ = m.ReasoningEfforts(llmprovider.ProviderClaude, "claude-x")
			_ = m.InterleavedField(llmprovider.ProviderClaude, "claude-x")
			var got bytes.Buffer
			if err := m.Encode(io.MultiWriter(&got)); err != nil || !bytes.Equal(got.Bytes(), want.Bytes()) {
				t.Errorf("Encode = %v; want the document's bytes", err)
			}
		})
	}
	wg.Wait()
}
