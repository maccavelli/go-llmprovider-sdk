package catalog

import (
	"encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"math"
	"slices"
	"sync"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Model metadata (MADR 0009 §2): a models.dev-format document the caller
// downloads, parses with ParseMetadata and hands in with WithMetadata. The
// SDK fetches none, caches none and embeds none (0093-MADR D1, D6, D7).
const (
	metadataKeyZen = "opencode"
	metadataKeyGo  = "opencode-go"
	metadataKeyHF  = "huggingface"
	// metadataKeyTogether is models.dev's key for Together AI (measured
	// 2026-09-30; 0017-REPORT).
	metadataKeyTogether = "togetherai"
	// The vendors' and Kilo's sections (0033-MADR D2; measured in
	// 0033-PLAN-public-model-metadata.md Phase 1).
	metadataKeyOpenAI    = "openai"
	metadataKeyAnthropic = "anthropic"
	metadataKeyGoogle    = "google"
	metadataKeyXAI       = "xai"
	metadataKeyKilo      = "kilo"
)

// modelCost is a models.dev price, USD per million tokens. Input and Output
// are required wherever cost is present (models.dev's schema), so a cost
// present is a known input and output price, a zero one included.
type modelCost struct {
	Input           float64         `json:"input"`
	Output          float64         `json:"output"`
	CacheRead       *float64        `json:"cache_read"`
	CacheWrite      *float64        `json:"cache_write"`
	Reasoning       *float64        `json:"reasoning"`
	Tiers           []modelCostTier `json:"tiers"`
	ContextOver200k *modelCostRates `json:"context_over_200k"`
}

// modelCostRates are one tier's prices. An omitted one is the base price.
type modelCostRates struct {
	Input      *float64 `json:"input"`
	Output     *float64 `json:"output"`
	CacheRead  *float64 `json:"cache_read"`
	CacheWrite *float64 `json:"cache_write"`
	Reasoning  *float64 `json:"reasoning"`
}

// modelCostTier is one entry of cost.tiers: rates for inputs past tier.size.
type modelCostTier struct {
	modelCostRates
	Tier struct {
		Type string `json:"type"`
		Size int    `json:"size"`
	} `json:"tier"`
}

// modelMetadata is the subset of one models.dev model entry the SDK reads:
// for ranking, and for ModelInfo (0033-MADR D1).
type modelMetadata struct {
	Name   string `json:"name"`
	Family string `json:"family"`
	// CanonicalModelID is the same model's id in its maker's section.
	CanonicalModelID string     `json:"canonical_model_id"`
	Reasoning        *bool      `json:"reasoning"`
	Cost             *modelCost `json:"cost"`
	// Limit.Context is required by models.dev's schema; a window of 0 is
	// read as unknown.
	Limit struct {
		Context int  `json:"context"`
		Input   *int `json:"input"`
		Output  *int `json:"output"`
	} `json:"limit"`
	Modalities *struct {
		Input  []string `json:"input"`
		Output []string `json:"output"`
	} `json:"modalities"`
	ToolCall         *bool                  `json:"tool_call"`
	Temperature      *bool                  `json:"temperature"`
	StructuredOutput *bool                  `json:"structured_output"`
	ReleaseDate      string                 `json:"release_date"`
	Status           string                 `json:"status"`
	ReasoningOptions []modelReasoningOption `json:"reasoning_options"`
	// Interleaved is {"field": name} when the provider expects prior reasoning
	// replayed on assistant messages under that field, or true/absent.
	Interleaved json.RawMessage `json:"interleaved"`
	// Provider.NPM names the AI SDK package OpenCode's client uses for the
	// model, which fixes its route on the Zen and Go gateways (MADR 0012
	// §3.1). Unset means the section's openai-compatible package.
	Provider struct {
		NPM string `json:"npm"`
	} `json:"provider"`
}

// modelReasoningOption is one models.dev reasoning_options entry: a toggle,
// effort values (a null one decodes as ""), or a budget range.
type modelReasoningOption struct {
	Type   string   `json:"type"`
	Values []string `json:"values"`
	Min    *int     `json:"min"`
	Max    *int     `json:"max"`
}

// modelMetadataSection is one provider's entry in the document.
type modelMetadataSection struct {
	Models map[string]modelMetadata `json:"models"`
}

// modelMetadataDoc is a parsed document: each section the SDK
// reads, by key, kept as read, for a lossless Encode (0033-MADR D3, as
// amended), and decoded the first time it is read. A caller pays only for the
// sections it reads (0033-PLAN-public-model-metadata.md, Phase 2's
// deviation). A nil *modelMetadataDoc is an empty document.
type modelMetadataDoc struct {
	sections map[string]*lazySection
	// canonical is the document's facts by canonical id, for a chatcompat
	// server's models (canonical.go).
	canonical canonicalIndex
}

// lazySection is one section's bytes, and its models once decoded. A key the
// document repeats keeps each occurrence's bytes, in order, and they merge
// when decoded, as encoding/json merges them (0028-PLAN D12).
type lazySection struct {
	raws   []jsontext.Value
	once   sync.Once
	models map[string]modelMetadata
	err    error
}

// decode decodes the section, once, however many readers ask at once (R20).
func (s *lazySection) decode() (map[string]modelMetadata, error) {
	s.once.Do(func() {
		var section modelMetadataSection
		for _, v := range s.raws {
			if err := unmarshalSection(v, &section); err != nil {
				s.err = err
				return
			}
		}
		s.models = section.Models
	})
	return s.models, s.err
}

// models returns the section key's models, and whether the document has it.
// A section without models, or one that does not decode, is absent.
func (d *modelMetadataDoc) models(key string) (map[string]modelMetadata, bool) {
	if d == nil || d.sections[key] == nil {
		return nil, false
	}
	m, err := d.sections[key].decode()
	return m, err == nil && m != nil
}

// decodeAll decodes every section, and returns the first one that fails.
func (d *modelMetadataDoc) decodeAll() error {
	if d == nil {
		return nil
	}
	for key, s := range d.sections {
		if _, err := s.decode(); err != nil {
			return fmt.Errorf("model metadata: decode %s: %w", key, err)
		}
	}
	return nil
}

// entry returns one model's entry, and whether the document lists it.
func (d *modelMetadataDoc) entry(provider llmprovider.ProviderID, model string) (modelMetadata, bool) {
	models, _ := d.models(modelMetadataKey(provider))
	m, ok := models[model]
	return m, ok
}

// reasoningEfforts returns the effort values the document lists for one
// model's reasoning_options, or nil.
func (d *modelMetadataDoc) reasoningEfforts(provider llmprovider.ProviderID, model string) []string {
	m, _ := d.entry(provider, model)
	for _, o := range m.ReasoningOptions {
		if o.Type == jsonKeyEffort {
			// Every reader shares the document; the caller gets its own
			// copy (R29; 0026-MADR F53).
			return slices.Clone(o.Values)
		}
	}
	return nil
}

// interleavedField returns the message field a model expects its prior
// reasoning replayed under ("reasoning_content"), or "" when it declares none
// (MADR 0012 §2, O5).
func (d *modelMetadataDoc) interleavedField(provider llmprovider.ProviderID, model string) string {
	m, _ := d.entry(provider, model)
	return interleavedFieldOf(m)
}

// interleavedFieldOf is an entry's interleaved.field, or "" when it
// declares none: interleaved is true, absent, or has no field.
func interleavedFieldOf(m modelMetadata) string {
	var declared struct {
		Field string `json:"field"`
	}
	if json.Unmarshal(m.Interleaved, &declared) != nil {
		return ""
	}
	return declared.Field
}

// npm returns a model's provider.npm, the AI SDK package OpenCode's client
// routes it by, and false when the document does not list the model (MADR
// 0012 §3.1).
func (d *modelMetadataDoc) npm(provider llmprovider.ProviderID, model string) (string, bool) {
	m, ok := d.entry(provider, model)
	if !ok {
		return "", false
	}
	return m.Provider.NPM, true
}

// Metadata is a models.dev-format document the caller downloaded and parsed
// with ParseMetadata, and hands in with WithMetadata (0093-MADR D1): its age
// and its source are the caller's. A provider reads it for one request:
// reasoning efforts, the interleaved-reasoning field and the AI SDK package,
// per model (MADR 0009 §2). It is safe for concurrent use: it is not changed
// once parsed, and each section decodes once, however many goroutines read
// it at once. The zero Metadata is no document.
//
// Moved here from llmprovider (0015-PLAN S8, commit 2).
type Metadata struct{ doc *modelMetadataDoc }

// WithMetadata hands in m, the models.dev document, for a listing's ranking,
// its menu's facts, the lookups (LookupModel, CachedModel, CachedFacts) and
// the limits (LookupLimits, CachedLimits), and, given to a provider's New,
// for that provider's requests, for its whole life (0093-MADR D1, D2). A
// request reads only the document its provider was given: one handed to a
// lookup is never seen by a request. It is a shared option, as
// WithModelLayer is: the built-in providers and chatcompat read it, a
// provider outside this module ignores it, and llmprovider.For scopes it.
// Given twice, the last wins, so WithMetadata(Metadata{}) given last drops
// one an earlier option carried.
//
// With no document, a zero one, or one holding none of the sections the SDK
// reads, every reader behaves as with none: the listings rank by name, the
// lookups read the listing and the caller's layers, and a request on a wire
// that requires an output limit, with no other source, is refused
// (0093-MADR D3, D8). The SDK never fetches a document.
func WithMetadata(m Metadata) llmprovider.Option {
	return llmprovider.SharedOption("catalog.WithMetadata", m)
}

// has reports whether m holds any section the SDK reads.
func (m Metadata) has() bool { return m.doc != nil && len(m.doc.sections) > 0 }

// ReasoningEfforts returns the effort values the document lists for a
// model's reasoning_options, or nil.
func (m Metadata) ReasoningEfforts(provider llmprovider.ProviderID, model string) []string {
	return m.doc.reasoningEfforts(provider, model)
}

// InterleavedField returns the message field a model expects its prior
// reasoning replayed under, or "".
func (m Metadata) InterleavedField(provider llmprovider.ProviderID, model string) string {
	return m.doc.interleavedField(provider, model)
}

// NPM returns a model's provider.npm, and false when the document does not
// list the model.
func (m Metadata) NPM(provider llmprovider.ProviderID, model string) (string, bool) {
	return m.doc.npm(provider, model)
}

// modelMetadataKey returns the document key for a provider, or "".
func modelMetadataKey(provider llmprovider.ProviderID) string {
	switch provider {
	case llmprovider.ProviderOpencodeZen:
		return metadataKeyZen
	case llmprovider.ProviderOpencodeGo:
		return metadataKeyGo
	case llmprovider.ProviderHuggingFace:
		return metadataKeyHF
	case llmprovider.ProviderTogether:
		return metadataKeyTogether
	case llmprovider.ProviderOpenAI:
		return metadataKeyOpenAI
	case llmprovider.ProviderClaude:
		return metadataKeyAnthropic
	case llmprovider.ProviderGemini:
		return metadataKeyGoogle
	case llmprovider.ProviderGrok:
		return metadataKeyXAI
	case llmprovider.ProviderKilo:
		return metadataKeyKilo
	}
	return ""
}

// metadataLimit bounds the metadata document (0021-MADR C3).
const metadataLimit = 32 << 20

// sectionBytes captures each occurrence of one section, as read.
type sectionBytes struct{ raws []jsontext.Value }

// UnmarshalJSONFrom keeps the value's bytes; it does not decode them.
func (s *sectionBytes) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	v, err := dec.ReadValue()
	if err != nil {
		return err
	}
	s.raws = append(s.raws, v.Clone())
	return nil
}

