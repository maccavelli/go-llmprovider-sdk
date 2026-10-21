package catalog

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0033-PLAN-public-model-metadata.md Phase 3: the listings, the cache and the
// resolver (0033-MADR D2 and D3, as amended on 2026-10-12).

// lookupDoc is a models.dev document with an entry for each listing's model,
// so a test can tell the listing's fields from models.dev's.
const lookupDoc = `{
 "anthropic": {"models": {"claude-x": {"name": "Claude X", "reasoning": true, "tool_call": true, "structured_output": true,
   "modalities": {"input": ["text", "image", "pdf"]}, "limit": {"context": 1000000, "output": 128000},
   "cost": {"input": 3, "output": 15}, "release_date": "2026-09-28"}}},
 "google": {"models": {"gemini-x": {"name": "Gemini X", "limit": {"context": 1048576, "output": 1000}, "cost": {"input": 0.3, "output": 2.5}}}},
 "xai": {"models": {"grok-x": {"name": "Grok X", "limit": {"context": 100, "output": 50}, "cost": {"input": 9, "output": 9}}}},
 "kilo": {"models": {"vendor/kilo-x": {"name": "Kilo X", "limit": {"context": 100, "output": 50}, "cost": {"input": 9, "output": 0.6}}}},
 "togetherai": {"models": {"org/together-x": {"name": "Together X", "limit": {"context": 100, "output": 50}, "cost": {"input": 9, "output": 9}}}},
 "openai": {"models": {"gpt-x": {"name": "GPT X", "limit": {"context": 1050000, "output": 128000}, "cost": {"input": 2, "output": 8}}}}
}`

// The listings' replies, one model each.
const (
	claudeListing0033 = `{"data": [{"id": "claude-x", "max_input_tokens": 900000, "max_tokens": 100000,
	 "lifecycle": "deprecated", "retires_at": "2026-11-30T00:00:00Z",
	 "capabilities": {"image_input": {"supported": true}, "pdf_input": {"supported": false},
	  "structured_outputs": {"supported": false},
	  "thinking": {"supported": true, "types": {"adaptive": {"supported": true}, "disabled": {"supported": false}}},
	  "effort": {"supported": true, "low": {"supported": true}, "medium": {"supported": false}, "high": {"supported": true}}}}],
	 "has_more": false}`
	geminiListing0033 = `{"models": [{"name": "models/gemini-x", "supportedGenerationMethods": ["generateContent"],
	 "inputTokenLimit": 1048576, "outputTokenLimit": 65536, "thinking": true}]}`
	grokListing0033 = `{"data": [{"id": "grok-x", "context_length": 500000,
	 "capabilities": {"reasoning_effort": ["low", "high"], "default_reasoning_effort": "high"},
	 "prompt_text_token_price": 20000, "cached_prompt_text_token_price": 5000, "completion_text_token_price": 60000,
	 "prompt_text_token_price_long_context": 40000, "cached_prompt_text_token_price_long_context": 10000,
	 "completion_text_token_price_long_context": 120000, "long_context_threshold": 200000}]}`
	kiloListing0033 = `{"data": [{"id": "vendor/kilo-x", "context_length": 131072, "top_provider": {"max_completion_tokens": 32768},
	 "pricing": {"prompt": "0.000002", "completion": "-1", "input_cache_read": "0.0000002",
	  "input_cache_write": "0.0000025", "internal_reasoning": "0.000003", "discount": 0.5},
	 "expiration_date": "2026-12-01"}]}`
	togetherListing0033 = `[{"id": "org/together-x", "type": "chat", "context_length": 1048575,
	 "pricing": {"input": 0.3, "output": 1.2, "cached_input": 0.006, "hourly": 0}}]`
	openaiListing0033 = `{"data": [{"id": "gpt-x", "shutdown_date": "2026-10-23"}]}`
	ollamaShow0033    = `{"model_info": {"general.architecture": "llama", "llama.context_length": 131072},
	 "capabilities": ["completion", "tools", "vision"]}`
)

