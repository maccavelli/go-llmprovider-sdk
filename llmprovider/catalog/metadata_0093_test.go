package catalog

// 0093-MADR's rows on the document a call hands in with WithMetadata (MH2 to
// MH15, MH52, MH53). Each names WithMetadata or Limits.RouteFallback, so
// each takes its red from its plant (0093-PLAN Phase 3).

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// doc0093 parses body, failing the test.
func doc0093(t *testing.T, body string) Metadata {
	t.Helper()
	m, err := ParseMetadata(strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// opusDoc0093 states claude-opus-4-6's window and output as a models.dev
// document.
func opusDoc0093(output string) string {
	return `{"anthropic":{"models":{"claude-opus-4-6":{"name":"Opus","limit":{"context":1000000,"output":` + output + `}}}}}`
}

// TestWithMetadata_Scope_0093 (MH2; D2): a call reads its own options'
// document; another call on the same cache, given none, reads none: there
// is no process-wide or cache-held document.
func TestWithMetadata_Scope_0093(t *testing.T) {
	_, c := limitsCache(t)
	ctx := context.Background()
	with := CachedLimits(ctx, llmprovider.ProviderClaude, "claude-opus-4-6", WithMetadata(doc0093(t, opusDoc0093("64000"))), c)
	wantFact(t, "the call given the document", with.Output, 64000, "models.dev")
	without := CachedLimits(ctx, llmprovider.ProviderClaude, "claude-opus-4-6", c)
	wantUnknown(t, "a later call on the same cache, given none", without.Output)
	if info, ok := CachedFacts(ctx, llmprovider.ProviderClaude, "claude-opus-4-6", c); ok {
		t.Errorf("CachedFacts given no document = %+v; want nothing known", info)
	}
	if info, ok := CachedFacts(ctx, llmprovider.ProviderClaude, "claude-opus-4-6"); ok {
		t.Errorf("CachedFacts on the process's cache, given no document = %+v; want nothing known", info)
	}
}

// TestWithMetadata_LastWins_0093 (MH3; D2, Q1): given twice, the last wins;
// a zero Metadata given last drops the document.
func TestWithMetadata_LastWins_0093(t *testing.T) {
	ctx := context.Background()
	first, second := WithMetadata(doc0093(t, opusDoc0093("64000"))), WithMetadata(doc0093(t, opusDoc0093("32000")))
	l := CachedLimits(ctx, llmprovider.ProviderClaude, "claude-opus-4-6", first, second, testCache(t))
	wantFact(t, "the second document", l.Output, 32000, "models.dev")
	l = CachedLimits(ctx, llmprovider.ProviderClaude, "claude-opus-4-6", first, WithMetadata(Metadata{}), testCache(t))
	wantUnknown(t, "a zero Metadata given last", l.Output)
}

// baselineListings0093 are MH4's listings, and the Recommended each gave at
// the record's base with the document off and the extract emptied
// (0093-PLAN Phase 0, step 4).
var baselineListings0093 = []struct {
	id      llmprovider.ProviderID
	listing string
	want    []string
}{
	{llmprovider.ProviderClaude,
		`{"data":[{"id":"claude-sonnet-5","type":"model"},{"id":"claude-haiku-4-5","type":"model"}],"has_more":false}`,
		[]string{"claude-sonnet-5", "claude-haiku-4-5"}},
	{llmprovider.ProviderOpencodeZen,
		`{"object":"list","data":[{"id":"gpt-5.5","object":"model"},{"id":"claude-sonnet-5","object":"model"},` +
			`{"id":"gemini-3.8-flash","object":"model"},{"id":"glm-5.3","object":"model"}]}`,
		[]string{"gemini-3.8-flash", "claude-sonnet-5", "gpt-5.5", "glm-5.3"}},
}

// TestWithMetadata_NoneIsOff_0093 (MH4; D3): no document, a zero Metadata,
// and a document holding none of the sections the SDK reads each give the
// base's behaviour with the document off and the extract emptied, recorded
// in Phase 0: the same rankings, nothing known of the model, no error.
func TestWithMetadata_NoneIsOff_0093(t *testing.T) {
	ctx := context.Background()
	for name, docOpt := range map[string][]llmprovider.Option{
		"none":            nil,
		"zero Metadata":   {WithMetadata(Metadata{})},
		"no section read": {WithMetadata(doc0093(t, `{"deepseek":{"models":{"claude-sonnet-5":{"limit":{"output":1}}}}}`))},
	} {
		for _, b := range baselineListings0093 {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(b.listing))
			}))
			opts := append([]llmprovider.Option{llmprovider.WithBaseURL(srv.URL), llmprovider.WithModelProbes(false),
				WithCache(new(Cache))}, docOpt...)
			cat, err := List(ctx, b.id, llmprovider.NewStaticToken("k"), opts...)
			if err != nil || !slices.Equal(cat.Recommended, b.want) {
				t.Errorf("%s, %s: Recommended %q, err %v; want the baseline %q", name, b.id, cat.Recommended, err, b.want)
			}
			info, err := LookupModel(ctx, b.id, nil, "claude-sonnet-5", opts...)
			if err != nil || info.known != 0 {
				t.Errorf("%s, %s: LookupModel %+v, err %v; want nothing known, no error", name, b.id, info, err)
			}
			l, err := LookupLimits(ctx, b.id, nil, "claude-sonnet-5", opts...)
			if err != nil || l.Output.Known || l.Window.Known || l.Input.Known || !l.Required || l.RouteFallback.Known {
				t.Errorf("%s, %s: LookupLimits %+v, err %v; want nothing known, required, no error", name, b.id, l, err)
			}
			srv.Close()
		}
	}
}