// decodeModelMetadata keeps the sections the SDK reads: MADR 0009's three,
// Together's (0017-MADR D1; 0020-MADR F4), and the vendors' and Kilo's
// (0033-MADR D2). Each section's bytes are kept as read, for Encode, and the
// section is decoded when it is first read; a section without models is
// then treated as absent.
// It reads as the body streams in, never holding the whole body, under
// metadataLimit (0028-MADR D-A1).
func decodeModelMetadata(r io.Reader) (*modelMetadataDoc, error) {
	var raw struct {
		Zen       sectionBytes `json:"opencode"`
		Go        sectionBytes `json:"opencode-go"`
		HF        sectionBytes `json:"huggingface"`
		Together  sectionBytes `json:"togetherai"`
		OpenAI    sectionBytes `json:"openai"`
		Anthropic sectionBytes `json:"anthropic"`
		Google    sectionBytes `json:"google"`
		XAI       sectionBytes `json:"xai"`
		Kilo      sectionBytes `json:"kilo"`
	}
	if err := decodeStreamed(r, metadataLimit, &raw, "model metadata"); err != nil {
		return nil, err
	}
	doc := &modelMetadataDoc{sections: map[string]*lazySection{}}
	for key, s := range map[string]sectionBytes{
		metadataKeyZen: raw.Zen, metadataKeyGo: raw.Go, metadataKeyHF: raw.HF, metadataKeyTogether: raw.Together,
		metadataKeyOpenAI: raw.OpenAI, metadataKeyAnthropic: raw.Anthropic, metadataKeyGoogle: raw.Google,
		metadataKeyXAI: raw.XAI, metadataKeyKilo: raw.Kilo,
	} {
		if len(s.raws) > 0 {
			doc.sections[key] = &lazySection{raws: s.raws}
		}
	}
	return doc, nil
}

