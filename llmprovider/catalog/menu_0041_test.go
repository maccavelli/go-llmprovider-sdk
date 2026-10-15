package catalog

import (
	"context"
	"encoding/json"
	"io"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0041-PLAN Phase 4, 0041-MADR D3 and A4: every usable model in the menu,
// the band by rank, then the uncaveated newest first, then by id, then the
// caveated the same way; Recommended is the menu less its caveated entries.

// menuNow is the fixture's now.
var menuNow = time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)

// dated is a fact with a release date, days before menuNow.
func dated(id string, days int) ModelInfo {
	b := newInfo(llmprovider.ProviderTogether, id)
	b.info.Released = menuNow.AddDate(0, 0, -days)
	b.set(FieldReleased)
	return b.done()
}

// undated is a fact with nothing known.
func undated(id string) ModelInfo { return newInfo(llmprovider.ProviderTogether, id).done() }

// deprecatedInfo is a dated fact with a deprecated status.
func deprecatedInfo(id string, days int) ModelInfo {
	b := newInfo(llmprovider.ProviderTogether, id)
	b.info.Released, b.info.Status = menuNow.AddDate(0, 0, -days), StatusDeprecated
	b.set(FieldReleased, FieldStatus)
	return b.done()
}

// menuFixture is a listing, its facts, and a curator that ranks band-1 and
// band-2 first, and would rank old-deprecated first of all were it offered.
func menuFixture() ([]string, map[string]ModelInfo, curator) {
	usable := []string{"z-old", "b-same", "old-deprecated", "m-undated", "a-same", "band-2", "c-undated", "band-1", "new"}
	facts := map[string]ModelInfo{
		"z-old": dated("z-old", 400), "b-same": dated("b-same", 30), "a-same": dated("a-same", 30),
		"old-deprecated": deprecatedInfo("old-deprecated", 10), "m-undated": undated("m-undated"),
		"c-undated": undated("c-undated"), "band-1": dated("band-1", 900), "band-2": dated("band-2", 5),
		"new": dated("new", 1),
	}
	curate := func(usable []string, limit int) []string {
		var out []string
		for _, id := range []string{"old-deprecated", "band-1", "band-2"} {
			if slices.Contains(usable, id) && (limit <= 0 || len(out) < limit) {
				out = append(out, id)
			}
		}
		return out
	}
	return usable, facts, curate
}

func menuIDs(menu []MenuEntry) []string {
	ids := make([]string, len(menu))
	for i, e := range menu {
		ids[i] = e.ID
	}
	return ids
}

// TestMenu_Order_0041: the band by rank, then newest first, then by id, a
// dated entry before an undated one; the caveated last, in the same order.
func TestMenu_Order_0041(t *testing.T) {
	usable, facts, curate := menuFixture()
	cat := menuOf(llmprovider.ProviderTogether, usable, facts, curate, 0, menuNow)
	want := []string{"band-1", "band-2", "new", "a-same", "b-same", "z-old", "c-undated", "m-undated", "old-deprecated"}
	if got := menuIDs(cat.Menu); !slices.Equal(got, want) {
		t.Errorf("menu:\n got  %v\n want %v", got, want)
	}
	if want := want[:len(want)-1]; !slices.Equal(cat.Recommended, want) {
		t.Errorf("Recommended:\n got  %v\n want %v", cat.Recommended, want)
	}
	for i, e := range cat.Menu {
		if want := map[int]int{0: 1, 1: 2}[i]; e.Rank != want {
			t.Errorf("%s: rank %d; want %d", e.ID, e.Rank, want)
		}
	}
	if !slices.Equal(cat.Usable, usable) || !cat.Live {
		t.Errorf("Usable %v, Live %t; want the listing, live", cat.Usable, cat.Live)
	}
	if e := cat.Menu[2]; !e.Info.Has(FieldReleased) || !e.Info.Released.Equal(menuNow.AddDate(0, 0, -1)) {
		t.Errorf("%s: Info %+v; want its facts", e.ID, e.Info)
	}
}

