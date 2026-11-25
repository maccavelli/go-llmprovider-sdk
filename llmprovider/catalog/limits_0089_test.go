package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// The tests of 0089-PLAN Phase 1: discovery (0089-MADR DL1 to DL30, DL66,
// DL70, DL71, DL74). Each serves its own listings and documents; none
// reaches a network. DL72's and DL75's test is in providers/claude, which
// imports this package.

// limitsCache is a cache of t's own, closed when t ends.
func limitsCache(t *testing.T) (*Cache, llmprovider.Option) {
	t.Helper()
	c := new(Cache)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := c.Close(ctx); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	return c, WithCache(c)
}

// withClock fixes the listing and document clocks at now, which the
// returned function moves.
func withClock(t *testing.T) func(time.Duration) {
	t.Helper()
	savedL := listingNow
	var mu sync.Mutex
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return now
	}
	listingNow = clock
	t.Cleanup(func() { listingNow = savedL })
	return func(d time.Duration) {
		mu.Lock()
		defer mu.Unlock()
		now = now.Add(d)
	}
}

// fixture reads a recorded file under testdata.
func fixture(t *testing.T, parts ...string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(append([]string{"testdata"}, parts...)...))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// claudePage is the recorded Claude listing's one page.
func claudePage(t *testing.T) []byte {
	t.Helper()
	var pages []json.RawMessage
	if err := json.Unmarshal(fixture(t, "firstparty-2026-10-14", "claude.json"), &pages); err != nil {
		t.Fatal(err)
	}
	return pages[0]
}

// claudeEntry is the recorded Claude listing's entry for id.
func claudeEntry(t *testing.T, id string) []byte {
	t.Helper()
	var page struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(claudePage(t), &page); err != nil {
		t.Fatal(err)
	}
	for _, e := range page.Data {
		var m struct{ ID string }
		if json.Unmarshal(e, &m) == nil && m.ID == id {
			return e
		}
	}
	t.Fatalf("no %s in the recording", id)
	return nil
}

// countingServer serves routes by path and counts each path's requests.
type countingServer struct {
	*httptest.Server
	mu     sync.Mutex
	counts map[string]int
}

func newCountingServer(t *testing.T, handle func(w http.ResponseWriter, r *http.Request)) *countingServer {
	t.Helper()
	s := &countingServer{counts: map[string]int{}}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.counts[r.URL.Path]++
		s.mu.Unlock()
		handle(w, r)
	}))
	t.Cleanup(s.Close)
	return s
}

func (s *countingServer) count(path string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.counts[path]
}

func (s *countingServer) total() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, c := range s.counts {
		n += c
	}
	return n
}

// claudeServer serves the recorded listing at /v1/models, and each
// /v1/models/{id} in one: the dated 4.5 entry for the alias
// claude-sonnet-4-5, 404 for any other.
func claudeServer(t *testing.T) *countingServer {
	t.Helper()
	page, alias := claudePage(t), claudeEntry(t, "claude-sonnet-4-5-20250929")
	return newCountingServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			_, _ = w.Write(page)
		case "/v1/models/claude-sonnet-4-5":
			_, _ = w.Write(alias)
		default:
			http.NotFound(w, r)
		}
	})
}

// docServer serves doc as the models.dev document.
func docServer(t *testing.T, doc string) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(doc)) }))
	t.Cleanup(s.Close)
	return s
}

// limitLayer is a caller's layer stating a model's output and window.
type limitLayer struct {
	name                string
	output, window      int
	hasOutput, hasWindw bool
}

func (l limitLayer) Name() string { return l.name }

func (l limitLayer) Model(_ llmprovider.ProviderID, id string) (ModelInfo, bool) {
	info := ModelInfo{ID: id, layer: l.name, MaxOutputTokens: l.output, ContextWindow: l.window}
	var known []Field
	if l.hasOutput {
		known = append(known, FieldMaxOutputTokens)
	}
	if l.hasWindw {
		known = append(known, FieldContextWindow)
	}
	return info.Known(known...), true
}

func outputLayer(name string, output int) limitLayer {
	return limitLayer{name: name, output: output, hasOutput: true}
}

// lookupLimitsT is LookupLimits, failing t on an error.
func lookupLimitsT(t *testing.T, id llmprovider.ProviderID, src llmprovider.TokenSource, model string, opts ...llmprovider.Option) Limits {
	t.Helper()
	l, err := LookupLimits(context.Background(), id, src, model, opts...)
	if err != nil {
		t.Fatalf("LookupLimits(%s, %s): %v", id, model, err)
	}
	return l
}

