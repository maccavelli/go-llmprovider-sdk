package catalog

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0033-PLAN-public-model-metadata.md Phase 2: ModelInfo, the models.dev layer
// and the document (0033-MADR D1 and D3, as amended on 2026-10-12).

const fixture0033 = "testdata/metadata-0033/fixture.json"

// fixtureMetadata0033 is the fixture document, parsed.
func fixtureMetadata0033(t *testing.T) Metadata {
	t.Helper()
	f, err := os.Open(fixture0033)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	m, err := ParseMetadata(f)
	if err != nil {
		t.Fatalf("ParseMetadata: %v", err)
	}
	return m
}

// model0033 is one fixture entry, which must exist.
func model0033(t *testing.T, m Metadata, p llmprovider.ProviderID, id string) ModelInfo {
	t.Helper()
	info, ok := m.Model(p, id)
	if !ok {
		t.Fatalf("Model(%s, %s): not found", p, id)
	}
	return info
}

// all is every section the document has, decoded, by key.
func (d *modelMetadataDoc) all() map[string]map[string]modelMetadata {
	out := map[string]map[string]modelMetadata{}
	if d == nil {
		return out
	}
	for key := range d.sections {
		if m, ok := d.models(key); ok {
			out[key] = m
		}
	}
	return out
}