// lookupServer serves the document at /api.json and reply at every other
// path, counting the listing requests and the credentials they carry.
type lookupServer struct {
	*httptest.Server
	listings atomic.Int32
	mu       sync.Mutex
	auths    []string
	block    chan struct{} // when set, the listing waits for it to close
}

func newLookupServer(t *testing.T, doc string, status int, reply string) *lookupServer {
	t.Helper()
	s := &lookupServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api.json" {
			_, _ = io.WriteString(w, doc)
			return
		}
		s.listings.Add(1)
		s.mu.Lock()
		s.auths = append(s.auths, r.Header.Get("Authorization")+r.Header.Get("x-api-key")+r.Header.Get("x-goog-api-key"))
		block := s.block
		s.mu.Unlock()
		if block != nil {
			<-block
		}
		w.WriteHeader(status)
		_, _ = io.WriteString(w, reply)
	}))
	t.Cleanup(s.Close)
	return s
}

// waitListingFetches waits for the background listing fetches.
func waitListingFetches() { processStore.listingFetching.Wait() }

// resetListingCache empties the listing cache, after any fetch in flight.
func resetListingCache() {
	waitListingFetches()
	processStore.listingMu.Lock()
	defer processStore.listingMu.Unlock()
	clear(processStore.listing)
	clear(processStore.listingInFlight)
}

// resetLookupCaches empties both caches on both sides of a test.
func resetLookupCaches(t *testing.T) {
	t.Helper()
	enableModelMetadata(t)
	resetListingCache()
	t.Cleanup(resetListingCache)
}

// lookupOpts point the listing and the document at s.
func lookupOpts(s *lookupServer) []llmprovider.Option {
	return []llmprovider.Option{llmprovider.WithBaseURL(s.URL), llmprovider.WithModelMetadataURL(s.URL + "/api.json")}
}

// listingCase is one listing's reply, model, and the fields it must set.
type listingCase struct {
	provider llmprovider.ProviderID
	model    string
	reply    string
	fields   []Field
	check    func(t *testing.T, info ModelInfo)
}