// wantFact fails t unless f is known at tokens, chosen from src.
func wantFact(t *testing.T, what string, f Fact, tokens int, src string) {
	t.Helper()
	if !f.Known || f.Tokens != tokens || len(f.Evidence) == 0 || f.Evidence[0].Source != src {
		t.Errorf("%s = %+v; want %d from %s", what, f, tokens, src)
	}
}

// wantUnknown fails t unless f is unknown with no evidence.
func wantUnknown(t *testing.T, what string, f Fact) {
	t.Helper()
	if f.Known || len(f.Evidence) != 0 {
		t.Errorf("%s = %+v; want unknown, no source stating it", what, f)
	}
}

// hasSource reports whether f's evidence holds src at tokens.
func hasSource(f Fact, src string, tokens int) bool {
	for _, e := range f.Evidence {
		if e.Source == src && e.Tokens == tokens {
			return true
		}
	}
	return false
}

// TestDiscover_ListingFacts_0089 (DL1, DL3, DL5): Claude's listing states
// the output and the input, not the window; Gemini's the same; Kilo's the
// output and the window, and a null cap states nothing.
func TestDiscover_ListingFacts_0089(t *testing.T) {
	key := llmprovider.NewStaticToken("k")

	claude := claudeServer(t)
	_, c := limitsCache(t)
	l := lookupLimitsT(t, llmprovider.ProviderClaude, key, "claude-opus-4-6", llmprovider.WithBaseURL(claude.URL), c)
	wantFact(t, "claude output", l.Output, 128000, "listing:claude")
	wantFact(t, "claude input", l.Input, 1000000, "listing:claude")
	wantUnknown(t, "claude window", l.Window)

	var pages []json.RawMessage
	if err := json.Unmarshal(fixture(t, "firstparty-2026-10-14", "gemini.json"), &pages); err != nil {
		t.Fatal(err)
	}
	raw := make([][]byte, len(pages))
	for i, p := range pages {
		raw[i] = p
	}
	gemini := newPagedServer(t, raw...)
	_, c = limitsCache(t)
	l = lookupLimitsT(t, llmprovider.ProviderGemini, key, "gemini-2.5-pro", llmprovider.WithBaseURL(gemini.URL), c)
	wantFact(t, "gemini output", l.Output, 65536, "listing:gemini")
	wantFact(t, "gemini input", l.Input, 1048576, "listing:gemini")
	wantUnknown(t, "gemini window", l.Window)

	kiloPage := fixture(t, "ranking-2026-10-12", "kilo.json")
	kilo := newCountingServer(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(kiloPage) })
	_, c = limitsCache(t)
	opts := []llmprovider.Option{llmprovider.WithBaseURL(kilo.URL), c}
	l = lookupLimitsT(t, llmprovider.ProviderKilo, key, "kilo-auto/balanced", opts...)
	wantFact(t, "kilo output", l.Output, 65536, "listing:kilo")
	wantFact(t, "kilo window", l.Window, 1000000, "listing:kilo")
	l = lookupLimitsT(t, llmprovider.ProviderKilo, key, "openrouter/auto", opts...)
	wantUnknown(t, "kilo null cap", l.Output)
	wantFact(t, "kilo null cap's window", l.Window, 2000000, "listing:kilo")
}

// TestDiscover_ClaudeAlias_0089 (DL2): a model the listing does not name is
// read by GET /v1/models/{id}: the alias resolves to the dated entry, whose
// id is kept; a 404 leaves it unknown.
func TestDiscover_ClaudeAlias_0089(t *testing.T) {
	s := claudeServer(t)
	cache, c := limitsCache(t)
	opts := []llmprovider.Option{llmprovider.WithBaseURL(s.URL), c}
	key := llmprovider.NewStaticToken("k")
	l := lookupLimitsT(t, llmprovider.ProviderClaude, key, "claude-sonnet-4-5", opts...)
	wantFact(t, "alias output", l.Output, 64000, "listing:claude")
	cfg := testConfig(t, append(opts, llmprovider.WithAPIKey("k"))...)
	lk := listingKeyFor(llmprovider.ProviderClaude, cfg, llmprovider.Token{Value: "k"})
	if r, ok := cache.value().Catalog.(*store).read(limitReadKey{lk, "claude-sonnet-4-5", readOneModel}); !ok ||
		r.resolved != "claude-sonnet-4-5-20250929" {
		t.Errorf("the one-model read = %+v, %v; want the resolved id kept", r, ok)
	}
	l = lookupLimitsT(t, llmprovider.ProviderClaude, key, "claude-nope", opts...)
	wantUnknown(t, "a 404's output", l.Output)
}