// TestLazySection_DecodesOnce_0033 (Phase 2's deviation): a section is
// decoded the first time it is read, once, however many read it at once, and
// a section that does not decode is absent.
func TestLazySection_DecodesOnce_0033(t *testing.T) {
	doc, err := decodeModelMetadata(strings.NewReader(`{"opencode":{"models":{"a":{"name":"A"}}},"kilo":{"models":{"b":{"limit":"wide"}}}}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if doc.sections[metadataKeyZen].models != nil {
		t.Fatal("the section was decoded before it was read")
	}
	done := make(chan bool)
	for range 8 {
		go func() { _, ok := doc.models(metadataKeyZen); done <- ok }()
	}
	for range 8 {
		if !<-done {
			t.Error("a concurrent read found no section")
		}
	}
	if _, ok := doc.models(metadataKeyKilo); ok {
		t.Error("a section that does not decode is present")
	}
	if _, err := ParseMetadata(strings.NewReader(`{"kilo":{"models":{"b":{"limit":"wide"}}}}`)); err == nil {
		t.Error("ParseMetadata accepted a section that does not decode")
	}
}

// public is m without its presence and sources, to compare values alone.
func (m ModelInfo) public() ModelInfo {
	m.known, m.sources, m.layer = 0, [fieldCount]string{}, ""
	return m
}

// TestModelInfo_Decodes_0033: every D1 field of one full entry, and one entry
// from each of the nine sections, by its provider.
func TestModelInfo_Decodes_0033(t *testing.T) {
	m := fixtureMetadata0033(t)
	got := model0033(t, m, llmprovider.ProviderClaude, "claude-x")
	want := ModelInfo{
		ID: "claude-x", Name: "Claude X", Family: "claude-x",
		ContextWindow: 1000000, MaxOutputTokens: 128000,
		Input:  []Modality{ModalityText, ModalityImage, ModalityPDF},
		Output: []Modality{ModalityText},
		Tools:  true, Temperature: true, StructuredOutput: true,
		Reasoning: ModelReasoning{Supported: true, Efforts: []llmprovider.Effort{"none", "low", "high", "max"}},
		Cost: ModelCost{
			ModelRates: ModelRates{Input: 3, Output: 15, CacheRead: 0.3, CacheWrite: 3.75},
			Tiers:      []ModelCostTier{{Above: 200000, Input: 6, Output: 22.5, CacheRead: 0.6, CacheWrite: 7.5}},
		},
		Released: time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC),
		Status:   StatusActive,
	}
	if !reflect.DeepEqual(got.public(), want.public()) {
		t.Errorf("claude-x:\n got %+v\nwant %+v", got.public(), want.public())
	}
	for p, id := range map[llmprovider.ProviderID]string{
		llmprovider.ProviderOpenAI: "gpt-x", llmprovider.ProviderGemini: "gemini-x", llmprovider.ProviderGrok: "grok-x",
		llmprovider.ProviderKilo: "vendor/kilo-x", llmprovider.ProviderOpencodeZen: "zen-x", llmprovider.ProviderOpencodeGo: "go-x",
		llmprovider.ProviderHuggingFace: "org/hf-x", llmprovider.ProviderTogether: "org/together-x",
	} {
		if info := model0033(t, m, p, id); info.ID != id || !info.Has(FieldContextWindow) {
			t.Errorf("%s %s: %+v, want the entry with its window", p, id, info.public())
		}
	}
	if gpt := model0033(t, m, llmprovider.ProviderOpenAI, "gpt-x"); gpt.MaxInputTokens != 922000 || !gpt.Has(FieldMaxInputTokens) {
		t.Errorf("gpt-x MaxInputTokens = %d (set %v), want 922000", gpt.MaxInputTokens, gpt.Has(FieldMaxInputTokens))
	}
	if zen := model0033(t, m, llmprovider.ProviderOpencodeZen, "zen-x"); zen.Cost.Reasoning != 2.5 || !zen.Has(FieldCostReasoning) {
		t.Errorf("zen-x Cost.Reasoning = %v (set %v), want 2.5", zen.Cost.Reasoning, zen.Has(FieldCostReasoning))
	}
	if _, ok := m.Model(llmprovider.ProviderOllama, "m"); ok {
		t.Error("ollama has no section, but Model found an entry")
	}
}

// TestModelInfo_Presence_0033: a zero price is known; an absent one is not;
// an entry without a status is active; SourceOf names models.dev, or nothing.
func TestModelInfo_Presence_0033(t *testing.T) {
	hf := model0033(t, fixtureMetadata0033(t), llmprovider.ProviderHuggingFace, "org/hf-x")
	if !hf.Has(FieldCostInput) || hf.Cost.Input != 0 {
		t.Errorf("a price of 0: Has %v, Input %v; want known, 0", hf.Has(FieldCostInput), hf.Cost.Input)
	}
	if hf.Has(FieldCostCacheWrite) {
		t.Error("an absent cache_write is known")
	}
	if !hf.Has(FieldStatus) || hf.Status != StatusActive {
		t.Errorf("no status: Has %v, Status %q; want StatusActive, set", hf.Has(FieldStatus), hf.Status)
	}
	if hf.Has(FieldMaxInputTokens) || hf.Has(FieldRetires) {
		t.Error("an absent limit.input, or a retirement models.dev never gives, is known")
	}
	if s := hf.SourceOf(FieldCostInput); s != "models.dev" {
		t.Errorf("SourceOf(FieldCostInput) = %q, want models.dev", s)
	}
	if s := hf.SourceOf(FieldCostCacheWrite); s != "" {
		t.Errorf("SourceOf an unset field = %q, want empty", s)
	}
	if hf.Tools || !hf.Has(FieldTools) {
		t.Errorf("tool_call false: Tools %v, Has %v; want false, known", hf.Tools, hf.Has(FieldTools))
	}
}

// TestModelInfo_Dates_0033: both date forms, and one that is neither.
func TestModelInfo_Dates_0033(t *testing.T) {
	m := fixtureMetadata0033(t)
	if g := model0033(t, m, llmprovider.ProviderGemini, "gemini-x"); !g.Released.Equal(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)) || !g.Has(FieldReleased) {
		t.Errorf(`"2026-09": %v, set %v; want 2026-09-01`, g.Released, g.Has(FieldReleased))
	}
	if c := model0033(t, m, llmprovider.ProviderClaude, "claude-x"); !c.Released.Equal(time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)) {
		t.Errorf(`"2026-09-28": %v`, c.Released)
	}
	if g := model0033(t, m, llmprovider.ProviderOpencodeGo, "go-x"); g.Has(FieldReleased) || !g.Released.IsZero() {
		t.Errorf(`"soon": %v, set %v; want unset`, g.Released, g.Has(FieldReleased))
	}
	for p, want := range map[llmprovider.ProviderID]ModelStatus{
		llmprovider.ProviderGemini: StatusBeta, llmprovider.ProviderGrok: StatusDeprecated, llmprovider.ProviderOpencodeGo: StatusAlpha,
	} {
		for _, info := range m.Models(p) {
			if info.Status != want {
				t.Errorf("%s %s: Status %q, want %q", p, info.ID, info.Status, want)
			}
		}
	}
}

// TestModelInfo_Reasoning_0033: each reasoning_options type, and reasoning off.
func TestModelInfo_Reasoning_0033(t *testing.T) {
	m := fixtureMetadata0033(t)
	if r := model0033(t, m, llmprovider.ProviderTogether, "org/together-x"); !r.Reasoning.Toggle || !r.Has(FieldReasoningToggle) {
		t.Errorf("toggle: %+v, set %v", r.Reasoning, r.Has(FieldReasoningToggle))
	}
	c := model0033(t, m, llmprovider.ProviderClaude, "claude-x")
	if !slices.Equal(c.Reasoning.Efforts, []llmprovider.Effort{"none", "low", "high", "max"}) || !c.Has(FieldReasoningEfforts) {
		t.Errorf("effort, null first: %v, set %v; want none, low, high, max", c.Reasoning.Efforts, c.Has(FieldReasoningEfforts))
	}
	if c.Reasoning.Toggle || !c.Has(FieldReasoningToggle) {
		t.Errorf("options without a toggle: Toggle %v, set %v; want false, known", c.Reasoning.Toggle, c.Has(FieldReasoningToggle))
	}
	z := model0033(t, m, llmprovider.ProviderOpencodeZen, "zen-x")
	if z.Reasoning.BudgetMin != 1024 || z.Reasoning.BudgetMax != 32000 || !z.Has(FieldReasoningBudget) || !z.Reasoning.Toggle {
		t.Errorf("budget_tokens and toggle: %+v, set %v", z.Reasoning, z.Has(FieldReasoningBudget))
	}
	k := model0033(t, m, llmprovider.ProviderKilo, "vendor/kilo-x")
	if k.Reasoning.Supported || !k.Has(FieldReasoning) {
		t.Errorf("reasoning false: Supported %v, set %v; want false, known", k.Reasoning.Supported, k.Has(FieldReasoning))
	}
	for _, f := range []Field{FieldReasoningToggle, FieldReasoningEfforts, FieldReasoningBudget} {
		if k.Has(f) {
			t.Errorf("reasoning false: field %d is known", f)
		}
	}
}

// TestModelCost_Tiers_0033: tiers sort by Above; a lone context_over_200k is
// a tier at 200000; an omitted rate is the base rate.
func TestModelCost_Tiers_0033(t *testing.T) {
	m := fixtureMetadata0033(t)
	gpt := model0033(t, m, llmprovider.ProviderOpenAI, "gpt-x")
	want := []ModelCostTier{
		{Above: 200000, Input: 4, Output: 16, CacheRead: 0.2},
		{Above: 1000000, Input: 8, Output: 32, CacheRead: 0.2},
	}
	if !reflect.DeepEqual(gpt.Cost.Tiers, want) || !gpt.Has(FieldCostTiers) {
		t.Errorf("gpt-x tiers = %+v, want %+v (tiers read over context_over_200k, omitted cache rate the base's)", gpt.Cost.Tiers, want)
	}
	grok := model0033(t, m, llmprovider.ProviderGrok, "grok-x")
	if lone := []ModelCostTier{{Above: 200000, Input: 4, Output: 12, CacheRead: 1}}; !reflect.DeepEqual(grok.Cost.Tiers, lone) {
		t.Errorf("grok-x tiers = %+v, want %+v", grok.Cost.Tiers, lone)
	}
	if k := model0033(t, m, llmprovider.ProviderKilo, "vendor/kilo-x"); len(k.Cost.Tiers) != 0 || !k.Has(FieldCostTiers) {
		t.Errorf("a flat price: tiers %+v, set %v; want none, known", k.Cost.Tiers, k.Has(FieldCostTiers))
	}
}

// TestModelCost_RatesFor_0033: the highest tier the input exceeds, else the
// base rates.
func TestModelCost_RatesFor_0033(t *testing.T) {
	base, first, second := ModelRates{Input: 1}, ModelRates{Input: 2}, ModelRates{Input: 3}
	c := ModelCost{ModelRates: base, Tiers: []ModelCostTier{{Above: 200000, ModelRates: first}, {Above: 1000000, ModelRates: second}}}
	for _, tc := range []struct {
		tokens int
		want   ModelRates
	}{{0, base}, {200000, base}, {200001, first}, {1000000, first}, {1000001, second}} {
		if got := c.RatesFor(tc.tokens); got != tc.want {
			t.Errorf("RatesFor(%d) = %+v, want %+v", tc.tokens, got, tc.want)
		}
	}
}

// TestModelInfo_Known_0033: Known marks a copy, never its receiver, with the
// source of the layer that returned it.
func TestModelInfo_Known_0033(t *testing.T) {
	c := model0033(t, fixtureMetadata0033(t), llmprovider.ProviderClaude, "claude-x")
	if c.Has(FieldMaxInputTokens) {
		t.Fatal("the fixture's claude-x has no limit.input")
	}
	k := c.Known(FieldMaxInputTokens)
	if !k.Has(FieldMaxInputTokens) || c.Has(FieldMaxInputTokens) {
		t.Errorf("Known: copy %v, receiver %v; want true, false", k.Has(FieldMaxInputTokens), c.Has(FieldMaxInputTokens))
	}
	if s := k.SourceOf(FieldMaxInputTokens); s != "models.dev" {
		t.Errorf("SourceOf = %q, want the layer that returned it, models.dev", s)
	}
	var own ModelInfo
	own.ContextWindow = 8192
	if mine := own.Known(FieldContextWindow); !mine.Has(FieldContextWindow) || own.Has(FieldContextWindow) {
		t.Error("Known on a caller's value: the copy is unmarked, or the receiver marked")
	}
}

// TestMetadata_ModelsCopies_0033: Models is sorted by id, and a change to
// what it returns does not reach the next call (R29).
func TestMetadata_ModelsCopies_0033(t *testing.T) {
	m := fixtureMetadata0033(t)
	first := m.Models(llmprovider.ProviderClaude)
	if ids := []string{first[0].ID, first[1].ID}; !slices.Equal(ids, []string{"claude-a", "claude-x"}) {
		t.Fatalf("Models ids = %v, want sorted", ids)
	}
	x := first[1]
	x.Input[0], x.Reasoning.Efforts[0], x.Cost.Tiers[0].Above = "changed", "changed", 1
	again := m.Models(llmprovider.ProviderClaude)[1]
	if again.Input[0] != ModalityText || again.Reasoning.Efforts[0] != "none" || again.Cost.Tiers[0].Above != 200000 {
		t.Errorf("a change to Models' result reached the next call: %+v", again.public())
	}
	single, _ := m.Model(llmprovider.ProviderClaude, "claude-x")
	single.Input[0] = "changed"
	if again, _ := m.Model(llmprovider.ProviderClaude, "claude-x"); again.Input[0] != ModalityText {
		t.Error("a change to Model's result reached the next call")
	}
}

// TestMetadata_EncodeLossless_0033: Encode, then ParseMetadata, gives the
// same models; each section's bytes are those read, unknown fields included.
func TestMetadata_EncodeLossless_0033(t *testing.T) {
	m := fixtureMetadata0033(t)
	var buf bytes.Buffer
	if err := m.Encode(&buf); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	again, err := ParseMetadata(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("ParseMetadata(Encode): %v", err)
	}
	for _, p := range []llmprovider.ProviderID{llmprovider.ProviderClaude, llmprovider.ProviderOpenAI, llmprovider.ProviderGemini,
		llmprovider.ProviderGrok, llmprovider.ProviderKilo, llmprovider.ProviderOpencodeZen, llmprovider.ProviderOpencodeGo,
		llmprovider.ProviderHuggingFace, llmprovider.ProviderTogether} {
		if a, b := m.Models(p), again.Models(p); !reflect.DeepEqual(a, b) || len(a) == 0 {
			t.Errorf("%s: %d models before, %d after, or they differ", p, len(a), len(b))
		}
	}
	raw, err := os.ReadFile(fixture0033)
	if err != nil {
		t.Fatal(err)
	}
	var read, written map[string]json.RawMessage
	if err := json.Unmarshal(raw, &read); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(buf.Bytes(), &written); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"anthropic", "openai", "google", "xai", "kilo", "opencode", "opencode-go", "huggingface", "togetherai"} {
		if !bytes.Equal(read[key], written[key]) {
			t.Errorf("section %s: the encoded bytes differ from those read", key)
		}
	}
	if !bytes.Contains(written["anthropic"], []byte(`"future_field"`)) {
		t.Error(`"future_field" was lost`)
	}
	if _, ok := written["ignored-section"]; ok {
		t.Error("a section the SDK does not read was written")
	}
	if m.Name() != "models.dev" {
		t.Errorf("Name = %q, want models.dev", m.Name())
	}
}

// failingTransport fails every request, to prove none is made.
type failingTransport struct{ t *testing.T }

func (f failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	f.t.Error("ParseMetadata made a request")
	return nil, errors.New("no request allowed")
}

// TestParseMetadata_Bounds_0033: it reads only its reader, refuses a body
// over the limit and one that is not a document, with a bounded error.
func TestParseMetadata_Bounds_0033(t *testing.T) {
	saved := http.DefaultTransport
	http.DefaultTransport = failingTransport{t}
	defer func() { http.DefaultTransport = saved }()
	if _, err := ParseMetadata(strings.NewReader(`{"opencode":{"models":{"a":{"name":"A"}}}}`)); err != nil {
		t.Fatalf("a small document: %v", err)
	}
	huge := io.MultiReader(strings.NewReader(`{"opencode":`), io.LimitReader(repeatReader(' '), metadataLimit+1), strings.NewReader(`{}}`))
	for name, r := range map[string]io.Reader{
		"over the limit": huge, "not JSON": strings.NewReader("not a document"), "an array": strings.NewReader(`[1, 2]`),
	} {
		_, err := ParseMetadata(r)
		if err == nil || len(err.Error()) > 512 {
			t.Errorf("%s: err %v (%d bytes); want an error of at most 512 bytes", name, err, len(errText(err)))
		}
	}
}

// repeatReader is an endless run of one byte.
type repeatReader byte

func (b repeatReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = byte(b)
	}
	return len(p), nil
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// TestModelMetadataKey_0033: the five sections the amendment adds.
func TestModelMetadataKey_0033(t *testing.T) {
	for p, want := range map[llmprovider.ProviderID]string{
		llmprovider.ProviderOpenAI: "openai", llmprovider.ProviderClaude: "anthropic", llmprovider.ProviderGemini: "google",
		llmprovider.ProviderGrok: "xai", llmprovider.ProviderKilo: "kilo", llmprovider.ProviderOllama: "",
	} {
		if got := modelMetadataKey(p); got != want {
			t.Errorf("modelMetadataKey(%s) = %q, want %q", p, got, want)
		}
	}
}