// unmarshalSection decodes one section's bytes with decodeStreamed's options.
func unmarshalSection(v jsontext.Value, section *modelMetadataSection) error {
	return jsonv2.Unmarshal(v, section, jsonv2.MatchCaseInsensitiveNames(true),
		jsontext.AllowDuplicateNames(true), jsontext.AllowInvalidUTF8(true))
}

// errPastLimit is a body longer than its decode's limit.
var errPastLimit = errors.New("past the limit")

// limitedReader reads r, and fails with errPastLimit when r holds more than
// left bytes.
type limitedReader struct {
	r    io.Reader
	left int64
}

func (l *limitedReader) Read(p []byte) (int, error) {
	if l.left <= 0 {
		var one [1]byte
		if n, err := l.r.Read(one[:]); n > 0 {
			return 0, errPastLimit
		} else if err != nil {
			return 0, err
		}
		return 0, nil
	}
	if int64(len(p)) > l.left {
		p = p[:l.left]
	}
	n, err := l.r.Read(p)
	l.left -= int64(n)
	return n, err
}

// decodeStreamed decodes one JSON value from r into v as it streams in, as
// decodeLimited does from a buffer: a body past limit bytes, or anything but
// white space after the value, is an error. encoding/json's Decoder, and
// json/v2 in v1's legacy mode, buffer the whole value; json/v2 takes only the
// v1 behaviours this decode can meet (names matched without regard to case,
// duplicate names, invalid UTF-8), and decodes as json.Unmarshal would
// (0028-PLAN D12).
func decodeStreamed(r io.Reader, limit int64, v any, what string) error {
	err := jsonv2.UnmarshalRead(&limitedReader{r: r, left: limit}, v, jsonv2.MatchCaseInsensitiveNames(true),
		jsontext.AllowDuplicateNames(true), jsontext.AllowInvalidUTF8(true))
	switch {
	case errors.Is(err, errPastLimit):
		return fmt.Errorf("%s: the reply is larger than %d MiB", what, limit>>20)
	case err != nil:
		return fmt.Errorf("%s: decode: %w", what, err)
	}
	return nil
}