// ollamaServer is a built-in Ollama: /api/tags lists qwen3:8b and
// gemma3:12b, /api/show states a trained context of 131,072 for each, and
// /api/ps serves qwen3:8b loaded at 8,192.
func ollamaServer(t *testing.T) *countingServer {
	t.Helper()
	return newCountingServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			_, _ = w.Write([]byte(`{"models":[{"name":"qwen3:8b","digest":"a"},{"name":"gemma3:12b","digest":"b"}]}`))
		case "/api/show":
			_, _ = w.Write([]byte(`{"model_info":{"general.architecture":"qwen3","qwen3.context_length":131072},` +
				`"capabilities":["completion","tools"]}`))
		case "/api/ps":
			_, _ = w.Write([]byte(`{"models":[{"name":"qwen3:8b","model":"qwen3:8b","context_length":8192}]}`))
		default:
			http.NotFound(w, r)
		}
	})
}

// TestDiscover_ReadsSurviveRefresh_0089 (DL66): a one-model read and a
// served read are kept apart from the listing, so a refresh keeps them; a
// 404 is kept, so a second lookup within ten minutes sends no request.
func TestDiscover_ReadsSurviveRefresh_0089(t *testing.T) {
	move := withClock(t)
	ctx := context.Background()

	claude := claudeServer(t)
	_, c := limitsCache(t)
	copts := []llmprovider.Option{llmprovider.WithBaseURL(claude.URL), c}
	key := llmprovider.NewStaticToken("k")
	lookupLimitsT(t, llmprovider.ProviderClaude, key, "claude-sonnet-4-5", copts...)
	lookupLimitsT(t, llmprovider.ProviderClaude, key, "claude-nope", copts...)
	lookupLimitsT(t, llmprovider.ProviderClaude, key, "claude-nope", copts...)
	if n := claude.count("/v1/models/claude-nope"); n != 1 {
		t.Errorf("a 404, looked up twice within ten minutes: %d requests; want 1", n)
	}

	ollama := ollamaServer(t)
	_, oc := limitsCache(t)
	oopts := []llmprovider.Option{llmprovider.WithBaseURL(ollama.URL), oc}
	lookupLimitsT(t, llmprovider.ProviderOllama, llmprovider.NewStaticToken(""), "qwen3:8b", oopts...)

	move(11 * time.Minute)
	if _, err := List(ctx, llmprovider.ProviderClaude, key, copts...); err != nil {
		t.Fatalf("List claude: %v", err)
	}
	if _, err := List(ctx, llmprovider.ProviderOllama, llmprovider.NewStaticToken(""), oopts...); err != nil {
		t.Fatalf("List ollama: %v", err)
	}
	if n := claude.count("/v1/models"); n != 2 {
		t.Fatalf("listing requests = %d; want 2, the refresh included", n)
	}
	l := CachedLimits(ctx, llmprovider.ProviderClaude, "claude-sonnet-4-5", append(copts, llmprovider.WithAPIKey("k"))...)
	wantFact(t, "the alias after a refresh", l.Output, 64000, "listing:claude")
	l = CachedLimits(ctx, llmprovider.ProviderOllama, "qwen3:8b", append(oopts, llmprovider.WithAPIKey(""))...)
	wantFact(t, "the served context after a refresh", l.Window, 8192, "server:ollama")
}