// TestWithMetadata_Swap_0093 (MH5; D4): calls in flight with either of two
// documents, under -race, each wholly its own: its ranking and its menu's
// facts are what a call alone with that document gives.
func TestWithMetadata_Swap_0093(t *testing.T) {
	listing := `{"data":[{"id":"claude-a","type":"model"},{"id":"claude-b","type":"model"}],"has_more":false}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(listing)) }))
	defer srv.Close()
	docs := []Metadata{
		doc0093(t, `{"anthropic":{"models":{"claude-a":{"limit":{"context":1000},"cost":{"input":1,"output":1}},`+
			`"claude-b":{"limit":{"context":2000},"cost":{"input":50,"output":90}}}}}`),
		doc0093(t, `{"anthropic":{"models":{"claude-a":{"limit":{"context":3000},"cost":{"input":50,"output":90}},`+
			`"claude-b":{"limit":{"context":4000},"cost":{"input":1,"output":1}}}}}`),
	}
	ctx := context.Background()
	run := func(i int) Catalog {
		cat, err := List(ctx, llmprovider.ProviderClaude, llmprovider.NewStaticToken("k"), llmprovider.WithBaseURL(srv.URL),
			llmprovider.WithModelProbes(false), WithCache(new(Cache)), WithMetadata(docs[i]))
		if err != nil {
			t.Error(err)
		}
		return cat
	}
	want := []Catalog{run(0), run(1)}
	window := func(cat Catalog) map[string]int {
		out := map[string]int{}
		for _, e := range cat.Menu {
			out[e.ID] = e.Info.ContextWindow
		}
		return out
	}
	var wg sync.WaitGroup
	for n := range 32 {
		wg.Go(func() {
			got := run(n % 2)
			if !slices.Equal(got.Recommended, want[n%2].Recommended) || !mapsEqual0093(window(got), window(want[n%2])) {
				t.Errorf("call %d with document %d: %q %v; want %q %v", n, n%2, got.Recommended, window(got),
					want[n%2].Recommended, window(want[n%2]))
			}
		})
	}
	wg.Wait()
	if window(want[0])["claude-a"] != 1000 || window(want[1])["claude-a"] != 3000 {
		t.Errorf("the menus' windows %v, %v; want each document's", window(want[0]), window(want[1]))
	}
}

func mapsEqual0093(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// TestWithMetadata_MergeRank_0093 (MH6; D5): in ModelInfo the document is
// lowest, then the listing, then the caller's layers; a field only the
// document states is the document's, source "models.dev".
func TestWithMetadata_MergeRank_0093(t *testing.T) {
	resetLookupCaches(t)
	s := newLookupServer(t, lookupDoc, 200, claudeListing0033)
	doc := WithMetadata(doc0093(t, lookupDoc))
	opts := []llmprovider.Option{llmprovider.WithBaseURL(s.URL), doc}
	info, err := LookupModel(context.Background(), llmprovider.ProviderClaude, llmprovider.NewStaticToken("k"), "claude-x", opts...)
	if err != nil {
		t.Fatal(err)
	}
	if info.MaxOutputTokens != 100000 || info.SourceOf(FieldMaxOutputTokens) != listingLayer(llmprovider.ProviderClaude) {
		t.Errorf("output %d from %q; want the listing's 100000 above the document's", info.MaxOutputTokens,
			info.SourceOf(FieldMaxOutputTokens))
	}
	if info.Cost.Input != 3 || info.SourceOf(FieldCostInput) != sourceModelsDev {
		t.Errorf("input price %v from %q; want the document's 3, from models.dev", info.Cost.Input, info.SourceOf(FieldCostInput))
	}
	layer := staticLayer{name: "caller", info: ModelInfo{ID: "claude-x", MaxOutputTokens: 7}.Known(FieldMaxOutputTokens)}
	info, _ = LookupModel(context.Background(), llmprovider.ProviderClaude, llmprovider.NewStaticToken("k"), "claude-x",
		append(opts, WithModelLayer(layer))...)
	if info.MaxOutputTokens != 7 || info.SourceOf(FieldMaxOutputTokens) != "caller" {
		t.Errorf("with a caller layer: output %d from %q; want the caller's 7", info.MaxOutputTokens, info.SourceOf(FieldMaxOutputTokens))
	}
}

// TestWithMetadata_LimitsRank_0093 (MH7; D5): caller layers above the
// service, the service above the registry, within it a "models.dev" layer
// above the document, and a remembered cap below all.
func TestWithMetadata_LimitsRank_0093(t *testing.T) {
	s := claudeServer(t)
	_, c := limitsCache(t)
	key := llmprovider.NewStaticToken("k")
	base := []llmprovider.Option{llmprovider.WithBaseURL(s.URL), c, WithMetadata(doc0093(t, opusDoc0093("50000")))}
	l := lookupLimitsT(t, llmprovider.ProviderClaude, key, "claude-opus-4-6", base...)
	wantFact(t, "the service over the document", l.Output, 128000, "listing:claude")
	if !hasSource(l.Output, "models.dev", 50000) {
		t.Errorf("output = %+v; want the document's 50,000 kept below", l.Output)
	}
	_, c2 := limitsCache(t)
	l = lookupLimitsT(t, llmprovider.ProviderClaude, nil, "claude-opus-4-6", c2, WithMetadata(doc0093(t, opusDoc0093("50000"))),
		WithModelLayer(doc0093(t, opusDoc0093("40000"))))
	wantFact(t, "a models.dev layer over the document", l.Output, 40000, "models.dev")
	l = lookupLimitsT(t, llmprovider.ProviderClaude, nil, "claude-opus-4-6", c2, WithMetadata(doc0093(t, opusDoc0093("50000"))),
		WithModelLayer(outputLayer("models.json", 20000)))
	wantFact(t, "a caller layer over the document", l.Output, 20000, "models.json")
}

// TestWithMetadata_EvidenceTime_0093 (MH8; D5): the document's evidence has
// no Seen, and is never stale: its age is the caller's.
func TestWithMetadata_EvidenceTime_0093(t *testing.T) {
	l := CachedLimits(context.Background(), llmprovider.ProviderClaude, "claude-opus-4-6",
		WithMetadata(doc0093(t, opusDoc0093("64000"))), testCache(t))
	for name, f := range map[string]Fact{"output": l.Output, "window": l.Window} {
		if !f.Known || len(f.Evidence) != 1 || !f.Evidence[0].Seen.IsZero() || f.Evidence[0].Stale {
			t.Errorf("%s = %+v; want one statement, Seen zero, not stale", name, f)
		}
	}
}

// TestWithMetadata_AlsoLayer_0093 (MH9; D5): one document given as
// WithMetadata and as WithModelLayer is read in both places.
func TestWithMetadata_AlsoLayer_0093(t *testing.T) {
	m := doc0093(t, opusDoc0093("64000"))
	l := CachedLimits(context.Background(), llmprovider.ProviderClaude, "claude-opus-4-6", WithMetadata(m), WithModelLayer(m),
		testCache(t))
	if len(l.Output.Evidence) != 2 || l.Output.Evidence[0].Source != "models.dev" || l.Output.Evidence[1].Source != "models.dev" {
		t.Errorf("output = %+v; want two statements, the layer's and the document's", l.Output)
	}
}

// TestListRaw_IgnoresMetadata_0093 (MH10; D2; 0041-MADR D9): ListRaw takes
// the option, and reads nothing of it.
func TestListRaw_IgnoresMetadata_0093(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"id":"claude-x","type":"model"}],"has_more":false}`))
	}))
	defer srv.Close()
	ctx := context.Background()
	bare, err := ListRaw(ctx, llmprovider.ProviderClaude, llmprovider.NewStaticToken("k"), llmprovider.WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	with, err := ListRaw(ctx, llmprovider.ProviderClaude, llmprovider.NewStaticToken("k"), llmprovider.WithBaseURL(srv.URL),
		WithMetadata(doc0093(t, lookupDoc)))
	if err != nil {
		t.Fatalf("ListRaw given WithMetadata: %v", err)
	}
	if len(with.Models) != len(bare.Models) {
		t.Errorf("ListRaw given WithMetadata: %d models; want %d, as without", len(with.Models), len(bare.Models))
	}
}