// metadataCandidate reads one models.dev entry (MADR 0009 §2). An id the
// document does not cover is a candidate with every field unknown.
func metadataCandidate(id string, m modelMetadata, covered bool, now time.Time) rankCandidate {
	if !covered {
		return rankCandidate{id: id, group: rankGroup(id, ""), small: isSmallModel(id)}
	}
	c := rankCandidate{
		id:      id,
		group:   rankGroup(id, m.Family),
		small:   isSmallModel(id, m.Family, m.Name),
		context: m.Limit.Context,
		status:  m.Status,
	}
	if m.Reasoning != nil {
		c.reasoning, c.reasoningKnown = *m.Reasoning, true
	}
	// A cost is known only when it is finite and not negative after the sum,
	// and an age only when the release is not after now (0021-MADR C9).
	if m.Cost != nil && m.Cost.Input >= 0 && m.Cost.Output >= 0 {
		if sum := m.Cost.Input + m.Cost.Output; !math.IsInf(sum, 0) && !math.IsNaN(sum) {
			c.cost, c.costKnown = sum, true
		}
	}
	if t, ok := parseRankDate(m.ReleaseDate); ok && !t.After(now) {
		c.ageDays, c.ageKnown = floorDays(t, now), true
	}
	// Small from a known price, and the name only without one (D4).
	if m.Cost != nil {
		if small, ok := smallByPrice(m.Cost.Input, m.Cost.Output); ok {
			c.small = small
		}
	}
	if m.Cost != nil && c.costKnown {
		c.prices = turnPrices{input: m.Cost.Input, output: m.Cost.Output, known: true}
		if r := m.Cost.CacheRead; r != nil && *r >= 0 && !math.IsInf(*r, 0) {
			c.prices.read, c.prices.readKnown = *r, true
		}
		if w := m.Cost.CacheWrite; w != nil && *w >= 0 && !math.IsInf(*w, 0) {
			c.prices.write, c.prices.writeKnown = *w, true
		}
	}
	c.unusable = m.ToolCall != nil && !*m.ToolCall ||
		m.Modalities != nil && m.Modalities.Output != nil && !slices.Contains(m.Modalities.Output, "text")
	return c
}

