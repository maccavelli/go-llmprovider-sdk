package wizard

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// 0061-PLAN Phase 4b: the wizard's search (0061-MADR, amendment "Model
// search, rebuilt", S4, and "S4 amended in execution").

// pageIDs are n ids, m-01 to m-n.
func pageIDs(n int) []string {
	ids := make([]string, 0, n)
	for i := 1; i <= n; i++ {
		ids = append(ids, fmt.Sprintf("m-%02d", i))
	}
	return ids
}

// TestSearch_Paging_0061: ten rows a page, More while any are unshown, the
// first row the default on every page; an exact id is the first row; ten
// matches show no More; Search again and Other.
func TestSearch_Paging_0061(t *testing.T) {
	ctx := context.Background()
	ids := pageIDs(25)
	matches := catalog.Search("", ids, "m-")
	f := newFake(t, fakePrompter{selects: []int{10, 10, 2}})
	id, other, err := pageMatches(ctx, f, "t", matches, nil)
	if err != nil || other || id != "m-23" {
		t.Fatalf("pageMatches = %q, %v, %v; want m-23", id, other, err)
	}
	if !slices.Equal(f.seenSelectDefault, []int{0, 0, 0}) {
		t.Errorf("defaults = %v, want the first row on each page", f.seenSelectDefault)
	}
	if got := labels(f.seenSelectItems[2]); !slices.Equal(got, append(slices.Clone(ids[20:]), searchAgainLabel, otherModelLabel)) {
		t.Errorf("the last page = %v, want m-21 to m-25, Search again, Other", got)
	}

	exact := catalog.Search("", ids, "m-12")
	f = newFake(t, fakePrompter{selects: []int{0}})
	if id, _, err := pageMatches(ctx, f, "t", exact, nil); err != nil || id != "m-12" {
		t.Errorf("the default for an exact id = %q, %v; want m-12", id, err)
	}

	ten := catalog.Search("", ids[:10], "m-")
	f = newFake(t, fakePrompter{selects: []int{10}})
	if id, other, err := pageMatches(ctx, f, "t", ten, nil); err != nil || id != "" || other {
		t.Errorf("Search again = %q, %v, %v; want no id", id, other, err)
	}
	if got := labels(f.seenSelectItems[0]); slices.ContainsFunc(got, func(l string) bool { return strings.HasPrefix(l, "More") }) {
		t.Errorf("ten matches = %v, want no More", got)
	}
	f = newFake(t, fakePrompter{selects: []int{11}})
	if _, other, err := pageMatches(ctx, f, "t", ten, nil); err != nil || !other {
		t.Errorf("Other = %v, %v; want other", other, err)
	}
}

// TestSearch_Detail_0061: a row's Detail is the facts known, in one order,
// then why it matched; a short token count; Catalog.Search, facts and all,
// for a listing List built, and catalog.Search's ids for one it did not;
// the fallbacks' search leaves out the excluded.
func TestSearch_Detail_0061(t *testing.T) {
	info := catalog.ModelInfo{ContextWindow: 200_000, Input: []catalog.Modality{catalog.ModalityText, catalog.ModalityImage,
		catalog.ModalityPDF}, Tools: true, Reasoning: catalog.ModelReasoning{Supported: true}}
	info.Cost.Input, info.Cost.Output = 3, 15
	info = info.Known(catalog.FieldContextWindow, catalog.FieldCostInput, catalog.FieldCostOutput, catalog.FieldInput,
		catalog.FieldTools, catalog.FieldReasoning)
	if got, want := matchDetail(info, "near sonnet"), "200k ctx · $3 / $15 per M · vision · pdf · tools · reasoning · near sonnet"; got != want {
		t.Errorf("matchDetail = %q, want %q", got, want)
	}
	free := catalog.ModelInfo{}.Known(catalog.FieldCostInput, catalog.FieldCostOutput)
	if got := matchDetail(free, ""); got != "free" {
		t.Errorf("matchDetail(free) = %q, want free", got)
	}
	silent := catalog.ModelInfo{ContextWindow: 8_000, Tools: true}
	if got := matchDetail(silent, "id prefix"); got != "id prefix" {
		t.Errorf("matchDetail(no facts known) = %q, want the reason alone", got)
	}
	for n, want := range map[int]string{131_072: "128k", 1_048_576: "1m", 2_000_000: "2m", 200_000: "200k", 32_768: "32k",
		999: "999", 1_050_000: "1050k"} {
		if got := tokenCount(n); got != want {
			t.Errorf("tokenCount(%d) = %q, want %q", n, got, want)
		}
	}

	d := llmprovider.Descriptor{ID: "acme", Label: "Acme"}
	menu := []catalog.MenuEntry{{ID: "seer", Info: catalog.ModelInfo{Input: []catalog.Modality{catalog.ModalityImage}}.Known(catalog.FieldInput)}}
	listed := catalog.Catalog{Usable: []string{"plain", "seer"}, Menu: menu, Source: catalog.ListLive}
	if got := searchCatalog(d, listed, nil, "vision"); len(got) != 1 || got[0].ID != "seer" {
		t.Errorf("vision on a listing List built = %+v, want seer", got)
	}
	if got := searchCatalog(d, listed, map[string]struct{}{"seer": {}}, "vision"); len(got) != 0 {
		t.Errorf("vision less seer = %+v, want none", got)
	}
	unlisted := listed
	unlisted.Source = ""
	if got := searchCatalog(d, unlisted, nil, "vision"); len(got) != 0 {
		t.Errorf("vision on a catalog List did not build = %+v, want none: its ids alone", got)
	}
	rows := matchChoices(searchCatalog(d, listed, nil, "seer"), menuFacts(listed))
	if len(rows) != 1 || rows[0].Detail != "vision · exact id" {
		t.Errorf("rows = %+v, want seer, vision · exact id", rows)
	}
}

// TestRenderChoices_Aligned_0061: the details are a column, the numbers
// right-aligned and the labels padded to the longest with a detail; a
// label past maxLabelPad is not padded and pads no other.
func TestRenderChoices_Aligned_0061(t *testing.T) {
	var out bytes.Buffer
	p := &TextPrompter{Out: &out}
	choices := []Choice{{Label: "a", Detail: "x"}, {Label: "abcd", Detail: "y"}, {Label: "plain"}}
	for range 8 {
		choices = append(choices, Choice{Label: "z"})
	}
	choices = append(choices, Choice{Label: strings.Repeat("w", maxLabelPad+1), Detail: "long"})
	p.renderChoices("T", choices, []int{1})
	lines := strings.Split(out.String(), "\n")
	for i, want := range map[int]string{
		2:  "   1) a    — x",
		3:  "   2) abcd — y (selected)",
		4:  "   3) plain",
		13: "  12) " + strings.Repeat("w", maxLabelPad+1) + " — long",
	} {
		if lines[i] != want {
			t.Errorf("line %d = %q, want %q", i, lines[i], want)
		}
	}
}