// TestDiscover_UnreadFields_0089 (DL4, DL8, DL9, DL10): the listing fields
// the SDK did not read: Kilo's per_request_limits, the smaller; Together's
// config.max_output_length; Hugging Face's per-provider windows, the
// suffixed provider's or the smallest; the ChatGPT backend's
// context_window.
func TestDiscover_UnreadFields_0089(t *testing.T) {
	key := llmprovider.NewStaticToken("k")
	serve := func(body []byte) *countingServer {
		return newCountingServer(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(body) })
	}

	kilo := serve([]byte(`{"data":[{"id":"x/m","context_length":131072,"top_provider":{"max_completion_tokens":65536},` +
		`"per_request_limits":{"prompt_tokens":100000,"completion_tokens":32768}}]}`))
	_, c := limitsCache(t)
	l := lookupLimitsT(t, llmprovider.ProviderKilo, key, "x/m", llmprovider.WithBaseURL(kilo.URL), c)
	wantFact(t, "kilo per-request output", l.Output, 32768, "listing:kilo")

	together := serve(fixture(t, "ranking-2026-10-12", "together.json"))
	_, c = limitsCache(t)
	l = lookupLimitsT(t, llmprovider.ProviderTogether, key, "deepseek-ai/DeepSeek-V3.1", llmprovider.WithBaseURL(together.URL), c)
	wantFact(t, "together output", l.Output, 40000, "listing:together")

	hf := serve(fixture(t, "ranking-2026-10-12", "hf.json"))
	_, c = limitsCache(t)
	hopts := []llmprovider.Option{llmprovider.WithBaseURL(hf.URL), c}
	l = lookupLimitsT(t, llmprovider.ProviderHuggingFace, key, "Qwen/Qwen3.8-27B", hopts...)
	wantFact(t, "hugging face window, the smallest", l.Window, 262144, "listing:huggingface")
	l = lookupLimitsT(t, llmprovider.ProviderHuggingFace, key, "Qwen/Qwen3.8-27B:novita", hopts...)
	wantFact(t, "hugging face window, novita's", l.Window, 1000000, "listing:huggingface")

	chatgpt := serve(fixture(t, "rawlistings-2026-10-15", "chatgpt.json"))
	_, c = limitsCache(t)
	l = lookupLimitsT(t, llmprovider.ProviderOpenAI, chatGPTSession(), "gpt-6-astra", llmprovider.WithBaseURL(chatgpt.URL), c)
	wantFact(t, "chatgpt window", l.Window, 272000, "listing:openai")
}

// TestDiscover_OllamaServed_0089 (DL12, DL13): a loaded model's served
// context ranks above its trained one, and the two conflict; a model not
// loaded has the trained one; and the same through chatcompat's Ollama
// dialect.
func TestDiscover_OllamaServed_0089(t *testing.T) {
	none := llmprovider.NewStaticToken("")

	s := ollamaServer(t)
	_, c := limitsCache(t)
	opts := []llmprovider.Option{llmprovider.WithBaseURL(s.URL), c}
	l := lookupLimitsT(t, llmprovider.ProviderOllama, none, "qwen3:8b", opts...)
	wantFact(t, "loaded window", l.Window, 8192, "server:ollama")
	if !l.Window.Conflicting || !hasSource(l.Window, "listing:ollama", 131072) {
		t.Errorf("loaded window = %+v; want the trained 131,072 kept, conflicting", l.Window)
	}
	l = lookupLimitsT(t, llmprovider.ProviderOllama, none, "gemma3:12b", opts...)
	wantFact(t, "not loaded window", l.Window, 131072, "listing:ollama")

	dir := filepath.Join("testdata", "compat", "ollama")
	compat := newCountingServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			_, _ = w.Write(fixture(t, "compat", "ollama", "models.json"))
		case "/api/tags":
			_, _ = w.Write(fixture(t, "compat", "ollama", "tags.json"))
		case "/api/show":
			var body struct{ Model string }
			_ = json.NewDecoder(r.Body).Decode(&body)
			name := "show-" + strings.NewReplacer(":", "_", "/", "_").Replace(body.Model) + ".json"
			if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write(fixture(t, "compat", "ollama", name))
		case "/api/ps":
			_, _ = w.Write([]byte(`{"models":[{"name":"qwen3:8b","model":"qwen3:8b","context_length":8192}]}`))
		default:
			http.NotFound(w, r)
		}
	})
	_, c = limitsCache(t)
	l = lookupLimitsT(t, "acme", none, "qwen3:8b", withCompatID("acme"), llmprovider.WithBaseURL(compat.URL+"/v1"), c)
	wantFact(t, "dialect loaded window", l.Window, 8192, "server:acme")
	if !l.Window.Conflicting || !hasSource(l.Window, "listing:acme", 40960) {
		t.Errorf("dialect window = %+v; want the trained 40,960 kept, conflicting", l.Window)
	}
}

// claudeOpusDoc is a models.dev document stating claude-opus-4-6's output
// as tokens.
func claudeOpusDoc(tokens string) string {
	return `{"anthropic":{"models":{"claude-opus-4-6":{"limit":{"context":1000000,"output":` + tokens + `}}}}}`
}