// TestMenu_Deterministic_0041: a shuffled listing gives the same menu.
func TestMenu_Deterministic_0041(t *testing.T) {
	usable, facts, curate := menuFixture()
	want := menuOf(llmprovider.ProviderTogether, usable, facts, curate, 0, menuNow)
	r := rand.New(rand.NewPCG(41, 4))
	for range 100 {
		shuffled := slices.Clone(usable)
		r.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
		got := menuOf(llmprovider.ProviderTogether, shuffled, facts, curate, 0, menuNow)
		if !slices.Equal(menuIDs(got.Menu), menuIDs(want.Menu)) || !slices.Equal(got.Recommended, want.Recommended) {
			t.Fatalf("listing %v:\n menu %v\n want %v", shuffled, menuIDs(got.Menu), menuIDs(want.Menu))
		}
	}
}

// TestMenu_Caveats_0041: each caveat from its fact; a caveated entry is in
// the menu, never in Recommended, and never ranked, even where the curator
// would rank it first.
func TestMenu_Caveats_0041(t *testing.T) {
	info := func(id string, f func(b *infoBuilder)) ModelInfo {
		b := newInfo(llmprovider.ProviderTogether, id)
		f(b)
		return b.done()
	}
	facts := map[string]ModelInfo{
		"ok": dated("ok", 10),
		"deprecated": info("deprecated", func(b *infoBuilder) {
			b.info.Status = StatusDeprecated
			b.set(FieldStatus)
		}),
		"retired": info("retired", func(b *infoBuilder) {
			b.info.Status = StatusRetired
			b.set(FieldStatus)
		}),
		"beta": info("beta", func(b *infoBuilder) {
			b.info.Status = StatusBeta
			b.set(FieldStatus)
		}),
		"notools":   info("notools", func(b *infoBuilder) { b.set(FieldTools) }),
		"tools":     info("tools", func(b *infoBuilder) { b.info.Tools = true; b.set(FieldTools) }),
		"imageonly": info("imageonly", func(b *infoBuilder) { b.info.Output = []Modality{ModalityImage}; b.set(FieldOutput) }),
		"textout":   info("textout", func(b *infoBuilder) { b.info.Output = []Modality{ModalityText}; b.set(FieldOutput) }),
		"expiring": info("expiring", func(b *infoBuilder) {
			b.info.Retires = menuNow.AddDate(0, 0, 30)
			b.set(FieldRetires)
		}),
		"later": info("later", func(b *infoBuilder) {
			b.info.Retires = menuNow.AddDate(0, 0, 31)
			b.set(FieldRetires)
		}),
		"x-contributor": undated("x-contributor"),
	}
	want := map[string][]Caveat{
		"deprecated": {CaveatDeprecated}, "retired": {CaveatDeprecated}, "notools": {CaveatNoTools},
		"imageonly": {CaveatNoTextOutput}, "expiring": {CaveatExpiring}, "x-contributor": {CaveatUnavailable},
	}
	usable := slices.Sorted(func(yield func(string) bool) {
		for id := range facts {
			if !yield(id) {
				return
			}
		}
	})
	everyFirst := func(usable []string, _ int) []string { return slices.Clone(usable) }
	cat := menuOf(llmprovider.ProviderTogether, usable, facts, everyFirst, 0, menuNow)
	if len(cat.Menu) != len(usable) {
		t.Fatalf("menu %v; want every usable id", menuIDs(cat.Menu))
	}
	for _, e := range cat.Menu {
		if !slices.Equal(e.Caveats, want[e.ID]) {
			t.Errorf("%s: caveats %v; want %v", e.ID, e.Caveats, want[e.ID])
		}
		if caveated := len(want[e.ID]) > 0; caveated == slices.Contains(cat.Recommended, e.ID) || caveated && e.Rank != 0 {
			t.Errorf("%s: in Recommended %t, rank %d; want %t", e.ID, slices.Contains(cat.Recommended, e.ID), e.Rank, !caveated)
		}
	}
}

