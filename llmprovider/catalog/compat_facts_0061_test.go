package catalog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/compatdialect"
)

// 0061-PLAN Phase 4c: a chatcompat server's facts (0061-MADR E2) and the
// menu of an unranked listing (E4), as the amendment "E2 and E4's menu, made
// exact before Phase 4c" and PLAN deviations 4 to 8 settle them.

// dataEntries is a recorded {"data": [...]} reply's entries, and whether the
// file is one.
func dataEntries(t *testing.T, path string) ([]compatdialect.Entry, bool) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var reply struct {
		Data []compatdialect.Entry `json:"data"`
	}
	if json.Unmarshal(raw, &reply) != nil || len(reply.Data) == 0 || reply.Data[0].ID == "" {
		return nil, false
	}
	return reply.Data, true
}

// TestDialect_Score_0061: each recorded server's /models scores its own
// dialect in full; every other recorded {"data": [...]} listing, the
// built-in providers', scores none at the threshold; one untagged id, or
// one missing created, turns Ollama's off.
func TestDialect_Score_0061(t *testing.T) {
	own, _ := filepath.Glob(filepath.Join("testdata", "compat", "*", "models.json"))
	if len(own) == 0 {
		t.Fatal("no recorded dialect")
	}
	for _, path := range own {
		entries, ok := dataEntries(t, path)
		name := filepath.Base(filepath.Dir(path))
		d, found := compatdialect.Of(entries)
		if !ok || !found || d.Name != name || dialectDetails[d.Name] == nil {
			t.Errorf("%s: dialect %q, %v; want %q, with its details reader", path, d.Name, found, name)
		}
	}
	others := 0
	_ = filepath.WalkDir("testdata", func(path string, de os.DirEntry, err error) error {
		if err != nil || de.IsDir() || !strings.HasSuffix(path, ".json") || strings.Contains(path, filepath.Join("testdata", "compat")) {
			return err
		}
		if entries, ok := dataEntries(t, path); ok {
			others++
			for _, d := range compatdialect.Table {
				if score := compatdialect.Score(d, entries); score >= compatdialect.Threshold {
					t.Errorf("%s scores %s at %v", path, d.Name, score)
				}
			}
		}
		return nil
	})
	if others == 0 {
		t.Error("no other recorded listing was scored")
	}
	entries, _ := dataEntries(t, own[0])
	untagged := slices.Clone(entries)
	untagged[0].ID = "plain-model"
	noCreated := slices.Clone(entries)
	noCreated[1].Created = nil
	for name, listing := range map[string][]compatdialect.Entry{"one untagged id": untagged, "one without created": noCreated} {
		if d, ok := compatdialect.Of(listing); ok {
			t.Errorf("%s: dialect %q, want none", name, d.Name)
		}
	}
}

// TestCanonicalID_0061: X1's canonical id and X2's variant key.
func TestCanonicalID_0061(t *testing.T) {
	for in, want := range map[string]string{
		"llama3.2:latest":                        "llama3-2",
		"qwen3:8b":                               "qwen3-8b",
		"granite3-dense:8b-instruct-q5_K_M":      "granite3-dense-8b-instruct",
		"hf.co/bartowski/Qwen3-8B-GGUF:Q4_K_M":   "qwen3-8b",
		"openai/gpt-oss-20b":                     "gpt-oss-20b",
		"gpt-oss:20b":                            "gpt-oss-20b",
		"Meta-Llama-3.1-8B-Instruct-AWQ":         "meta-llama-3-1-8b-instruct",
		"mistral-7b-iq3_xxs":                     "mistral-7b",
		"phi-4-4bit-mlx":                         "phi-4",
		"deepseek-r1:7b-qwen-distill-fp16":       "deepseek-r1-7b-qwen-distill",
		"Qwen/Qwen3-235B-A22B-Instruct-2507-FP8": "qwen3-235b-a22b-instruct-2507",
		"gemma_2:it-bf16":                        "gemma-2-it",
	} {
		if got := canonicalID(in); got != want {
			t.Errorf("canonicalID(%q) = %q, want %q", in, got, want)
		}
	}
	if a, b := canonicalID("qwen3-235b-a22b-instruct-2507"), canonicalID("qwen3-235b-a22b-2507"); a == b {
		t.Errorf("an instruct model and its base share the canonical id %q", a)
	}
	for _, pair := range [][2]string{
		{"granite3-dense:8b", "granite3-dense:8b-instruct-q5_K_M"},
		{"qwen3-235b-a22b-instruct-2507", "qwen3-235b-a22b-2507"},
		{"gemma3:12b", "gemma3:12b-it-q8_0"},
		{"llama3.1:8b", "llama3.1:8b-chat"},
	} {
		if a, b := variantKey(pair[0]), variantKey(pair[1]); a != b || a == "" {
			t.Errorf("variantKey(%q) = %q, variantKey(%q) = %q; want one key", pair[0], a, pair[1], b)
		}
	}
	if a, b := variantKey("qwen3:8b"), variantKey("qwen3:14b"); a == b {
		t.Errorf("two sizes share the variant key %q", a)
	}
}