// TestDiscover_EvidenceKinds_0089 (DL15, DL16, DL29): a layer named
// "models.dev" is registry evidence, below the listing; any other layer is
// a caller's, above it; with no document handed in, a "models.dev" layer
// is the only registry evidence: nothing embedded answers (0093-MADR D7,
// D12 (c)).
func TestDiscover_EvidenceKinds_0089(t *testing.T) {
	key := llmprovider.NewStaticToken("k")
	meta, err := ParseMetadata(strings.NewReader(claudeOpusDoc("64000")))
	if err != nil {
		t.Fatal(err)
	}
	s := claudeServer(t)
	_, c := limitsCache(t)
	base := []llmprovider.Option{llmprovider.WithBaseURL(s.URL), c}

	l := lookupLimitsT(t, llmprovider.ProviderClaude, key, "claude-opus-4-6", append(base, WithModelLayer(meta))...)
	wantFact(t, "listing over a models.dev layer", l.Output, 128000, "listing:claude")
	if !hasSource(l.Output, "models.dev", 64000) {
		t.Errorf("output = %+v; want the models.dev layer's 64,000 kept", l.Output)
	}
	if len(l.Output.Evidence) != 2 {
		t.Errorf("output = %+v; want the listing's and the layer's alone, no extract", l.Output)
	}

	l = lookupLimitsT(t, llmprovider.ProviderClaude, key, "claude-opus-4-6", append(base, WithModelLayer(outputLayer("models.json", 20000)))...)
	wantFact(t, "a caller's layer over the listing", l.Output, 20000, "models.json")
}

// TestDiscover_EmbeddedExtract_0089 (DL70, DL24; 0093-MADR D7, D12 (a) to
// (c)): nothing is embedded. The extract's limits are the committed fixture,
// a document a test hands in; a handed-in document's facts are its own,
// undated and never stale, with no extract beneath them; with no document,
// nothing is known.
func TestDiscover_EmbeddedExtract_0089(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "extract-2026-10-12", "api.json"))
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := ParseMetadata(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if info, ok := fixture.Model(llmprovider.ProviderClaude, "claude-opus-4-6"); !ok || info.MaxOutputTokens != 128000 {
		t.Errorf("the fixture's claude-opus-4-6 = %+v, %v; want output 128,000", info, ok)
	}

	_, c := limitsCache(t)
	l, err := LookupLimits(context.Background(), llmprovider.ProviderClaude, nil, "claude-opus-4-6",
		handed(t, claudeOpusDoc("32000")), c)
	if err != nil {
		t.Fatal(err)
	}
	wantFact(t, "the document", l.Output, 32000, "models.dev")
	if len(l.Output.Evidence) != 1 || !l.Output.Evidence[0].Seen.IsZero() || l.Output.Evidence[0].Stale {
		t.Errorf("output = %+v; want the document's 32,000 alone, undated, not stale", l.Output)
	}

	_, c = limitsCache(t)
	l, err = LookupLimits(context.Background(), llmprovider.ProviderClaude, nil, "claude-opus-4-6", c)
	if err != nil {
		t.Errorf("no document: err = %v; want none", err)
	}
	wantUnknown(t, "no document", l.Output)
}

// TestDiscover_MatchStages_0089 (DL71): the exact entry, else the canonical
// id's, else the family's, each under its own source name.
func TestDiscover_MatchStages_0089(t *testing.T) {
	both := `{"anthropic":{"models":{"claude-opus-4-6-20260205":{"limit":{"context":1000000,"output":128000}},` +
		`"claude-opus-4-6":{"limit":{"context":1000000,"output":64000}}}}}`
	family := `{"anthropic":{"models":{"claude-opus-4-6":{"limit":{"context":1000000,"output":64000}}}}}`
	run := func(doc string, id llmprovider.ProviderID, model string) Limits {
		t.Helper()
		_, c := limitsCache(t)
		return lookupLimitsT(t, id, nil, model, servedDoc(t, docServer(t, doc).URL), c)
	}
	wantFact(t, "exact", run(both, llmprovider.ProviderClaude, "claude-opus-4-6-20260205").Output, 128000, "models.dev")
	wantFact(t, "family", run(family, llmprovider.ProviderClaude, "claude-opus-4-6-20260205").Output, 64000, "models.dev (family)")
	wantFact(t, "canonical", run(family, llmprovider.ProviderKilo, "anthropic/claude-opus-4.6").Output, 64000,
		"models.dev (canonical id)")
}

