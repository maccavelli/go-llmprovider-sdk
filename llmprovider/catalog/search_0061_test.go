package catalog

import (
	"context"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0061-PLAN Phase 4b: the search engine (0061-MADR, amendment "Model search,
// rebuilt", S1 to S5, and "S1 amended in execution").

// TestTokenise_0061: words split on the separators and between a letter and
// a digit, a size kept whole, each once, in order.
func TestTokenise_0061(t *testing.T) {
	for _, c := range []struct {
		in   string
		want []string
	}{
		{"gpt4o", []string{"gpt", "4", "o"}},
		{"qwen3:8b", []string{"qwen", "3", "8b"}},
		{"meta-llama/Llama-3.1-8B-Instruct", []string{"meta", "llama", "3", "1", "8b", "instruct"}},
		{"gemma2-27b", []string{"gemma", "2", "27b"}},
		{"128k 1m 2T", []string{"128k", "1m", "2t"}},
		{"8bit", []string{"8", "bit"}},
		{"o3-mini o3", []string{"o", "3", "mini"}},
		{" -_/.:~ ", nil},
	} {
		if got := searchWords(c.in); !slices.Equal(got, c.want) {
			t.Errorf("searchWords(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// osaReference is the optimal string alignment distance, the whole table.
func osaReference(a, b []rune) int {
	d := make([][]int, len(a)+1)
	for i := range d {
		d[i] = make([]int, len(b)+1)
		d[i][0] = i
	}
	for j := range d[0] {
		d[0][j] = j
	}
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			d[i][j] = min(d[i-1][j]+1, d[i][j-1]+1, d[i-1][j-1]+cost)
			if i > 1 && j > 1 && a[i-1] == b[j-2] && a[i-2] == b[j-1] {
				d[i][j] = min(d[i][j], d[i-2][j-2]+1)
			}
		}
	}
	return d[len(a)][len(b)]
}

// TestDistance_0061: the bound by length, a transposition one edit, and
// the early stop never changing an answer: editsWithin agrees with the
// whole table on every pair, at every bound.
func TestDistance_0061(t *testing.T) {
	for n, want := range map[int]int{1: 0, 3: 0, 4: 1, 7: 1, 8: 2, 20: 2} {
		if got := editBound(n); got != want {
			t.Errorf("editBound(%d) = %d, want %d", n, got, want)
		}
	}
	for _, c := range []struct {
		a, b string
		d    int
	}{
		{"sonet", "sonnet", 1}, {"gemnii", "gemini", 1}, {"abcd", "abdc", 1}, {"kitten", "sitting", 3},
		{"ab", "abcd", 2}, {"aaaaaaaa", "bbbbbbbb", 8}, {"qwen", "qwen", 0}, {"ca", "abc", 3},
	} {
		if got := osaReference([]rune(c.a), []rune(c.b)); got != c.d {
			t.Fatalf("osaReference(%q, %q) = %d, want %d", c.a, c.b, got, c.d)
		}
		for bound := range 4 {
			d, ok := editsWithin([]rune(c.a), []rune(c.b), bound)
			if ok != (c.d <= bound) || (ok && d != c.d) {
				t.Errorf("editsWithin(%q, %q, %d) = %d, %v; want %d, %v", c.a, c.b, bound, d, ok, c.d, c.d <= bound)
			}
		}
	}
}

// facts is a ModelInfo with fields known.
func facts(info ModelInfo, fields ...Field) ModelInfo { return info.Known(fields...) }

// bm25Of is d's BM25F score in ix for q.
func bm25Of(ix *searchIndex, id, q string) float64 {
	sq := &searchQuery{words: searchWords(q)}
	sq.near = make([]map[string]float64, len(sq.words))
	for i := range ix.docs {
		if ix.docs[i].match.ID != id {
			continue
		}
		hits := make([]wordHit, len(sq.words))
		for j := range sq.words {
			hits[j] = sq.hit(&ix.docs[i], j)
		}
		return ix.bm25(&ix.docs[i], hits)
	}
	return -1
}

// TestBM25F_0061: a word in the id outweighs it in the family alone; a rare
// word outweighs a common one in the same field.
func TestBM25F_0061(t *testing.T) {
	cat := Catalog{Menu: []MenuEntry{
		{ID: "zeta-common"},
		{ID: "acme-common", Info: facts(ModelInfo{Family: "zeta"}, FieldFamily)},
		{ID: "rare-common"},
	}}
	ix := cat.newIndex()
	if id, family := bm25Of(ix, "zeta-common", "zeta"), bm25Of(ix, "acme-common", "zeta"); id <= family {
		t.Errorf("zeta in the id scores %v, in the family %v; want the id higher", id, family)
	}
	if rare, common := bm25Of(ix, "rare-common", "rare"), bm25Of(ix, "rare-common", "common"); rare <= common {
		t.Errorf("rare scores %v, common %v; want rare higher", rare, common)
	}

	// The context buckets are one word of the facets' length: a 2m window
	// and an 8k one score alike on vision, and the listing's order holds.
	vision := []Modality{ModalityText, ModalityImage}
	windows := Catalog{Menu: []MenuEntry{
		{ID: "big", Info: facts(ModelInfo{Input: vision, ContextWindow: 2_000_000}, FieldInput, FieldContextWindow)},
		{ID: "small", Info: facts(ModelInfo{Input: vision, ContextWindow: 8_000}, FieldInput, FieldContextWindow)},
	}}
	if ids := matchIDs(windows.Search("vision")); !slices.Equal(ids, []string{"big", "small"}) {
		t.Errorf("vision = %v, want big, small: a larger window does not lengthen the facets", ids)
	}
}

// TestSearch_Key_0061: a stronger tier outranks a weaker one whatever
// BM25F says; inside a tier BM25F orders; a fallback never adds a row to a
// query a tier answered.
func TestSearch_Key_0061(t *testing.T) {
	cat := Catalog{Menu: []MenuEntry{
		{ID: "zeta-pro", Info: facts(ModelInfo{Name: "Zeta", Family: "zeta"}, FieldName, FieldFamily)},
		{ID: "zeta"},
	}}
	got := cat.Search("zeta")
	if ids := matchIDs(got); !slices.Equal(ids, []string{"zeta", "zeta-pro"}) {
		t.Fatalf("zeta = %v, want the exact id before the prefix", ids)
	}
	ix := cat.newIndex()
	if exact, prefix := bm25Of(ix, "zeta", "zeta"), bm25Of(ix, "zeta-pro", "zeta"); exact >= prefix {
		t.Fatalf("precondition: the prefix's BM25F (%v) must pass the exact id's (%v)", prefix, exact)
	}
	if got[0].Score != scoreExact || got[0].Why != "exact id" || got[1].Score != scoreIDPrefix || got[1].Why != "id prefix" {
		t.Errorf("zeta = %+v, want 6000 exact id, then 5000 id prefix", got)
	}

	inTier := Catalog{Menu: []MenuEntry{
		{ID: "acme-zeta-1"},
		{ID: "acme-zeta-2", Info: facts(ModelInfo{Family: "zeta"}, FieldFamily)},
	}}
	if ids := matchIDs(inTier.Search("zeta")); !slices.Equal(ids, []string{"acme-zeta-2", "acme-zeta-1"}) {
		t.Errorf("zeta = %v, want the one with the family first, against the listing's order", ids)
	}

	typo := Catalog{Usable: []string{"claude-sonnet-5", "claude-sonet-x"}}
	if ids := matchIDs(typo.Search("sonnet")); !slices.Equal(ids, []string{"claude-sonnet-5"}) {
		t.Errorf("sonnet = %v, want claude-sonnet-5 alone: a near word adds no row to a query a tier answered", ids)
	}
}

// TestSearch_Typo_0061: a word one edit off finds its model, its reason
// naming the word it was near; a size typed short finds by prefix.
func TestSearch_Typo_0061(t *testing.T) {
	got := Search("", matcherFixture, "sonet")
	if ids := matchIDs(got); !slices.Equal(ids, []string{"claude-sonnet-5"}) || got[0].Score != scoreNearAll ||
		got[0].Why != "near sonnet" {
		t.Errorf("sonet = %+v, want claude-sonnet-5 at %d, near sonnet", got, scoreNearAll)
	}
	got = Search("", matcherFixture, "gemnii")
	if ids := matchIDs(got); len(ids) != 3 || !slices.ContainsFunc(ids, func(id string) bool { return id == "gemini-2.5-pro" }) {
		t.Errorf("gemnii = %v, want the three gemini models", ids)
	}
	for _, m := range got {
		if m.Score != scoreNearAll || m.Why != "near gemini" {
			t.Errorf("gemnii: %+v, want %d, near gemini", m, scoreNearAll)
		}
	}
	got = Search("", []string{"llama3:8b", "qwen3:32b", "qwen3:8b"}, "qwen3:8")
	if ids := matchIDs(got); len(ids) == 0 || ids[0] != "qwen3:8b" {
		t.Errorf("qwen3:8 = %v, want qwen3:8b first", ids)
	}
	got = Search("", []string{"alpha-one", "beta-two"}, "alpah gamma")
	if ids := matchIDs(got); !slices.Equal(ids, []string{"alpha-one"}) || got[0].Score != scorePartial+1 ||
		got[0].Why != "some words: alpah" {
		t.Errorf("alpah gamma = %+v, want alpha-one at %d, some words: alpah", got, scorePartial+1)
	}
}

// facetMenu is a menu whose facts say vision, the context window, the
// price and reasoning, and one model whose facts say nothing.
func facetMenu() Catalog {
	day := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	return Catalog{Menu: []MenuEntry{
		{ID: "claude-sonnet-5", Info: facts(ModelInfo{Family: "claude", Input: []Modality{ModalityText, ModalityImage},
			ContextWindow: 200_000, Released: day}, FieldFamily, FieldInput, FieldContextWindow, FieldReleased)},
		{ID: "claude-haiku-5", Info: facts(ModelInfo{Family: "claude", Input: []Modality{ModalityText},
			ContextWindow: 200_000}, FieldFamily, FieldInput, FieldContextWindow)},
		{ID: "gpt-x", Info: facts(ModelInfo{Input: []Modality{ModalityImage}, ContextWindow: 128_000},
			FieldInput, FieldContextWindow)},
		{ID: "zero-cost", Info: facts(ModelInfo{}, FieldCostInput, FieldCostOutput)},
		{ID: "thinker", Info: facts(ModelInfo{Reasoning: ModelReasoning{Supported: true}}, FieldReasoning)},
		{ID: "silent", Info: ModelInfo{Input: []Modality{ModalityImage}, ContextWindow: 1_000_000, Tools: true,
			Reasoning: ModelReasoning{Supported: true}}},
	}}
}

// TestSearch_Facets_0061: a model's facts are words, each only where the
// facts state it; the reason names the facets matched.
func TestSearch_Facets_0061(t *testing.T) {
	cat := facetMenu()
	for _, c := range []struct {
		q, why string
		want   []string
	}{
		{"claude vision 200k", "every word: vision, 200k", []string{"claude-sonnet-5"}},
		{"vision", "every word: vision", []string{"claude-sonnet-5", "gpt-x"}},
		{"128k", "every word: 128k", []string{"claude-sonnet-5", "claude-haiku-5", "gpt-x"}},
		{"free", "every word: free", []string{"zero-cost"}},
		{"reasoning", "every word: reasoning", []string{"thinker"}},
		{"thinking", "every word: thinking", []string{"thinker"}},
		{"claude 2026", "every word: 2026", []string{"claude-sonnet-5"}},
		{"1m", "", nil},
		{"tools", "", nil},
	} {
		got := cat.Search(c.q)
		// The order is BM25F's, TestBM25F_0061's; here, which models.
		if ids := slices.Sorted(slices.Values(matchIDs(got))); !slices.Equal(ids, slices.Sorted(slices.Values(c.want))) {
			t.Errorf("%q = %v, want %v", c.q, ids, c.want)
			continue
		}
		for _, m := range got {
			if m.Score != scoreAllWords || m.Why != c.why {
				t.Errorf("%q: %+v, want %d, %q", c.q, m, scoreAllWords, c.why)
			}
		}
	}
}

// TestCatalogSearch_Usable_0061: Catalog.Search searches Usable, in its
// order, with Menu's facts by id; a model WithLimit cut from Menu is found
// by its id; Menu's ids when Usable is empty (0061-MADR, "S1 amended in
// execution").
func TestCatalogSearch_Usable_0061(t *testing.T) {
	cat := Catalog{
		Usable: []string{"m-3", "M-1", "m-2"},
		Menu:   []MenuEntry{{ID: "m-1", Info: facts(ModelInfo{Input: []Modality{ModalityImage}}, FieldInput)}},
	}
	if ids := matchIDs(cat.Search("m")); !slices.Equal(ids, []string{"m-3", "M-1", "m-2"}) {
		t.Errorf("m = %v, want Usable's order", ids)
	}
	if ids := matchIDs(cat.Search("vision")); !slices.Equal(ids, []string{"M-1"}) {
		t.Errorf("vision = %v, want M-1, by Menu's facts of m-1", ids)
	}
	cat.Usable = nil
	if ids := matchIDs(cat.Search("m")); !slices.Equal(ids, []string{"m-1"}) {
		t.Errorf("m with no Usable = %v, want Menu's m-1", ids)
	}
}

// TestCatalogSearch_Index_0061: List builds the index once, with its
// provider's labels; Catalog.Search reads it and allocates no index per
// query; a Catalog whose Usable was replaced is indexed again.
func TestCatalogSearch_Index_0061(t *testing.T) {
	useSteadyClock(t)
	s := newSteadyServer(t, 0, steadyReply{status: http.StatusOK, body: compatListing})
	cat, err := compatList(context.Background(), s, &Cache{}, "k")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if cat.index == nil || !cat.index.indexes(cat) || cat.provider != "acme" {
		t.Fatalf("List's index = %v, provider %q; want an index of its Usable, acme", cat.index, cat.provider)
	}
	if got := cat.Search("m3"); len(got) != 1 || got[0].ID != "m3" || got[0].Why != "exact id" {
		t.Errorf("m3 = %+v, want m3, exact id", got)
	}
	other := cat
	other.Usable = []string{"z9"}
	if ids := matchIDs(other.Search("z9")); !slices.Equal(ids, []string{"z9"}) {
		t.Errorf("z9 on a replaced Usable = %v, want z9", ids)
	}

	big := Catalog{Usable: benchIDs0061()}
	big.index = big.newIndex()
	if allocs := testing.AllocsPerRun(20, func() { _ = big.Search("sonnet") }); allocs > 300 {
		t.Errorf("Catalog.Search on a built index: %v allocations a query, want at most 300", allocs)
	}
}

// TestSearch_ExistingTiers_0061: each of today's tiers keeps its score, and
// says why; the 21 Search tests before this phase pass unchanged but
// TestSearchModels_Subsequence's query (0061-PLAN Phase 4b, step 1).
func TestSearch_ExistingTiers_0061(t *testing.T) {
	withLabel(t, llmprovider.ProviderClaude, "acme/m1", "Roadrunner Large [balanced speed]")
	ids := append(slices.Clone(matcherFixture), "acme/m1")
	for _, c := range []struct {
		q, id, why string
		score      int
	}{
		{"gpt-5.4", "gpt-5.4", "exact id", scoreExact},
		{"gemini-2", "gemini-2.5-pro", "id prefix", scoreIDPrefix},
		{"flash-lite", "gemini-3.5-flash-lite", "in the id", scoreIDSubstring},
		{"gpt41", "gpt-4.1-mini", "in the id, without separators", scoreCompactSubstring},
		{"roadrunner", "acme/m1", "in the name", scoreLabelSubstring},
		{"mini 4.1", "gpt-4.1-mini", "every word", scoreTokenPrefix},
		{"cdsnt", "claude-sonnet-5", "letters in order", scoreSubsequence + 10},
	} {
		got := Search(llmprovider.ProviderClaude, ids, c.q)
		if len(got) == 0 || got[0].ID != c.id || got[0].Score != c.score || got[0].Why != c.why {
			t.Errorf("%q = %+v, want %s first at %d, %q", c.q, got, c.id, c.score, c.why)
		}
	}
}

// TestSearch_Glob_Why_0061: a glob's matches say whether the id or the name
// matched.
func TestSearch_Glob_Why_0061(t *testing.T) {
	got := Search(llmprovider.ProviderClaude, []string{"claude-haiku-5-5", "claude-sonnet-5"}, "claude-s*")
	if len(got) != 1 || got[0].Why != "id pattern" {
		t.Errorf("claude-s* = %+v, want claude-sonnet-5, id pattern", got)
	}
}