var listingCases = []listingCase{
	{llmprovider.ProviderClaude, "claude-x", claudeListing0033,
		[]Field{FieldMaxInputTokens, FieldMaxOutputTokens, FieldInput, FieldStructuredOutput, FieldReasoning,
			FieldReasoningToggle, FieldReasoningEfforts},
		func(t *testing.T, i ModelInfo) {
			if i.MaxInputTokens != 900000 || i.MaxOutputTokens != 100000 || !slices.Equal(i.Input, []Modality{ModalityText, ModalityImage}) ||
				i.StructuredOutput || !i.Reasoning.Supported || i.Reasoning.Toggle ||
				!slices.Equal(i.Reasoning.Efforts, []llmprovider.Effort{"low", "high"}) {
				t.Errorf("claude: %+v", i.public())
			}
		}},
	{llmprovider.ProviderGemini, "gemini-x", geminiListing0033,
		[]Field{FieldMaxInputTokens, FieldMaxOutputTokens, FieldReasoning},
		func(t *testing.T, i ModelInfo) {
			if i.MaxInputTokens != 1048576 || i.MaxOutputTokens != 65536 || !i.Reasoning.Supported {
				t.Errorf("gemini: %+v", i.public())
			}
		}},
	{llmprovider.ProviderGrok, "grok-x", grokListing0033,
		[]Field{FieldContextWindow, FieldReasoningEfforts, FieldCostInput, FieldCostOutput, FieldCostCacheRead, FieldCostTiers},
		func(t *testing.T, i ModelInfo) {
			want := ModelCost{Input: 2, Output: 6, CacheRead: 0.5,
				Tiers: []ModelCostTier{{Above: 199999, Input: 4, Output: 12, CacheRead: 1}}}
			if i.ContextWindow != 500000 || !slices.Equal(i.Reasoning.Efforts, []llmprovider.Effort{"low", "high"}) ||
				!costEqual(i.Cost, want) {
				t.Errorf("grok: %+v, want cost %+v", i.public(), want)
			}
		}},
	{llmprovider.ProviderKilo, "vendor/kilo-x", kiloListing0033,
		[]Field{FieldContextWindow, FieldMaxOutputTokens, FieldCostInput, FieldCostCacheRead, FieldCostCacheWrite,
			FieldCostReasoning, FieldRetires},
		func(t *testing.T, i ModelInfo) {
			if i.ContextWindow != 131072 || i.MaxOutputTokens != 32768 || !near(i.Cost.Input, 2) || !near(i.Cost.CacheRead, 0.2) ||
				!near(i.Cost.CacheWrite, 2.5) || !near(i.Cost.Reasoning, 3) ||
				!i.Retires.Equal(time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)) {
				t.Errorf("kilo: %+v", i.public())
			}
		}},
	{llmprovider.ProviderTogether, "org/together-x", togetherListing0033,
		[]Field{FieldContextWindow, FieldCostInput, FieldCostOutput, FieldCostCacheRead},
		func(t *testing.T, i ModelInfo) {
			if i.ContextWindow != 1048575 || !near(i.Cost.Input, 0.3) || !near(i.Cost.Output, 1.2) || !near(i.Cost.CacheRead, 0.006) {
				t.Errorf("together: %+v", i.public())
			}
		}},
	{llmprovider.ProviderOpenAI, "gpt-x", openaiListing0033,
		[]Field{FieldRetires},
		func(t *testing.T, i ModelInfo) {
			if !i.Retires.Equal(time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC)) {
				t.Errorf("openai: Retires %v", i.Retires)
			}
		}},
	{llmprovider.ProviderOllama, "llama3.2:latest", ollamaShow0033,
		[]Field{FieldContextWindow, FieldTools, FieldInput, FieldOutput, FieldReasoning},
		func(t *testing.T, i ModelInfo) {
			if i.ContextWindow != 131072 || !i.Tools || !slices.Equal(i.Input, []Modality{ModalityText, ModalityImage}) ||
				!slices.Equal(i.Output, []Modality{ModalityText}) || i.Reasoning.Supported {
				t.Errorf("ollama: %+v", i.public())
			}
		}},
}

func near(a, b float64) bool { return a-b < 1e-9 && b-a < 1e-9 }

func costEqual(a, b ModelCost) bool {
	if !near(a.Input, b.Input) || !near(a.Output, b.Output) || !near(a.CacheRead, b.CacheRead) || len(a.Tiers) != len(b.Tiers) {
		return false
	}
	for i := range a.Tiers {
		x, y := a.Tiers[i], b.Tiers[i]
		if x.Above != y.Above || !near(x.Input, y.Input) || !near(x.Output, y.Output) || !near(x.CacheRead, y.CacheRead) {
			return false
		}
	}
	return true
}

// lookup runs LookupModel against a server replying with c's listing.
func lookup(t *testing.T, c listingCase, extra ...llmprovider.Option) (ModelInfo, error) {
	t.Helper()
	s := newLookupServer(t, lookupDoc, http.StatusOK, c.reply)
	return LookupModel(context.Background(), c.provider, llmprovider.NewStaticToken("key-1"), c.model,
		append(lookupOpts(s), extra...)...)
}

// TestLookupModel_Listings_0033: each listing sets the fields amended D2
// gives it, sourced listing:<provider>; models.dev sets the rest.
func TestLookupModel_Listings_0033(t *testing.T) {
	for _, c := range listingCases {
		t.Run(string(c.provider), func(t *testing.T) {
			resetLookupCaches(t)
			info, err := lookup(t, c)
			if err != nil {
				t.Fatalf("LookupModel: %v", err)
			}
			for _, f := range c.fields {
				if src := info.SourceOf(f); src != "listing:"+string(c.provider) {
					t.Errorf("field %d: source %q, want listing:%s", f, src, c.provider)
				}
			}
			if c.provider != llmprovider.ProviderOllama {
				if src := info.SourceOf(FieldName); src != "models.dev" {
					t.Errorf("Name: source %q, want models.dev", src)
				}
			}
			c.check(t, info)
		})
	}
}