// TestDiscover_Conflict_0089 (DL17, DL18): sources that differ conflict, and
// the higher rank is chosen; of two caller layers, the later.
func TestDiscover_Conflict_0089(t *testing.T) {
	s := claudeServer(t)
	doc := docServer(t, `{"anthropic":{"models":{"claude-opus-4-5-20251101":{"limit":{"context":200000,"output":128000}}}}}`)
	_, c := limitsCache(t)
	l := lookupLimitsT(t, llmprovider.ProviderClaude, llmprovider.NewStaticToken("k"), "claude-opus-4-5-20251101",
		llmprovider.WithBaseURL(s.URL), servedDoc(t, doc.URL), c)
	wantFact(t, "listing over document", l.Output, 64000, "listing:claude")
	if !l.Output.Conflicting || !hasSource(l.Output, "models.dev", 128000) {
		t.Errorf("output = %+v; want conflicting, the document's 128,000 kept", l.Output)
	}

	_, c = limitsCache(t)
	l = lookupLimitsT(t, llmprovider.ProviderClaude, nil, "m", c,
		WithModelLayer(outputLayer("first", 20000)), WithModelLayer(outputLayer("second", 16000)))
	wantFact(t, "the later layer", l.Output, 16000, "second")
	if !l.Output.Conflicting {
		t.Errorf("output = %+v; want conflicting", l.Output)
	}
}

// TestDiscover_OutputAboveWindow_0089 (DL19): an output above the window is
// held to it, conflicting.
func TestDiscover_OutputAboveWindow_0089(t *testing.T) {
	_, c := limitsCache(t)
	l := lookupLimitsT(t, llmprovider.ProviderClaude, nil, "m", c,
		WithModelLayer(limitLayer{name: "models.json", output: 16384, window: 8192, hasOutput: true, hasWindw: true}))
	if l.Output.Tokens != 8192 || !l.Output.Conflicting || !hasSource(l.Output, "models.json", 16384) {
		t.Errorf("output = %+v; want 8,192, conflicting, the 16,384 kept", l.Output)
	}
}

// TestDiscover_StatedZero_0089 (DL20): a stated 0 is not a limit, and is
// kept.
func TestDiscover_StatedZero_0089(t *testing.T) {
	_, c := limitsCache(t)
	l := lookupLimitsT(t, llmprovider.ProviderClaude, nil, "m", c,
		WithModelLayer(outputLayer("models.json", 0)))
	if l.Output.Known || l.Output.Tokens != 0 || !hasSource(l.Output, "models.json", 0) {
		t.Errorf("output = %+v; want unknown, the 0 kept", l.Output)
	}
}

// TestDiscover_Unknown_0089 (DL21): a model no source knows, the extract
// included: every fact unknown, no error.
func TestDiscover_Unknown_0089(t *testing.T) {
	_, c := limitsCache(t)
	l, err := LookupLimits(context.Background(), llmprovider.ProviderClaude, nil, "claude-no-such-model",
		c)
	if err != nil {
		t.Errorf("err = %v; want nil", err)
	}
	wantUnknown(t, "output", l.Output)
	wantUnknown(t, "window", l.Window)
	wantUnknown(t, "input", l.Input)
}

// TestDiscover_SourceFailure_0089 (DL22): a failing listing is reported,
// its *APIError or ErrAuthFailure in the error, beside the document's facts.
func TestDiscover_SourceFailure_0089(t *testing.T) {
	for _, status := range []int{http.StatusServiceUnavailable, http.StatusUnauthorized} {
		s := newCountingServer(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) })
		doc := docServer(t, claudeOpusDoc("64000"))
		_, c := limitsCache(t)
		l, err := LookupLimits(context.Background(), llmprovider.ProviderClaude, llmprovider.NewStaticToken("k"),
			"claude-opus-4-6", llmprovider.WithBaseURL(s.URL), servedDoc(t, doc.URL), c)
		wantFact(t, "document's output", l.Output, 64000, "models.dev")
		var apiErr *llmprovider.APIError
		if !errors.As(err, &apiErr) {
			t.Errorf("%d: err = %v; want the listing's *APIError", status, err)
		}
		if status == http.StatusUnauthorized && !errors.Is(err, llmprovider.ErrAuthFailure) {
			t.Errorf("401: err = %v; want ErrAuthFailure", err)
		}
	}
}

// TestDiscover_Stale_0089 (DL23): a fact older than its freshness is still
// returned, marked stale, its Seen unchanged.
func TestDiscover_Stale_0089(t *testing.T) {
	move := withClock(t)
	s := claudeServer(t)
	_, c := limitsCache(t)
	opts := []llmprovider.Option{llmprovider.WithBaseURL(s.URL), c}
	first := lookupLimitsT(t, llmprovider.ProviderClaude, llmprovider.NewStaticToken("k"), "claude-opus-4-6", opts...)
	if first.Output.Evidence[0].Stale {
		t.Fatalf("a fresh read: %+v", first.Output)
	}
	move(11 * time.Minute)
	l := CachedLimits(context.Background(), llmprovider.ProviderClaude, "claude-opus-4-6", append(opts, llmprovider.WithAPIKey("k"))...)
	wantFact(t, "stale output", l.Output, 128000, "listing:claude")
	if e := l.Output.Evidence[0]; !e.Stale || !e.Seen.Equal(first.Output.Evidence[0].Seen) {
		t.Errorf("evidence = %+v; want stale, seen at %v", e, first.Output.Evidence[0].Seen)
	}
}

