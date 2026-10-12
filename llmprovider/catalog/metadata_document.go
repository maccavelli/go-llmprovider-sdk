package catalog

import (
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"sort"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// sourceModelsDev is the source name of the models.dev layer.
const sourceModelsDev = "models.dev"

// ParseMetadata reads a models.dev-format document from r, such as one a
// caller persisted with Encode, and makes no request (0033-MADR D3). It keeps
// the sections the SDK reads, decoding each now, so a malformed one is
// reported here, and refuses a document over the size the SDK fetches, or one
// that is not a document.
func ParseMetadata(r io.Reader) (Metadata, error) {
	doc, err := decodeModelMetadata(r)
	if err == nil {
		err = doc.decodeAll()
	}
	if err != nil {
		return Metadata{}, err
	}
	return Metadata{doc: doc}, nil
}

// Encode writes the sections of m the SDK reads, in models.dev's format, each
// section's bytes exactly as they were read, so fields the SDK does not
// decode survive. ParseMetadata reads the result back to an equal Metadata
// (0033-MADR D3, as amended).
func (m Metadata) Encode(w io.Writer) error {
	keys := make([]string, 0)
	if m.doc != nil {
		for key := range m.doc.sections {
			if _, ok := m.doc.models(key); ok {
				keys = append(keys, key)
			}
		}
	}
	sort.Strings(keys)
	out := []byte{'{'}
	for _, key := range keys {
		name, err := json.Marshal(key)
		if err != nil {
			return fmt.Errorf("model metadata: encode: %w", err)
		}
		for _, v := range m.doc.sections[key].raws {
			if len(out) > 1 {
				out = append(out, ',')
			}
			out = append(append(append(out, name...), ':'), v...)
		}
	}
	out = append(out, '}', '\n')
	if _, err := w.Write(out); err != nil {
		return fmt.Errorf("model metadata: encode: %w", err)
	}
	return nil
}

// Name is the source name of the models.dev layer, as SourceOf reports it.
func (m Metadata) Name() string { return sourceModelsDev }

// Model returns what the document says of one model, and whether it lists it.
// The result is the caller's own copy (R29).
func (m Metadata) Model(provider llmprovider.ProviderID, id string) (ModelInfo, bool) {
	e, ok := m.doc.entry(provider, id)
	if !ok {
		return ModelInfo{}, false
	}
	return modelInfoFrom(id, e), true
}

// Models returns what the document says of every model it lists for provider,
// sorted by id, as the caller's own copies (R29).
func (m Metadata) Models(provider llmprovider.ProviderID) []ModelInfo {
	models, _ := m.doc.models(modelMetadataKey(provider))
	ids := make([]string, 0, len(models))
	for id := range models {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]ModelInfo, len(ids))
	for i, id := range ids {
		out[i] = modelInfoFrom(id, models[id])
	}
	return out
}

// modelInfoFrom reads one models.dev entry, marking each field it sets. Every
// slice is built afresh, so the result shares nothing with the document.
func modelInfoFrom(id string, e modelMetadata) ModelInfo {
	info := ModelInfo{ID: id, layer: sourceModelsDev}
	var known []Field
	set := func(f Field) { known = append(known, f) }
	if e.Name != "" {
		info.Name = e.Name
		set(FieldName)
	}
	if e.Family != "" {
		info.Family = e.Family
		set(FieldFamily)
	}
	if e.Limit.Context > 0 {
		info.ContextWindow = e.Limit.Context
		set(FieldContextWindow)
	}
	if e.Limit.Input != nil {
		info.MaxInputTokens = *e.Limit.Input
		set(FieldMaxInputTokens)
	}
	if e.Limit.Output != nil {
		info.MaxOutputTokens = *e.Limit.Output
		set(FieldMaxOutputTokens)
	}
	if e.Modalities != nil {
		if e.Modalities.Input != nil {
			info.Input = modalities(e.Modalities.Input)
			set(FieldInput)
		}
		if e.Modalities.Output != nil {
			info.Output = modalities(e.Modalities.Output)
			set(FieldOutput)
		}
	}
	for f, v := range map[Field]*bool{FieldTools: e.ToolCall, FieldTemperature: e.Temperature, FieldStructuredOutput: e.StructuredOutput} {
		if v != nil {
			switch f {
			case FieldTools:
				info.Tools = *v
			case FieldTemperature:
				info.Temperature = *v
			default:
				info.StructuredOutput = *v
			}
			set(f)
		}
	}
	known = append(known, reasoningFrom(&info.Reasoning, e)...)
	known = append(known, costFrom(&info.Cost, e.Cost)...)
	if t, ok := parseReleaseDate(e.ReleaseDate); ok {
		info.Released = t
		set(FieldReleased)
	}
	if s, ok := statusFrom(e.Status); ok {
		info.Status = s
		set(FieldStatus)
	}
	return info.Known(known...)
}

