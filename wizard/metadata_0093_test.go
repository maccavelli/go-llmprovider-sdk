package wizard

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// TestWizard_HandedDocument_0093 (0093-MADR MH22; D2, D6): a document handed
// in through Options.ProviderOptions reaches the wizard's model listing, whose
// recommended menu it ranks; with none, the listing's fallback curators rank
// the menu; the wizard asks for no document either way. The listing and the
// document are the committed 2026-10-12 snapshot's OpenCode Zen listing and
// its models.dev document.
func TestWizard_HandedDocument_0093(t *testing.T) {
	dir := filepath.Join("..", "llmprovider", "catalog", "testdata", "ranking-2026-10-12")
	listing, err := os.ReadFile(filepath.Join(dir, "zen.json"))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := catalog.ParseMetadata(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	var (
		mu    sync.Mutex
		other []string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/models") {
			_, _ = w.Write(listing)
			return
		}
		mu.Lock()
		other = append(other, r.URL.Path)
		mu.Unlock()
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	p := llmprovider.ProviderOpencodeZen

	// menu is the recommended menu the wizard shows, each run in a subtest
	// with a catalog cache of its own.
	menu := func(name string, opts ...llmprovider.Option) []string {
		var got []string
		t.Run(name, func(t *testing.T) {
			f := newFake(t, fakePrompter{blankSearches: true, selects: []int{providerIdx(t, p), 0},
				inputs: []string{srv.URL}, secrets: []string{testKey}})
			if _, err := ConfigureLLM(context.Background(), f, Options{Discover: true, DiscoverLimit: 5 * time.Second,
				ProviderOptions: append([]llmprovider.Option{testCache(t)}, opts...)}); err != nil {
				t.Fatalf("ConfigureLLM: %v", err)
			}
			got = labels(f.seenSelectItems[1])
		})
		return got
	}
	// listed is catalog.List's recommended ids for the same listing, as labels.
	listed := func(opts ...llmprovider.Option) []string {
		t.Helper()
		cat, err := catalog.List(context.Background(), p, llmprovider.NewStaticToken(testKey),
			append([]llmprovider.Option{llmprovider.WithBaseURL(srv.URL), catalog.WithCache(new(catalog.Cache))}, opts...)...)
		if err != nil || cat.Err != nil {
			t.Fatalf("List: %v, %v", err, cat.Err)
		}
		out := make([]string, 0, len(cat.Recommended))
		for _, id := range cat.Recommended {
			out = append(out, catalog.Label(p, id))
		}
		return out
	}

	ranked, fallback := listed(catalog.WithMetadata(doc)), listed()
	if slices.Equal(ranked, fallback) {
		t.Fatalf("the document's ranking %v is the fallback's; the case tells nothing", ranked)
	}
	if got := menu("none"); len(got) < len(fallback) || !slices.Equal(got[:len(fallback)], fallback) {
		t.Errorf("with no document, the menu %v does not start with the fallback curators' %v", got, fallback)
	}
	if got := menu("ranked", catalog.WithMetadata(doc)); len(got) < len(ranked) || !slices.Equal(got[:len(ranked)], ranked) {
		t.Errorf("with the document handed in, the menu %v does not start with its ranking %v", got, ranked)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(other) != 0 {
		t.Errorf("requests other than the listing: %v; want none (no document is fetched)", other)
	}
}