// TestDiscover_Offline_0089 (DL24, its Phase 1 part): every server closed
// and an empty document handed in: the caller's layer answers, the empty
// document states nothing and nothing embedded answers (0093-MADR D3, D7,
// D12 (c)); a passed-back cap is kept beside it (0089-PLAN deviation 1: its
// case added in the merged phase).
func TestDiscover_Offline_0089(t *testing.T) {
	listing := newCountingServer(t, func(w http.ResponseWriter, _ *http.Request) {})
	listing.Close()
	_, c := limitsCache(t)
	l, err := LookupLimits(context.Background(), llmprovider.ProviderClaude, llmprovider.NewStaticToken("k"), "claude-opus-4-6",
		llmprovider.WithBaseURL(listing.URL), handed(t, `{}`), c,
		WithModelLayer(outputLayer("models.json", 20000)))
	var apiErr *llmprovider.APIError
	if err == nil || errors.As(err, &apiErr) {
		t.Errorf("err = %v; want the listing's transport failure", err)
	}
	wantFact(t, "offline output", l.Output, 20000, "models.json")
	if len(l.Output.Evidence) != 1 {
		t.Errorf("output = %+v; want the caller's layer alone, no extract", l.Output)
	}
	for _, f := range []Fact{l.Output, l.Window, l.Input} {
		for _, e := range f.Evidence {
			if e.Source == "models.dev" {
				t.Errorf("evidence %+v from the empty document", e)
			}
		}
	}

	// The passed-back cap's case: offline, on a cold cache, a caller layer
	// and a passed-back cap: those two alone.
	_, c = limitsCache(t)
	back := llmprovider.LimitObservation{Model: "claude-opus-4-6", BaseURL: listing.URL, Tokens: 30000, Seen: time.Now().Add(-time.Hour)}
	l, err = LookupLimits(context.Background(), llmprovider.ProviderClaude, llmprovider.NewStaticToken("k"), "claude-opus-4-6",
		llmprovider.WithBaseURL(listing.URL), handed(t, `{}`), c,
		WithModelLayer(outputLayer("models.json", 20000)), llmprovider.WithLimitObservations(back))
	if err == nil || errors.As(err, &apiErr) {
		t.Errorf("passed back: err = %v; want the listing's transport failure", err)
	}
	wantFact(t, "offline output, passed back", l.Output, 20000, "models.json")
	if len(l.Output.Evidence) != 2 || !hasSource(l.Output, "remembered", 30000) {
		t.Errorf("offline output, passed back = %+v; want the layer's and the remembered 30,000 alone", l.Output)
	}
}