// modalities converts models.dev's modality names, keeping any it does not
// name a constant for.
func modalities(names []string) []Modality {
	out := make([]Modality, len(names))
	for i, n := range names {
		out[i] = Modality(n)
	}
	return out
}

// reasoningFrom fills r from the entry's reasoning flag and options, and
// returns the fields it set. The options' fields are known only where the
// options are given; with reasoning off, only Supported is.
func reasoningFrom(r *ModelReasoning, e modelMetadata) []Field {
	var known []Field
	if e.Reasoning != nil {
		r.Supported = *e.Reasoning
		known = append(known, FieldReasoning)
	}
	if e.ReasoningOptions == nil || (e.Reasoning != nil && !*e.Reasoning) {
		return known
	}
	known = append(known, FieldReasoningToggle)
	for _, o := range e.ReasoningOptions {
		switch o.Type {
		case "toggle":
			r.Toggle = true
		case jsonKeyEffort:
			r.Efforts = make([]llmprovider.Effort, len(o.Values))
			for i, v := range o.Values {
				if v == "" { // models.dev's null: no reasoning
					v = "none"
				}
				r.Efforts[i] = llmprovider.Effort(v)
			}
			known = append(known, FieldReasoningEfforts)
		case "budget_tokens":
			if o.Min != nil {
				r.BudgetMin = *o.Min
			}
			if o.Max != nil {
				r.BudgetMax = *o.Max
			}
			known = append(known, FieldReasoningBudget)
		}
	}
	return known
}

// costFrom fills c from the entry's cost, and returns the fields it set.
// tiers is read over context_over_200k when both are given; a tier's omitted
// rate is the base rate.
func costFrom(c *ModelCost, cost *modelCost) []Field {
	if cost == nil {
		return nil
	}
	known := []Field{FieldCostInput, FieldCostOutput, FieldCostTiers}
	c.Input, c.Output = cost.Input, cost.Output
	for f, v := range map[Field]*float64{FieldCostCacheRead: cost.CacheRead, FieldCostCacheWrite: cost.CacheWrite,
		FieldCostReasoning: cost.Reasoning} {
		if v == nil {
			continue
		}
		switch f {
		case FieldCostCacheRead:
			c.CacheRead = *v
		case FieldCostCacheWrite:
			c.CacheWrite = *v
		default:
			c.Reasoning = *v
		}
		known = append(known, f)
	}
	c.Tiers = []ModelCostTier{}
	for _, t := range cost.Tiers {
		if t.Tier.Type == "context" {
			c.Tiers = append(c.Tiers, ModelCostTier{Above: t.Tier.Size, ModelRates: tierRates(c.ModelRates, t.modelCostRates)})
		}
	}
	if len(c.Tiers) == 0 && cost.ContextOver200k != nil {
		c.Tiers = append(c.Tiers, ModelCostTier{Above: 200000, ModelRates: tierRates(c.ModelRates, *cost.ContextOver200k)})
	}
	slices.SortFunc(c.Tiers, func(a, b ModelCostTier) int { return a.Above - b.Above })
	return known
}

// tierRates is a tier's rates, each omitted one the base's.
func tierRates(base ModelRates, t modelCostRates) ModelRates {
	pick := func(v *float64, fallback float64) float64 {
		if v == nil {
			return fallback
		}
		return *v
	}
	return ModelRates{Input: pick(t.Input, base.Input), Output: pick(t.Output, base.Output),
		CacheRead: pick(t.CacheRead, base.CacheRead), CacheWrite: pick(t.CacheWrite, base.CacheWrite),
		Reasoning: pick(t.Reasoning, base.Reasoning)}
}

// parseReleaseDate reads models.dev's YYYY-MM-DD or YYYY-MM, as a UTC date.
func parseReleaseDate(s string) (time.Time, bool) {
	for _, layout := range []string{time.DateOnly, "2006-01"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// statusFrom reads models.dev's status: absent is active; a value it does not
// know is unknown.
func statusFrom(s string) (ModelStatus, bool) {
	switch s {
	case "":
		return StatusActive, true
	case string(StatusAlpha), string(StatusBeta), string(StatusDeprecated), string(StatusRetired):
		return ModelStatus(s), true
	}
	return "", false
}