// TestList_RanksHandedDocument_0093 (MH11; D5): a listing ranks by the
// document handed in: the 2026-10-12 snapshot's OpenCode Zen capable six, as
// the fetched document gave it; with the document dropped, the name
// curator's, which differs (the static list is the utility six).
func TestList_RanksHandedDocument_0093(t *testing.T) {
	dir := "testdata/ranking-2026-10-12"
	want := []string{"mistral-large-4", "gpt-6.1-sol", "claude-sonnet-5-5", "claude-opus-5-5", "gpt-6-sol", "grok-4.7"}
	cat := replaySnapshot(t, dir, llmprovider.ProviderOpencodeZen, ProfileCapable, WithLimit(MaxListed))
	if !slices.Equal(cat.Recommended, want) {
		t.Errorf("with the document: %q; want the snapshot's six %q", cat.Recommended, want)
	}
	byName := replaySnapshot(t, dir, llmprovider.ProviderOpencodeZen, ProfileCapable, WithLimit(MaxListed),
		WithMetadata(Metadata{}))
	if slices.Equal(byName.Recommended, want) {
		t.Errorf("with the document dropped: %q; want the name curator's, not the document's", byName.Recommended)
	}
}

// TestMenu_FactsFromHandedDocument_0093 (MH12; D5): a listing's menu takes a
// model's facts from the document handed in where the listing states none,
// and Search finds them; with none, the menu has none of them.
func TestMenu_FactsFromHandedDocument_0093(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"id":"claude-x","type":"model"}],"has_more":false}`))
	}))
	defer srv.Close()
	list := func(more ...llmprovider.Option) Catalog {
		cat, err := List(context.Background(), llmprovider.ProviderClaude, llmprovider.NewStaticToken("k"),
			append([]llmprovider.Option{llmprovider.WithBaseURL(srv.URL), llmprovider.WithModelProbes(false),
				WithCache(new(Cache))}, more...)...)
		if err != nil {
			t.Fatal(err)
		}
		return cat
	}
	entry := func(cat Catalog) ModelInfo {
		for _, e := range cat.Menu {
			if e.ID == "claude-x" {
				return e.Info
			}
		}
		t.Fatalf("no claude-x in the menu %+v", cat.Menu)
		return ModelInfo{}
	}
	with := entry(list(WithMetadata(doc0093(t, lookupDoc))))
	if with.ContextWindow != 1000000 || with.SourceOf(FieldContextWindow) != sourceModelsDev {
		t.Errorf("menu with the document: window %d from %q; want the document's 1000000", with.ContextWindow,
			with.SourceOf(FieldContextWindow))
	}
	if without := entry(list()); without.Has(FieldContextWindow) {
		t.Errorf("menu with no document: window %d; want none", without.ContextWindow)
	}
}

// TestTogether_ListedOnlyHandedDocument_0093 (MH13; D5): with a document,
// Together ranks only the ids its section lists; without one, curateTogether
// ranks the listing.
func TestTogether_ListedOnlyHandedDocument_0093(t *testing.T) {
	listing := `[{"id":"org/listed","type":"chat"},{"id":"org/unlisted","type":"chat"}]`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(listing)) }))
	defer srv.Close()
	rank := func(more ...llmprovider.Option) int {
		cat, err := List(context.Background(), llmprovider.ProviderTogether, llmprovider.NewStaticToken("k"),
			append([]llmprovider.Option{llmprovider.WithBaseURL(srv.URL), llmprovider.WithModelProbes(false),
				WithCache(new(Cache))}, more...)...)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range cat.Menu {
			if e.ID == "org/unlisted" {
				return e.Rank
			}
		}
		t.Fatalf("no org/unlisted in the menu %+v", cat.Menu)
		return 0
	}
	if r := rank(WithMetadata(doc0093(t, `{"togetherai":{"models":{"org/listed":{"limit":{"context":1000}}}}}`))); r != 0 {
		t.Errorf("with the document: org/unlisted ranked %d; want it outside the band, 0", r)
	}
	if r := rank(); r == 0 {
		t.Errorf("with no document: org/unlisted unranked; want curateTogether to rank it")
	}
}

// TestCompat_CanonicalFromHandedDocument_0093 (MH14; D5): a chatcompat
// server's model takes the document's facts by canonical id; without one,
// the listing's only.
func TestCompat_CanonicalFromHandedDocument_0093(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"claude-opus-4-6","object":"model"}]}`))
	}))
	defer srv.Close()
	opts := []llmprovider.Option{withCompatID("acme-0093"), llmprovider.WithBaseURL(srv.URL + "/v1"), testCache(t)}
	info, _ := LookupModel(context.Background(), "acme-0093", nil, "claude-opus-4-6",
		append(opts, WithMetadata(doc0093(t, opusDoc0093("64000"))))...)
	if info.MaxOutputTokens != 64000 || info.SourceOf(FieldMaxOutputTokens) != sourceCanonical {
		t.Errorf("with the document: output %d from %q; want 64000 by canonical id", info.MaxOutputTokens,
			info.SourceOf(FieldMaxOutputTokens))
	}
	info, _ = LookupModel(context.Background(), "acme-0093", nil, "claude-opus-4-6", opts...)
	if info.Has(FieldMaxOutputTokens) {
		t.Errorf("with no document: output %d; want none", info.MaxOutputTokens)
	}
}