// TestLookupModel_AnthropicLifecycle_0033: Anthropic's lifecycle and
// retires_at set nothing; models.dev's status stands ("sources that lag").
func TestLookupModel_AnthropicLifecycle_0033(t *testing.T) {
	resetLookupCaches(t)
	info, err := lookup(t, listingCases[0])
	if err != nil {
		t.Fatal(err)
	}
	if info.Status != StatusActive || info.SourceOf(FieldStatus) != "models.dev" || info.Has(FieldRetires) {
		t.Errorf("Status %q from %q, Retires set %v; want active from models.dev, no Retires", info.Status, info.SourceOf(FieldStatus), info.Has(FieldRetires))
	}
}

// TestLookupModel_Units_0033: xAI's cents per 100 million and threshold,
// Kilo's per-token strings, its -1, and Together's USD per million.
func TestLookupModel_Units_0033(t *testing.T) {
	resetLookupCaches(t)
	grok, err := lookup(t, listingCases[2])
	if err != nil || !near(grok.Cost.Input, 2) || len(grok.Cost.Tiers) != 1 || grok.Cost.Tiers[0].Above != 199999 {
		t.Errorf("xAI: %v, cost %+v; want 2.0 and a tier Above 199999", err, grok.Cost)
	}
	kilo, err := lookup(t, listingCases[3])
	if err != nil || !near(kilo.Cost.Input, 2) || kilo.SourceOf(FieldCostOutput) != "models.dev" || !near(kilo.Cost.Output, 0.6) {
		t.Errorf("Kilo: %v, Input %v, Output %v from %q; want 2.0, and the -1 output left to models.dev's 0.6",
			err, kilo.Cost.Input, kilo.Cost.Output, kilo.SourceOf(FieldCostOutput))
	}
	together, err := lookup(t, listingCases[4])
	if err != nil || !near(together.Cost.Input, 0.3) {
		t.Errorf("Together: %v, Input %v; want 0.3", err, together.Cost.Input)
	}
}

// TestLookupModel_Retires_0033: OpenAI's shutdown_date and Kilo's
// expiration_date set Retires.
func TestLookupModel_Retires_0033(t *testing.T) {
	resetLookupCaches(t)
	for _, c := range []listingCase{listingCases[5], listingCases[3]} {
		info, err := lookup(t, c)
		if err != nil || !info.Has(FieldRetires) || info.SourceOf(FieldRetires) != "listing:"+string(c.provider) {
			t.Errorf("%s: %v, Retires %v from %q", c.provider, err, info.Retires, info.SourceOf(FieldRetires))
		}
	}
}

// staticLayer is a caller's layer: one model's values, the fields it marks.
type staticLayer struct {
	name string
	info ModelInfo
}

func (l staticLayer) Name() string { return l.name }

func (l staticLayer) Model(_ llmprovider.ProviderID, id string) (ModelInfo, bool) {
	if id != l.info.ID {
		return ModelInfo{}, false
	}
	return l.info, true
}