// TestDiscover_Closed_0089 (DL25): a closed cache is refused, and no request
// reaches the listing, the document, the one-model read or /api/ps.
func TestDiscover_Closed_0089(t *testing.T) {
	claude, ollama := claudeServer(t), ollamaServer(t)
	doc := newCountingServer(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{}`)) })
	cache, c := limitsCache(t)
	if err := cache.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		id    llmprovider.ProviderID
		model string
		base  string
	}{{llmprovider.ProviderClaude, "claude-sonnet-4-5", claude.URL}, {llmprovider.ProviderOllama, "qwen3:8b", ollama.URL}} {
		_, err := LookupLimits(context.Background(), tc.id, llmprovider.NewStaticToken("k"), tc.model,
			llmprovider.WithBaseURL(tc.base), handed(t, `{}`), c)
		if !errors.Is(err, llmprovider.ErrInvalidRequest) {
			t.Errorf("%s: err = %v; want the closed-cache error", tc.id, err)
		}
	}
	for name, s := range map[string]*countingServer{"claude": claude, "ollama": ollama, "document": doc} {
		if n := s.total(); n != 0 {
			t.Errorf("%s: %d requests after Close; want none", name, n)
		}
	}
}

// TestDiscover_Cancelled_0089 (DL26): ctx cancelled while the listing
// stalls: ctx's error, beside the layers' facts.
func TestDiscover_Cancelled_0089(t *testing.T) {
	arrived, release := make(chan struct{}, 1), make(chan struct{})
	s := newCountingServer(t, func(_ http.ResponseWriter, _ *http.Request) {
		select {
		case arrived <- struct{}{}:
		default:
		}
		<-release
	})
	t.Cleanup(func() { close(release) })
	_, c := limitsCache(t)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-arrived
		cancel()
	}()
	l, err := LookupLimits(ctx, llmprovider.ProviderClaude, llmprovider.NewStaticToken("k"), "claude-opus-4-6",
		llmprovider.WithBaseURL(s.URL), c, WithModelLayer(outputLayer("models.json", 20000)))
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v; want ctx's", err)
	}
	wantFact(t, "the layer's output", l.Output, 20000, "models.json")
}

// TestDiscover_Concurrent_0089 (DL27): concurrent lookups of one model share
// one listing fetch; CachedLimits then reads it.
func TestDiscover_Concurrent_0089(t *testing.T) {
	page := claudePage(t)
	s := newCountingServer(t, func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(50 * time.Millisecond)
		_, _ = w.Write(page)
	})
	_, c := limitsCache(t)
	opts := []llmprovider.Option{llmprovider.WithBaseURL(s.URL), c}
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			<-start
			if _, err := LookupLimits(context.Background(), llmprovider.ProviderClaude, llmprovider.NewStaticToken("k"),
				"claude-opus-4-6", opts...); err != nil {
				t.Errorf("LookupLimits: %v", err)
			}
		})
	}
	close(start)
	wg.Wait()
	if n := s.count("/v1/models"); n != 1 {
		t.Errorf("listing requests = %d; want 1", n)
	}
	l := CachedLimits(context.Background(), llmprovider.ProviderClaude, "claude-opus-4-6", append(opts, llmprovider.WithAPIKey("k"))...)
	wantFact(t, "cached output", l.Output, 128000, "listing:claude")
}

// TestDiscover_InvalidProvider_0089 (DL28): an id LookupModel refuses.
func TestDiscover_InvalidProvider_0089(t *testing.T) {
	_, err := LookupLimits(context.Background(), "nope", nil, "m")
	if !errors.Is(err, llmprovider.ErrInvalidProvider) {
		t.Errorf("err = %v; want ErrInvalidProvider", err)
	}
}

// TestDiscover_WireFacts_0089 (DL6, DL7, DL11, DL30): Required on Claude and
// OpenCode's messages route, by the document's provider.npm where it lists
// the model, and on no other provider or route.
func TestDiscover_WireFacts_0089(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		id    llmprovider.ProviderID
		model string
		want  bool
	}{
		{llmprovider.ProviderClaude, "claude-opus-4-6", true},
		{llmprovider.ProviderOpencodeZen, "claude-opus-4-6", true},
		{llmprovider.ProviderOpencodeZen, "gpt-5.5", false},
		{llmprovider.ProviderOpencodeZen, "gemini-3-pro", false},
		{llmprovider.ProviderKilo, "anthropic/claude-opus-4.6", false},
		{llmprovider.ProviderGrok, "grok-5", false},
		{llmprovider.ProviderOpenAI, "gpt-5.5", false},
		{llmprovider.ProviderGemini, "gemini-3-pro", false},
		{llmprovider.ProviderTogether, "m", false},
		{llmprovider.ProviderHuggingFace, "m", false},
		{llmprovider.ProviderOllama, "m", false},
	} {
		if got := CachedLimits(ctx, tc.id, tc.model).Required; got != tc.want {
			t.Errorf("%s %s: Required = %v; want %v", tc.id, tc.model, got, tc.want)
		}
	}
	doc := docServer(t, `{"opencode":{"models":{"custom-m":{"limit":{"context":1000},"provider":{"npm":"@ai-sdk/anthropic"}}}}}`)
	_, c := limitsCache(t)
	if l := lookupLimitsT(t, llmprovider.ProviderOpencodeZen, nil, "custom-m", servedDoc(t, doc.URL), c); !l.Required {
		t.Error("a model the document routes to messages: Required false")
	}
}

// TestDiscover_RequiredWithoutDocument_0089 (DL74): with no document, an
// OpenCode model's route is read from the route table, as the provider
// routes it.
func TestDiscover_RequiredWithoutDocument_0089(t *testing.T) {
	ctx := context.Background()
	if !CachedLimits(ctx, llmprovider.ProviderOpencodeZen, "claude-opus-4-6").Required {
		t.Error("Zen claude-opus-4-6 with no document: Required false; want true")
	}
	if CachedLimits(ctx, llmprovider.ProviderOpencodeZen, "glm-5").Required {
		t.Error("Zen glm-5, a chat-route model: Required true; want false")
	}
}