// TestLookupModel_HandedDocument_0093 (MH15; D5, D6): LookupModel, CachedModel
// and CachedFacts read the document handed in as the models.dev layer, with
// no request and no error.
func TestLookupModel_HandedDocument_0093(t *testing.T) {
	ctx := context.Background()
	opts := []llmprovider.Option{WithMetadata(doc0093(t, opusDoc0093("64000"))), testCache(t),
		llmprovider.WithHTTPClient(&http.Client{Transport: refuseAll0093{t}})}
	info, err := LookupModel(ctx, llmprovider.ProviderClaude, nil, "claude-opus-4-6", opts...)
	if err != nil || info.MaxOutputTokens != 64000 || info.SourceOf(FieldMaxOutputTokens) != sourceModelsDev {
		t.Errorf("LookupModel: %+v, %v; want the document's 64000, no error", info, err)
	}
	if info, ok := CachedModel(ctx, llmprovider.ProviderClaude, nil, "claude-opus-4-6", opts...); !ok || info.MaxOutputTokens != 64000 {
		t.Errorf("CachedModel: %+v, %t; want the document's 64000", info, ok)
	}
	if info, ok := CachedFacts(ctx, llmprovider.ProviderClaude, "claude-opus-4-6", opts...); !ok || info.MaxOutputTokens != 64000 {
		t.Errorf("CachedFacts: %+v, %t; want the document's 64000", info, ok)
	}
}