// recordedMetadata is the recorded models.dev document.
func recordedMetadata(t *testing.T) Metadata {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", "ranking-2026-10-12", "metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	m, err := ParseMetadata(f)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// TestCanonicalMatch_ExactOnly_0061: a server's model takes models.dev's
// facts only by an equal canonical id, never a near one; they name their
// source.
func TestCanonicalMatch_ExactOnly_0061(t *testing.T) {
	m := recordedMetadata(t)
	info, ok := m.canonicalModel("gpt-oss:20b")
	if !ok || info.ID != "gpt-oss:20b" || !info.Has(FieldContextWindow) || info.SourceOf(FieldContextWindow) != sourceCanonical {
		t.Errorf("gpt-oss:20b = %+v, %v; want models.dev's openai/gpt-oss-20b, from %q", info, ok, sourceCanonical)
	}
	for _, near := range []string{"qwen3:8b", "gpt-oss:21b", "gpt-oss", "deepseek-r1:7b", "llama3.2:latest"} {
		if info, ok := m.canonicalModel(near); ok {
			t.Errorf("%s matched %+v; want nothing", near, info)
		}
	}
}

// TestCanonicalMerge_0061: X5: the value most entries give; a tied limit the
// smaller; any other tie unknown; one entry itself.
func TestCanonicalMerge_0061(t *testing.T) {
	entry := func(window int, tools bool, name string) ModelInfo {
		return ModelInfo{ContextWindow: window, Tools: tools, Name: name}.Known(FieldContextWindow, FieldTools, FieldName)
	}
	got := mergeCanonical([]ModelInfo{entry(128_000, true, "A"), entry(128_000, false, "B"), entry(200_000, true, "A")})
	if got.ContextWindow != 128_000 || !got.Tools || got.Name != "A" || got.SourceOf(FieldName) != sourceCanonical {
		t.Errorf("majority = %+v, want 128000, tools, A, from %q", got, sourceCanonical)
	}
	got = mergeCanonical([]ModelInfo{entry(200_000, true, "A"), entry(128_000, false, "B")})
	if got.ContextWindow != 128_000 || got.Has(FieldTools) || got.Has(FieldName) {
		t.Errorf("tie = %+v, want 128000 and neither tools nor name known", got)
	}
	got = mergeCanonical([]ModelInfo{entry(64_000, false, "C")})
	if got.ContextWindow != 64_000 || !got.Has(FieldTools) || got.Tools || got.Name != "C" {
		t.Errorf("one entry = %+v, want it", got)
	}
}

// ollamaFixtureServer serves the recorded Ollama replies: /v1/models,
// /api/tags, and /api/show for the recorded models, 404 for the others.
func ollamaFixtureServer(t *testing.T) *httptest.Server {
	t.Helper()
	dir := filepath.Join("testdata", "compat", "ollama")
	read := func(name string) []byte {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			_, _ = w.Write(read("models.json"))
		case "/api/tags":
			_, _ = w.Write(read("tags.json"))
		case "/api/show":
			var body struct{ Model string }
			_ = json.NewDecoder(r.Body).Decode(&body)
			name := "show-" + strings.NewReplacer(":", "_", "/", "_").Replace(body.Model) + ".json"
			if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write(read(name))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// compatDoc is a models.dev document of one section the SDK keeps (X9),
// whose qwen3-8b states a window, a price and its tools.
const compatDoc = `{"huggingface":{"id":"huggingface","models":{"Qwen/Qwen3-8B":{"id":"Qwen/Qwen3-8B","name":"Qwen3 8B",` +
	`"limit":{"context":32768,"output":8192},"cost":{"input":0.1,"output":0.2},"tool_call":false}}}}`

// listFixture is List of "acme" against the recorded Ollama, with the
// document at doc ("" for none) and opts.
func listFixture(t *testing.T, doc string, opts ...llmprovider.Option) Catalog {
	t.Helper()
	resetLookupCaches(t)
	srv := ollamaFixtureServer(t)
	meta := []llmprovider.Option{llmprovider.WithoutModelMetadata()}
	if doc != "" {
		docSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(doc)) }))
		t.Cleanup(docSrv.Close)
		meta = []llmprovider.Option{llmprovider.WithModelMetadataURL(docSrv.URL)}
	}
	cat, err := List(context.Background(), "acme", llmprovider.NewStaticToken(""), append(append([]llmprovider.Option{
		withCompatID("acme"), llmprovider.WithBaseURL(srv.URL + "/v1"), WithCache(&Cache{})}, meta...), opts...)...)
	if err != nil || !cat.Live {
		t.Fatalf("List = %+v, %v; want a live listing", cat, err)
	}
	return cat
}