// TestLookupModel_CallerLayers_0033: a caller layer beats the listing and
// models.dev; the later of two wins; a field no layer sets stays unset; tiers
// are replaced whole; a known zero beats a listing's price.
func TestLookupModel_CallerLayers_0033(t *testing.T) {
	resetLookupCaches(t)
	first := ModelInfo{ID: "grok-x", ContextWindow: 1, Cost: ModelCost{ModelRates: ModelRates{Input: 0}}}.
		Known(FieldContextWindow, FieldCostInput)
	second := ModelInfo{ID: "grok-x", ContextWindow: 2,
		Cost: ModelCost{Tiers: []ModelCostTier{{Above: 100, Input: 7}}}}.
		Known(FieldContextWindow, FieldCostTiers)
	info, err := lookup(t, listingCases[2], WithModelLayer(staticLayer{"first", first}), WithModelLayer(staticLayer{"second", second}))
	if err != nil {
		t.Fatal(err)
	}
	if info.ContextWindow != 2 || info.SourceOf(FieldContextWindow) != "second" {
		t.Errorf("ContextWindow %d from %q, want 2 from the later layer", info.ContextWindow, info.SourceOf(FieldContextWindow))
	}
	if info.Cost.Input != 0 || info.SourceOf(FieldCostInput) != "first" {
		t.Errorf("Cost.Input %v from %q, want the known zero from the caller", info.Cost.Input, info.SourceOf(FieldCostInput))
	}
	if len(info.Cost.Tiers) != 1 || info.Cost.Tiers[0].Above != 100 {
		t.Errorf("tiers %+v, want the caller's list whole", info.Cost.Tiers)
	}
	if info.Has(FieldMaxInputTokens) {
		t.Error("a field no layer sets is set")
	}
}

// TestLookupModel_Unknown_0033: a model no layer knows is not an error; a
// listing that fails leaves models.dev's fields and an error naming it.
func TestLookupModel_Unknown_0033(t *testing.T) {
	resetLookupCaches(t)
	info, err := lookup(t, listingCase{provider: llmprovider.ProviderClaude, model: "nobody", reply: claudeListing0033})
	if err != nil || info.ID != "nobody" || info.known != 0 {
		t.Errorf("unknown model: %+v, %v; want no error, nothing set", info.public(), err)
	}
	s := newLookupServer(t, lookupDoc, http.StatusInternalServerError, `{"error":{"message":"down"}}`)
	info, err = LookupModel(context.Background(), llmprovider.ProviderClaude, llmprovider.NewStaticToken("key-1"), "claude-x", lookupOpts(s)...)
	if err == nil || !strings.Contains(err.Error(), "listing claude") || info.ContextWindow != 1000000 {
		t.Errorf("a failed listing: window %d, err %v; want models.dev's window and an error naming the listing", info.ContextWindow, err)
	}
}

// requestCounter0033 counts requests and fails each.
type requestCounter0033 struct{ n atomic.Int32 }

func (c *requestCounter0033) RoundTrip(*http.Request) (*http.Response, error) {
	c.n.Add(1)
	return nil, errors.New("no request allowed")
}

// TestLookupModel_Offline_0033: no metadata fetch, a parsed document as a
// layer, no source: no request, and the document's fields.
func TestLookupModel_Offline_0033(t *testing.T) {
	resetLookupCaches(t)
	f, err := os.Open(fixture0033)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	doc, err := ParseMetadata(f)
	if err != nil {
		t.Fatal(err)
	}
	ct := &requestCounter0033{}
	info, err := LookupModel(context.Background(), llmprovider.ProviderClaude, nil, "claude-x",
		llmprovider.WithoutModelMetadata(), llmprovider.WithHTTPClient(&http.Client{Transport: ct}), WithModelLayer(doc))
	if err != nil || ct.n.Load() != 0 || info.ContextWindow != 1000000 || info.SourceOf(FieldContextWindow) != "models.dev" {
		t.Errorf("offline: %d requests, window %d from %q, %v", ct.n.Load(), info.ContextWindow, info.SourceOf(FieldContextWindow), err)
	}
}