// TestMenu_Snapshot_0041: on the 2026-10-12 snapshot, each ranked
// provider's menu is exactly Usable, Recommended is its uncaveated prefix,
// and each full list is the length measured, the profile's exclusions and
// Together's unlisted ids in it (A4).
func TestMenu_Snapshot_0041(t *testing.T) {
	dir := filepath.Join("testdata", "ranking-2026-10-12")
	for provider := range snapshotFile {
		for _, p := range []Profile{ProfileUtility, ProfileCapable} {
			cat := replaySnapshot(t, dir, provider, p)
			ids := menuIDs(cat.Menu)
			if !sameSet(ids, cat.Usable) {
				t.Errorf("%s profile %d: the menu holds %d ids; want Usable's %d", provider, p, len(ids), len(cat.Usable))
			}
			var uncaveated []string
			for _, e := range cat.Menu {
				if len(e.Caveats) == 0 {
					uncaveated = append(uncaveated, e.ID)
				}
			}
			if !slices.Equal(cat.Recommended, uncaveated) || !slices.Equal(ids[:len(uncaveated)], uncaveated) {
				t.Errorf("%s profile %d: Recommended is not the menu's uncaveated prefix", provider, p)
			}
			if want := snapshotMenu[provider][p]; len(cat.Recommended) != want {
				t.Errorf("%s profile %d: the full list holds %d ids; want %d, as measured", provider, p, len(cat.Recommended), want)
			}
		}
	}
	// OpenAI's section, as the evidence test reads it: no deprecated model
	// in the full list (§4's 11 become 0), and gpt-3.5-turbo, which takes no
	// tools, in the menu with its caveat.
	doc := evidence0041Doc(t)
	models, _ := doc.models(modelMetadataKey(llmprovider.ProviderOpenAI))
	var ids []string
	for id := range models {
		if isUsableOpenAIChatModel(id) {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	meta := func() modelMetadataResult { return modelMetadataResult{doc: doc} }
	cfg := config{}
	cat := menuOf(llmprovider.ProviderOpenAI, ids, factsOf(llmprovider.ProviderOpenAI, cfg, ids, nil, doc),
		metadataCurate(llmprovider.ProviderOpenAI, ProfileUtility, defaultWorkload, meta, curateOpenAI), 0, rankingNow())
	deprecated := 0
	for _, id := range cat.Recommended {
		if models[id].Status == "deprecated" {
			deprecated++
		}
	}
	turbo := slices.IndexFunc(cat.Menu, func(e MenuEntry) bool { return e.ID == "gpt-3.5-turbo" })
	if deprecated != 0 || turbo < 0 || !slices.Contains(cat.Menu[turbo].Caveats, CaveatNoTools) ||
		slices.Contains(cat.Recommended, "gpt-3.5-turbo") {
		t.Errorf("OpenAI's section: %d deprecated in the full list, gpt-3.5-turbo at %d; want 0, and it caveated", deprecated, turbo)
	}
}

// snapshotMenu is each full list's length on the 2026-10-12 snapshot, by
// profile, measured in 0041-PLAN Phase 4.
var snapshotMenu = map[llmprovider.ProviderID][2]int{
	llmprovider.ProviderHuggingFace: {118, 118}, llmprovider.ProviderKilo: {276, 276},
	llmprovider.ProviderOpencodeGo: {25, 25}, llmprovider.ProviderOpencodeZen: {77, 77},
	llmprovider.ProviderTogether: {166, 166},
}

// TestRecommended_Prefix_0041: WithLimit(n) is a prefix of the unlimited
// list for n = 1 to 8, and of the menu; WithLimit(MaxListed) is the band.
func TestRecommended_Prefix_0041(t *testing.T) {
	dir := filepath.Join("testdata", "ranking-2026-10-12")
	for provider := range snapshotFile {
		full := replaySnapshot(t, dir, provider, ProfileUtility)
		for n := 1; n <= 8; n++ {
			cut := replaySnapshot(t, dir, provider, ProfileUtility, WithLimit(n))
			if !slices.Equal(cut.Recommended, full.Recommended[:min(n, len(full.Recommended))]) ||
				!slices.Equal(menuIDs(cut.Menu), menuIDs(full.Menu)[:min(n, len(full.Menu))]) {
				t.Errorf("%s WithLimit(%d): %v; want the full list's first %d", provider, n, cut.Recommended, n)
			}
		}
		six := replaySnapshot(t, dir, provider, ProfileUtility, WithLimit(MaxListed))
		for i, e := range six.Menu {
			if e.Rank != i+1 {
				t.Errorf("%s WithLimit(MaxListed): %s ranked %d; want the band, %d", provider, e.ID, e.Rank, i+1)
			}
		}
	}
}

// TestMenu_Static_0041: a failed listing's menu is the static list, in its
// order, each ranked by its place, and Recommended still equals Usable.
func TestMenu_Static_0041(t *testing.T) {
	srv := httptestStatus(t, 500)
	cat := listCatalog(context.Background(), t, llmprovider.ProviderOpenAI, llmprovider.WithBaseURL(srv))
	if cat.Live || !slices.Equal(menuIDs(cat.Menu), staticOpenAI) || !slices.Equal(cat.Recommended, cat.Usable) {
		t.Fatalf("Live %t, menu %v, Recommended %v; want the static list in both", cat.Live, menuIDs(cat.Menu), cat.Recommended)
	}
	for i, e := range cat.Menu {
		if e.Rank != i+1 {
			t.Errorf("%s: rank %d; want %d", e.ID, e.Rank, i+1)
		}
	}
}

// ollamaShowServer is an Ollama with names installed: /api/tags lists them,
// and /api/show gives each the tools capability unless noTools names it. It
// counts the /api/show requests.
func ollamaShowServer(t *testing.T, names []string, noTools map[string]bool) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var shows atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			_, _ = io.WriteString(w, ollamaTags(names...))
		case "/api/show":
			shows.Add(1)
			var body struct {
				Model string `json:"model"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			capabilities := `["completion","tools"]`
			if noTools[body.Model] {
				capabilities = `["completion"]`
			}
			_, _ = io.WriteString(w, `{"model_info":{},"capabilities":`+capabilities+`}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &shows
}

// TestMenu_Ollama_0041: Ollama's menu is the installed models in /api/tags
// order, those /api/show marks as taking no tools after the others and out
// of Recommended; each reply is asked for once and cached.
func TestMenu_Ollama_0041(t *testing.T) {
	resetLookupCaches(t)
	srv, shows := ollamaShowServer(t, []string{"b:1", "notools:1", "a:1"}, map[string]bool{"notools:1": true})
	list := func() Catalog {
		cat, err := List(context.Background(), llmprovider.ProviderOllama, llmprovider.NewStaticToken(""),
			llmprovider.WithBaseURL(srv.URL))
		if err != nil {
			t.Fatal(err)
		}
		return cat
	}
	cat := list()
	if got, want := menuIDs(cat.Menu), []string{"b:1", "a:1", "notools:1"}; !slices.Equal(got, want) {
		t.Errorf("menu %v; want %v", got, want)
	}
	if want := []string{"b:1", "a:1"}; !slices.Equal(cat.Recommended, want) {
		t.Errorf("Recommended %v; want %v", cat.Recommended, want)
	}
	if !slices.Equal(cat.Usable, []string{"b:1", "notools:1", "a:1"}) {
		t.Errorf("Usable %v; want the installed models in order", cat.Usable)
	}
	if e := cat.Menu[2]; !slices.Equal(e.Caveats, []Caveat{CaveatNoTools}) {
		t.Errorf("notools:1: caveats %v; want no tools", e.Caveats)
	}
	if n := shows.Load(); n != 3 {
		t.Errorf("/api/show asked %d times; want once for each of 3 models", n)
	}
	list()
	if n := shows.Load(); n != 3 {
		t.Errorf("/api/show asked %d times after a second listing; want the cached 3", n)
	}
}