// metadataCurate ranks a provider's usable models with the metadata document
// (MADR 0009 §2). No document, or one without the provider's key, returns
// fallback's curation unchanged. doc is the call's document, read once for
// the call, so the curation and the menu read the same one (0038-MADR A1;
// 0093-MADR D4). w prices each candidate's cached turn, for ProfileAgent
// (0041-MADR D5).
func metadataCurate(provider llmprovider.ProviderID, profile Profile, w workload, doc *modelMetadataDoc,
	fallback curator) curator {
	return func(usable []string, limit int) []string {
		models, ok := doc.models(modelMetadataKey(provider))
		if !ok {
			return fallback(usable, limit)
		}
		if listedOnly(provider) {
			usable = slices.DeleteFunc(slices.Clone(usable), func(id string) bool {
				_, listed := models[id]
				return !listed
			})
		}
		now := rankingNow()
		cands := make([]rankCandidate, 0, len(usable))
		for _, id := range usable {
			m, covered := models[id]
			c := metadataCandidate(id, m, covered, now)
			c.prices = withCanonicalRead(c.prices, m, doc)
			c = c.withTurn(w)
			// On one vendor's catalog the group is the whole family, not its
			// leading letters (D4).
			if oneVendor[provider] && m.Family != "" {
				c.group = m.Family
			}
			cands = append(cands, c)
		}
		return rankRecommended(profile, provider, cands, append(fallback(usable, limit), usable...), limit)
	}
}

// oneVendor are the providers whose catalog is one vendor's models.
var oneVendor = map[llmprovider.ProviderID]bool{
	llmprovider.ProviderOpenAI: true, llmprovider.ProviderClaude: true,
	llmprovider.ProviderGemini: true, llmprovider.ProviderGrok: true,
}

// listedOnly reports whether a provider ranks and fills only the ids its
// models.dev section lists: Together, whose section is its serverless catalog.
// An id outside it can need streaming, a dedicated endpoint or account access
// (0009-MADR §3 item 10, an exception to "Unknown values never exclude"). Such
// an id stays in Usable, so a search finds it.
func listedOnly(provider llmprovider.ProviderID) bool {
	return provider == llmprovider.ProviderTogether
}