// menuEntry is id's entry in menu.
func menuEntry(t *testing.T, menu []MenuEntry, id string) MenuEntry {
	t.Helper()
	i := slices.IndexFunc(menu, func(e MenuEntry) bool { return e.ID == id })
	if i < 0 {
		t.Fatalf("%s is not in the menu %v", id, menuIDs(menu))
	}
	return menu[i]
}

// TestFacts_Order_0061: X6: the caller's layer, then the server's details,
// then models.dev by canonical id, each field from the strongest that
// states it, and each named; the Ollama dialect's details read from the
// root, as the listing's.
func TestFacts_Order_0061(t *testing.T) {
	caller := staticLayer{name: "caller", info: ModelInfo{ID: "qwen3:8b", Name: "My Qwen"}.Known(FieldName)}
	cat := listFixture(t, compatDoc, WithModelLayer(caller))
	info := menuEntry(t, cat.Menu, "qwen3:8b").Info
	for _, c := range []struct {
		field  Field
		source string
		ok     bool
	}{
		{FieldName, "caller", info.Name == "My Qwen"},
		{FieldContextWindow, listingLayer("acme"), info.ContextWindow == 40960},
		{FieldTools, listingLayer("acme"), info.Tools},
		{FieldReasoning, listingLayer("acme"), info.Reasoning.Supported},
		{FieldCostInput, sourceCanonical, info.Cost.Input == 0.1},
		{FieldMaxOutputTokens, sourceCanonical, info.MaxOutputTokens == 8192},
	} {
		if !c.ok || info.SourceOf(c.field) != c.source {
			t.Errorf("field %d = %+v from %q; want it from %q", c.field, info, info.SourceOf(c.field), c.source)
		}
	}
	gptOSS := menuEntry(t, cat.Menu, "gpt-oss:20b").Info
	if !gptOSS.Has(FieldOutput) || gptOSS.SourceOf(FieldOutput) != listingLayer("acme") || gptOSS.Has(FieldContextWindow) {
		t.Errorf("gpt-oss:20b = %+v; want its /api/tags output, and no window: no show, and no qwen3-8b match", gptOSS)
	}
	plain := listFixture(t, "")
	if info := menuEntry(t, plain.Menu, "qwen3:8b").Info; info.Has(FieldCostInput) || info.ContextWindow != 40960 {
		t.Errorf("with no document: %+v; want the details alone", info)
	}
}

// TestMenu_Variants_0061: X2: a model's variants sit at the first one's
// place, the plainest first; a caveated variant stays with the caveated.
func TestMenu_Variants_0061(t *testing.T) {
	now := time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)
	ids := []string{"granite3-dense:8b-instruct-q5_K_M", "llama3.2:latest", "granite3-dense:8b", "granite3-dense:8b-q8_0",
		"granite3-dense:8b-chat"}
	facts := map[string]ModelInfo{"granite3-dense:8b-chat": ModelInfo{Status: StatusDeprecated}.Known(FieldStatus)}
	cat := compatMenu("acme", ids, facts, nil, 0, now)
	want := []string{"granite3-dense:8b", "granite3-dense:8b-q8_0", "granite3-dense:8b-instruct-q5_K_M", "llama3.2:latest",
		"granite3-dense:8b-chat"}
	if got := menuIDs(cat.Menu); !slices.Equal(got, want) {
		t.Errorf("compat menu = %v, want %v", got, want)
	}
	if !slices.Equal(cat.Usable, ids) || !slices.Equal(cat.Recommended, want[:4]) {
		t.Errorf("Usable %v, Recommended %v; want the server's order, and the menu less the deprecated", cat.Usable, cat.Recommended)
	}
	ollama := ollamaMenu(ids, facts, nil, 0, now)
	if got := menuIDs(ollama.Menu); !slices.Equal(got, want) {
		t.Errorf("Ollama menu = %v, want %v", got, want)
	}
}

