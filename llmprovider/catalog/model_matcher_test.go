package catalog

import (
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// matcherFixture is the shared Search corpus. Expected results below were
// derived from a reference implementation of MADR 0007 §3 before this code was
// written (0007-PLAN Appendix C).
var matcherFixture = []string{
	"gemini-3.5-flash", "gemini-3.5-flash-lite", "gemini-2.5-pro",
	"claude-sonnet-5", "claude-haiku-4-5",
	"meta-llama/Llama-3.1-8B-Instruct", "meta-llama/kilo-auto",
	"kilo-auto/balanced", "kilo-auto/free", "kilo-auto/fast", "kilo-auto-legacy",
	"gpt-4.1-mini", "chatgpt-4o-latest", "gpt-5.4",
	"llama3:latest", "fireworks/llama-ash",
}

func matchIDs(ms []Match) []string {
	out := make([]string, 0, len(ms))
	for _, m := range ms {
		out = append(out, m.ID)
	}
	return out
}

func matchScores(ms []Match) []int {
	out := make([]int, 0, len(ms))
	for _, m := range ms {
		out = append(out, m.Score)
	}
	return out
}

func assertIDs(t *testing.T, query string, got []Match, want []string) {
	t.Helper()
	if ids := matchIDs(got); !slices.Equal(ids, want) {
		t.Errorf("SearchModels(%q) ids = %v, want %v", query, ids, want)
	}
}

func TestSearchModels_GlobCrossesSlash(t *testing.T) {
	got := Search("", matcherFixture, "kilo-auto/*")
	assertIDs(t, "kilo-auto/*", got, []string{"kilo-auto/balanced", "kilo-auto/free", "kilo-auto/fast"})
}

func TestSearchModels_GlobMatchesAcrossOrg(t *testing.T) {
	got := Search("", matcherFixture, "*llama*")
	if !slices.Contains(matchIDs(got), "meta-llama/Llama-3.1-8B-Instruct") {
		t.Errorf("*llama* = %v, want it to contain meta-llama/Llama-3.1-8B-Instruct", matchIDs(got))
	}
}

func TestSearchModels_GlobIsAnchored(t *testing.T) {
	got := Search("", matcherFixture, "gpt-4*")
	assertIDs(t, "gpt-4*", got, []string{"gpt-4.1-mini"})
}

func TestSearchModels_GlobKeepsInputOrder(t *testing.T) {
	got := Search("", matcherFixture, "*")
	assertIDs(t, "*", got, matcherFixture)
	for _, m := range got {
		if m.Score != scoreGlobID { // every id matches "*" (0021-MADR C11)
			t.Errorf("%s score = %d, want %d", m.ID, m.Score, scoreGlobID)
		}
	}
}

// TestSearchModels_SubstringOutranksSubsequence: once an id matches a
// stronger tier, no subsequence is offered (0021-MADR C11), so
// fireworks/llama-ash, a subsequence of flash, is not.
func TestSearchModels_SubstringOutranksSubsequence(t *testing.T) {
	got := Search("", matcherFixture, "flash")
	assertIDs(t, "flash", got, []string{"gemini-3.5-flash", "gemini-3.5-flash-lite"})
	if scores := matchScores(got); !slices.Equal(scores, []int{4000, 4000}) {
		t.Errorf("scores = %v, want [4000 4000]", scores)
	}
}

func TestSearchModels_TokenPrefix(t *testing.T) {
	for query, want := range map[string]string{
		"llama 8b":      "meta-llama/Llama-3.1-8B-Instruct",
		"llama3 latest": "llama3:latest",
	} {
		got := Search("", matcherFixture, query)
		assertIDs(t, query, got, []string{want})
		if scores := matchScores(got); !slices.Equal(scores, []int{scoreTokenPrefix}) {
			t.Errorf("%q scores = %v, want [%d]", query, scores, scoreTokenPrefix)
		}
	}
}

func TestSearchModels_Subsequence(t *testing.T) {
	got := Search("", matcherFixture, "sonet")
	assertIDs(t, "sonet", got, []string{"claude-sonnet-5"})
	if scores := matchScores(got); !slices.Equal(scores, []int{1035}) {
		t.Errorf("scores = %v, want [1035]", scores)
	}
}

func TestSearchModels_NoSubsequenceOverLabels(t *testing.T) {
	// hspd is an in-order subsequence of the haiku label ("…high speed…") but of
	// nothing in any id, so it must match nothing.
	label := strings.ToLower(Label("", "claude-haiku-4-5"))
	if _, ok := subsequenceBonus(label, "hspd"); !ok {
		t.Fatalf("precondition: hspd must be a subsequence of the label %q", label)
	}
	if got := Search("", matcherFixture, "hspd"); len(got) != 0 {
		t.Errorf("hspd = %v, want no matches", matchIDs(got))
	}
}

func TestSearchModels_EmptyQuery(t *testing.T) {
	for _, q := range []string{"", "   "} {
		if got := Search("", matcherFixture, q); got != nil {
			t.Errorf("SearchModels(%q) = %v, want nil", q, got)
		}
	}
}

// TestSearchModels_TieBreak: equal scores keep the input order, the
// listing's own (0021-MADR C11).
func TestSearchModels_TieBreak(t *testing.T) {
	got := Search("", matcherFixture, "auto")
	assertIDs(t, "auto", got, []string{
		"meta-llama/kilo-auto", "kilo-auto/balanced", "kilo-auto/free", "kilo-auto/fast", "kilo-auto-legacy",
	})
}

func TestSearchModels_Tiers(t *testing.T) {
	got := Search("", []string{"xgpt-5.4", "gpt-5.4-mini", "gpt-5.4"}, "gpt-5.4")
	assertIDs(t, "gpt-5.4", got, []string{"gpt-5.4", "gpt-5.4-mini", "xgpt-5.4"})
	if scores := matchScores(got); !slices.Equal(scores, []int{scoreExact, scoreIDPrefix, scoreIDSubstring}) {
		t.Errorf("scores = %v, want exact, prefix, substring", scores)
	}
}

func TestSearchModels_Dedupe(t *testing.T) {
	got := Search("", []string{"a-b", "A-B", "a-b"}, "a")
	assertIDs(t, "a", got, []string{"a-b"})
}

func TestSearchModels_LabelFromModelLabel(t *testing.T) {
	got := Search("", []string{"claude-haiku-4-5"}, "haiku")
	if len(got) != 1 || got[0].Label != Label("", "claude-haiku-4-5") {
		t.Errorf("got %+v, want one match labelled %q", got, Label("", "claude-haiku-4-5"))
	}
}

func TestSearchModels_SeparatorOnlyQuery(t *testing.T) {
	got := Search("", []string{"ab", "a-b"}, "-")
	assertIDs(t, "-", got, []string{"a-b"})
}

// withLabel adds a curated label for one test.
func withLabel(t *testing.T, id, label string) {
	t.Helper()
	modelLabels[id] = label
	t.Cleanup(func() { delete(modelLabels, id) })
}

// TestSearch_AnnotationNotSearched (0021-MADR C11): "fast" is only in the
// bracketed annotations of OpenAI's labels, so it matches no OpenAI model.
func TestSearch_AnnotationNotSearched(t *testing.T) {
	if got := Search(llmprovider.ProviderOpenAI, Static(llmprovider.ProviderOpenAI), "fast"); len(got) != 0 {
		t.Errorf(`"fast" = %v, want no match: it is only in the annotations`, matchIDs(got))
	}
}

// TestSearch_CompactQuery (0021-MADR C11): a query without separators finds
// the id with them.
func TestSearch_CompactQuery(t *testing.T) {
	got := Search(llmprovider.ProviderOpenAI, Static(llmprovider.ProviderOpenAI), "gpt41")
	i := slices.IndexFunc(got, func(m Match) bool { return m.ID == "gpt-4.1" })
	if i < 0 || got[i].Score != scoreCompactSubstring {
		t.Errorf(`"gpt41" = %v, want gpt-4.1 at scoreCompactSubstring`, got)
	}
}

// TestSearch_ShortQueryNoSubsequence (0021-MADR C11): "o3" returns the ids
// that hold it, not every id with an o and then a 3.
func TestSearch_ShortQueryNoSubsequence(t *testing.T) {
	got := Search("", []string{"gpt-4o-2024-08-13", "o3", "o3-mini", "gpt-4.1", "openai/o3-pro"}, "o3")
	assertIDs(t, "o3", got, []string{"o3", "o3-mini", "openai/o3-pro"})
}

// TestSearch_TiesKeepListingOrder (0021-MADR C11): equal scores keep the
// input order, not the shorter id first.
func TestSearch_TiesKeepListingOrder(t *testing.T) {
	models := []string{"zeta-flash", "alpha-flash", "mid-flash"}
	assertIDs(t, "flash", Search("", models, "flash"), models)
}

// TestSearch_GlobIDBeforeLabel (0021-MADR C11): a glob that matches an id
// ranks it before one that matches only a label.
func TestSearch_GlobIDBeforeLabel(t *testing.T) {
	withLabel(t, "vendor/mirror", "kilo-auto/mirror")
	got := Search("", []string{"vendor/mirror", "kilo-auto/small"}, "kilo-auto/*")
	assertIDs(t, "kilo-auto/*", got, []string{"kilo-auto/small", "vendor/mirror"})
	if scores := matchScores(got); !slices.Equal(scores, []int{scoreGlobID, scoreGlobLabel}) {
		t.Errorf("scores = %v, want [%d %d]", scores, scoreGlobID, scoreGlobLabel)
	}
}