// refuseAll0093 fails the test on any request.
type refuseAll0093 struct{ t *testing.T }

func (r refuseAll0093) RoundTrip(req *http.Request) (*http.Response, error) {
	r.t.Errorf("a request was sent: %s", req.URL)
	return nil, http.ErrHandlerTimeout
}

// TestWithMetadata_OtherSectionsOnly_0093 (MH52; D3): a document with an
// anthropic section only is a document: an OpenCode model whose canonical
// id it lists takes its limits by canonical id; OpenCode's own exact and
// family stages find nothing, and its ranking is the name curator's.
func TestWithMetadata_OtherSectionsOnly_0093(t *testing.T) {
	doc := WithMetadata(doc0093(t, `{"anthropic":{"models":{"claude-sonnet-5":{"limit":{"context":1000000,"output":128000}}}}}`))
	l := CachedLimits(context.Background(), llmprovider.ProviderOpencodeZen, "claude-sonnet-5", doc, testCache(t))
	wantFact(t, "by canonical id", l.Output, 128000, "models.dev (canonical id)")
	if len(l.Output.Evidence) != 1 {
		t.Errorf("output = %+v; want the canonical stage alone", l.Output)
	}
	b := baselineListings0093[1]
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(b.listing)) }))
	defer srv.Close()
	cat, err := List(context.Background(), b.id, llmprovider.NewStaticToken("k"), llmprovider.WithBaseURL(srv.URL),
		llmprovider.WithModelProbes(false), WithCache(new(Cache)), doc)
	if err != nil || !slices.Equal(cat.Recommended, b.want) {
		t.Errorf("Recommended %q, %v; want the name curator's %q", cat.Recommended, err, b.want)
	}
}

// TestLimits_RouteFallbackPerCall_0093 (MH53; D7, Q4): two calls on one
// cache under -race, one given a document and one not: each Limits carries
// its own RouteFallback, the document's for the one, none for the other.
func TestLimits_RouteFallbackPerCall_0093(t *testing.T) {
	_, c := limitsCache(t)
	doc := WithMetadata(doc0093(t, `{"anthropic":{"models":{"claude-a":{"limit":{"output":20000}},"claude-b":{"limit":{"output":30000}}}}}`))
	var wg sync.WaitGroup
	for n := range 32 {
		wg.Go(func() {
			if n%2 == 0 {
				l := CachedLimits(context.Background(), llmprovider.ProviderClaude, "claude-unstated", doc, c)
				wantFact(t, "the document's fallback", l.RouteFallback, 20000, "models.dev")
				return
			}
			l := CachedLimits(context.Background(), llmprovider.ProviderClaude, "claude-unstated", c)
			wantUnknown(t, "no document, no fallback", l.RouteFallback)
		})
	}
	wg.Wait()
}