// TestCachedModel_NeverWaits_0033: cold, it returns at once and starts the
// fetches; once they end it answers; warm, it sends nothing.
func TestCachedModel_NeverWaits_0033(t *testing.T) {
	resetLookupCaches(t)
	s := newLookupServer(t, lookupDoc, http.StatusOK, grokListing0033)
	s.block = make(chan struct{})
	src := llmprovider.NewStaticToken("key-1")
	start := time.Now()
	_, ok := CachedModel(context.Background(), llmprovider.ProviderGrok, src, "grok-x", lookupOpts(s)...)
	if took := time.Since(start); ok || took > 50*time.Millisecond {
		t.Errorf("cold: ok %v after %v; want false within 50 ms", ok, took)
	}
	s.mu.Lock()
	close(s.block)
	s.block = nil
	s.mu.Unlock()
	waitListingFetches()
	waitMetadataFetches()
	info, ok := CachedModel(context.Background(), llmprovider.ProviderGrok, src, "grok-x", lookupOpts(s)...)
	if !ok || info.ContextWindow != 500000 || info.Name != "Grok X" {
		t.Errorf("after the fetch: %+v, %v; want the listing's window and models.dev's name", info.public(), ok)
	}
	before := s.listings.Load()
	if _, ok := CachedModel(context.Background(), llmprovider.ProviderGrok, src, "grok-x", lookupOpts(s)...); !ok || s.listings.Load() != before {
		t.Errorf("warm: ok %v, %d new listing requests; want none", ok, s.listings.Load()-before)
	}
}

// TestListingCache_Keys_0033: one listing per provider, base URL and
// credential, for ten minutes.
func TestListingCache_Keys_0033(t *testing.T) {
	resetLookupCaches(t)
	s := newLookupServer(t, lookupDoc, http.StatusOK, grokListing0033)
	get := func(key string, srv *lookupServer) {
		t.Helper()
		if _, err := LookupModel(context.Background(), llmprovider.ProviderGrok, llmprovider.NewStaticToken(key), "grok-x", lookupOpts(srv)...); err != nil {
			t.Fatal(err)
		}
	}
	get("key-1", s)
	get("key-1", s)
	if n := s.listings.Load(); n != 1 {
		t.Errorf("the same key twice: %d listing requests, want 1", n)
	}
	get("key-2", s)
	if n := s.listings.Load(); n != 2 {
		t.Errorf("another credential: %d listing requests, want 2", n)
	}
	other := newLookupServer(t, lookupDoc, http.StatusOK, grokListing0033)
	get("key-1", other)
	if n := other.listings.Load(); n != 1 {
		t.Errorf("another base URL: %d listing requests there, want 1", n)
	}
	saved := listingNow
	t.Cleanup(func() { listingNow = saved })
	listingNow = func() time.Time { return saved().Add(11 * time.Minute) }
	get("key-1", s)
	if n := s.listings.Load(); n != 3 {
		t.Errorf("after ten minutes: %d listing requests, want 3", n)
	}
}

// TestLookupModel_Bounds_0033: a cancelled context (R40), a bounded error
// with no credential (R35), and an unknown provider.
func TestLookupModel_Bounds_0033(t *testing.T) {
	resetLookupCaches(t)
	s := newLookupServer(t, lookupDoc, http.StatusOK, grokListing0033)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := LookupModel(ctx, llmprovider.ProviderGrok, llmprovider.NewStaticToken("key-1"), "grok-x", lookupOpts(s)...); !errors.Is(err, context.Canceled) {
		t.Errorf("a cancelled context: %v, want context.Canceled", err)
	}
	const secret = "sk-secret-credential-0033"
	big := newLookupServer(t, lookupDoc, http.StatusBadRequest, `{"error":{"message":"`+strings.Repeat("x", 100<<10)+secret+`"}}`)
	_, err := LookupModel(context.Background(), llmprovider.ProviderGrok, llmprovider.NewStaticToken(secret), "grok-x", lookupOpts(big)...)
	if err == nil || len(err.Error()) > 512 || strings.Contains(err.Error(), secret) {
		t.Errorf("a 100 KiB error body: %d bytes, credential present %v; want at most 512, none", len(errText(err)), strings.Contains(errText(err), secret))
	}
	if _, err := LookupModel(context.Background(), "no-such-provider", nil, "m"); !errors.Is(err, llmprovider.ErrInvalidProvider) {
		t.Errorf("an unknown provider: %v, want ErrInvalidProvider", err)
	}
}