// TestMenu_DefaultRank_0061: X3: the uncaveated first; then the larger known
// window; then the newer created; then the listing's order; the band ranked
// by place; the id-token caveat on a chatcompat listing alone (deviation 8);
// and the recorded Ollama, end to end.
func TestMenu_DefaultRank_0061(t *testing.T) {
	now := time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)
	day := func(d int) time.Time { return time.Date(2026, 7, d, 0, 0, 0, 0, time.UTC) }
	window := func(n int) ModelInfo { return ModelInfo{ContextWindow: n}.Known(FieldContextWindow) }
	ids := []string{"old", "nomic-embed-text:latest", "new", "small-window", "big-window", "undated", "no-tools"}
	facts := map[string]ModelInfo{"small-window": window(8_000), "big-window": window(128_000),
		"no-tools": ModelInfo{Tools: false}.Known(FieldTools)}
	created := map[string]time.Time{"old": day(1), "new": day(9), "nomic-embed-text:latest": day(20), "no-tools": day(30)}
	cat := compatMenu("acme", ids, facts, created, 0, now)
	// The caveated follow, ordered alike: no-tools, created on the 30th,
	// before nomic-embed-text, on the 20th.
	want := []string{"big-window", "small-window", "new", "old", "undated", "no-tools", "nomic-embed-text:latest"}
	if got := menuIDs(cat.Menu); !slices.Equal(got, want) {
		t.Errorf("compat menu = %v, want %v", got, want)
	}
	if !slices.Equal(cat.Recommended, want[:5]) {
		t.Errorf("Recommended %v, want %v", cat.Recommended, want[:5])
	}
	for i, e := range cat.Menu {
		if wantRank := map[bool]int{true: i + 1, false: 0}[i < 5]; e.Rank != wantRank {
			t.Errorf("%s: rank %d, want %d", e.ID, e.Rank, wantRank)
		}
	}
	if got := menuEntry(t, cat.Menu, "nomic-embed-text:latest").Caveats; !slices.Equal(got, []Caveat{CaveatNoTextOutput}) {
		t.Errorf("nomic-embed-text on a compat listing: caveats %v, want no-text-output", got)
	}
	ollama := ollamaMenu(ids, facts, created, 0, now)
	if e := menuEntry(t, ollama.Menu, "nomic-embed-text:latest"); len(e.Caveats) != 0 ||
		!slices.Contains(ollama.Recommended, e.ID) {
		t.Errorf("nomic-embed-text on Ollama: caveats %v; want none, and recommended (0042-MADR D1)", e.Caveats)
	}

	// A plain server's created orders its menu, where its order differs:
	// the recorded Ollama lists the newest first, so its order alone would
	// pass for created's.
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"id":"a","created":100},{"id":"b","created":300},{"id":"c"},{"id":"d","created":200}]}`))
	}))
	t.Cleanup(plain.Close)
	listed, err := List(context.Background(), "acme", llmprovider.NewStaticToken(""), withCompatID("acme"),
		llmprovider.WithBaseURL(plain.URL), WithCache(&Cache{}), llmprovider.WithoutModelMetadata())
	if got := menuIDs(listed.Menu); err != nil || !slices.Equal(got, []string{"b", "d", "a", "c"}) {
		t.Errorf("a plain server's menu = %v, %v; want b, d, a, then c with no created", got, err)
	}

	recorded := listFixture(t, "")
	wantRecorded := []string{"qwen3:8b", "llama3.2:latest", "granite3-dense:8b", "granite3-dense:8b-instruct-q5_K_M",
		"gemma4:latest", "deepseek-r1:7b", "gpt-oss:20b", "gemma3:12b"}
	if got := menuIDs(recorded.Menu); !slices.Equal(got, wantRecorded) {
		t.Errorf("the recorded Ollama's menu = %v, want %v", got, wantRecorded)
	}
	if got := menuEntry(t, recorded.Menu, "gemma3:12b").Caveats; !slices.Equal(got, []Caveat{CaveatNoTools}) {
		t.Errorf("gemma3:12b: caveats %v, want no-tools, from its /api/show", got)
	}
}
