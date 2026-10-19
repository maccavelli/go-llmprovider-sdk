package catalog

import (
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// ModelInfo is what the SDK knows about one model: its limits, modalities,
// capabilities, prices and lifecycle (0033-MADR D1, as amended on
// 2026-10-12).
//
// A value that is not known is not zero by convention. Has reports whether a
// layer set a field, so a known zero, such as a free price, is told apart from
// an unknown one, and SourceOf names the layer that set it. ModelInfo is a
// plain value: its slices are the caller's own copies. It is safe for
// concurrent use only while no goroutine changes it.
type ModelInfo struct {
	// ID is the model's id, as the provider names it.
	ID string
	// CanonicalID is models.dev's canonical_model_id: the same model's id
	// in its maker's section, as section/model, such as
	// "anthropic/claude-opus-4-1" for a gateway's copy (0041-MADR D1).
	CanonicalID string
	// Name and Family are the model's display name and family.
	Name, Family string
	// ContextWindow is the model's context window, in tokens. MaxInputTokens
	// and MaxOutputTokens are its input and output limits, where the source
	// gives them apart from the window.
	ContextWindow, MaxInputTokens, MaxOutputTokens int
	// Input and Output are the modalities the model reads and writes.
	Input, Output []Modality
	// Tools, Temperature and StructuredOutput report whether the model takes
	// tools, a sampling temperature, and a structured-output schema.
	Tools, Temperature, StructuredOutput bool
	// Reasoning describes the model's reasoning controls.
	Reasoning ModelReasoning
	// Cost is the model's price, in USD per million tokens.
	Cost ModelCost
	// Released is the model's release date, in UTC.
	Released time.Time
	// Status is the model's lifecycle stage.
	Status ModelStatus
	// Retires is a published retirement or shutdown date, in UTC.
	Retires time.Time

	known   uint64             // bit f is set when a layer set Field f
	sources [fieldCount]string // the layer that set each field
	layer   string             // the layer that built this value, for Known
}

// ModelReasoning is a model's reasoning controls (0033-MADR Q10). It is safe
// for concurrent use only while no goroutine changes it.
type ModelReasoning struct {
	// Supported reports whether the model reasons.
	Supported bool
	// Toggle reports whether reasoning can be turned off and on.
	Toggle bool
	// Efforts are the effort levels the model takes, in the source's order
	// and with its names: "none", "minimal" and "max" among them, which
	// llmprovider.Effort holds without constants.
	Efforts []llmprovider.Effort
	// BudgetMin and BudgetMax bound a reasoning budget in tokens, for a model
	// that takes one. Zero is an unstated bound.
	BudgetMin, BudgetMax int
	// ReplayField names the message field a Chat Completions request
	// replays the model's prior reasoning under, such as
	// "reasoning_content"; empty for none (0041-MADR D2, A3).
	ReplayField string
}

// ModelRates are prices, in USD per million tokens. It is safe for
// concurrent use only while no goroutine changes it.
type ModelRates struct {
	// Input and Output are the prices of input and output tokens.
	Input, Output float64
	// CacheRead and CacheWrite are the prices of cached input read and
	// written.
	CacheRead, CacheWrite float64
	// Reasoning is the price of reasoning tokens, where it differs from
	// Output.
	Reasoning float64
}

// ModelCost is a model's price: its base rates, and the rates of each tier.
// It is safe for concurrent use only while no goroutine changes it.
type ModelCost struct {
	ModelRates
	// Tiers are the prices for longer inputs, ascending by Above.
	Tiers []ModelCostTier
}

// ModelCostTier is the price that applies to an input of more than Above
// tokens. It is safe for concurrent use only while no goroutine changes it.
type ModelCostTier struct {
	// Above is the input size, in tokens, past which the tier applies.
	Above int
	ModelRates
}

// RatesFor returns the rates for an input of inputTokens tokens: those of the
// highest tier whose Above it exceeds, else the base rates. This is the rule
// Pi and OpenCode apply; a provider's "at or above t" threshold is the tier
// Above t-1.
func (c ModelCost) RatesFor(inputTokens int) ModelRates {
	rates := c.ModelRates
	for _, tier := range c.Tiers {
		if inputTokens > tier.Above {
			rates = tier.ModelRates
		}
	}
	return rates
}

// Modality is a kind of model input or output (R21). It is a string, safe for
// concurrent use.
type Modality string

// The modalities models.dev names.
const (
	// ModalityText is text.
	ModalityText Modality = "text"
	// ModalityImage is an image.
	ModalityImage Modality = "image"
	// ModalityAudio is audio.
	ModalityAudio Modality = "audio"
	// ModalityVideo is video.
	ModalityVideo Modality = "video"
	// ModalityPDF is a PDF document.
	ModalityPDF Modality = "pdf"
)

// ModelStatus is a model's lifecycle stage (R21). It is a string, safe for
// concurrent use.
type ModelStatus string

// The lifecycle stages.
const (
	// StatusActive is a model generally available.
	StatusActive ModelStatus = "active"
	// StatusBeta is a model in beta.
	StatusBeta ModelStatus = "beta"
	// StatusAlpha is a model in alpha.
	StatusAlpha ModelStatus = "alpha"
	// StatusDeprecated is a model that is deprecated, or no longer served by
	// its provider's public API.
	StatusDeprecated ModelStatus = "deprecated"
	// StatusRetired is a model its provider has retired.
	StatusRetired ModelStatus = "retired"
)

// Field names one ModelInfo field, for Has, SourceOf and Known (R21). A field
// added later is one more constant. It is an integer, safe for
// concurrent use.
type Field int

// The fields of ModelInfo, and of its Reasoning and Cost.
const (
	// FieldName is ModelInfo.Name.
	FieldName Field = iota
	// FieldFamily is ModelInfo.Family.
	FieldFamily
	// FieldContextWindow is ModelInfo.ContextWindow.
	FieldContextWindow
	// FieldMaxInputTokens is ModelInfo.MaxInputTokens.
	FieldMaxInputTokens
	// FieldMaxOutputTokens is ModelInfo.MaxOutputTokens.
	FieldMaxOutputTokens
	// FieldInput is ModelInfo.Input.
	FieldInput
	// FieldOutput is ModelInfo.Output.
	FieldOutput
	// FieldTools is ModelInfo.Tools.
	FieldTools
	// FieldTemperature is ModelInfo.Temperature.
	FieldTemperature
	// FieldStructuredOutput is ModelInfo.StructuredOutput.
	FieldStructuredOutput
	// FieldReasoning is ModelInfo.Reasoning.Supported.
	FieldReasoning
	// FieldReasoningToggle is ModelInfo.Reasoning.Toggle.
	FieldReasoningToggle
	// FieldReasoningEfforts is ModelInfo.Reasoning.Efforts.
	FieldReasoningEfforts
	// FieldReasoningBudget is ModelInfo.Reasoning.BudgetMin and BudgetMax.
	FieldReasoningBudget
	// FieldCostInput is ModelInfo.Cost.Input.
	FieldCostInput
	// FieldCostOutput is ModelInfo.Cost.Output.
	FieldCostOutput
	// FieldCostCacheRead is ModelInfo.Cost.CacheRead.
	FieldCostCacheRead
	// FieldCostCacheWrite is ModelInfo.Cost.CacheWrite.
	FieldCostCacheWrite
	// FieldCostReasoning is ModelInfo.Cost.Reasoning.
	FieldCostReasoning
	// FieldCostTiers is ModelInfo.Cost.Tiers, as a whole: a layer that sets
	// it replaces the list. Known with no tiers is a flat price.
	FieldCostTiers
	// FieldReleased is ModelInfo.Released.
	FieldReleased
	// FieldStatus is ModelInfo.Status.
	FieldStatus
	// FieldRetires is ModelInfo.Retires.
	FieldRetires
	// FieldReasoningReplay is ModelInfo.Reasoning.ReplayField.
	FieldReasoningReplay
	// FieldCanonicalID is ModelInfo.CanonicalID.
	FieldCanonicalID

	fieldCount
)

// valid reports whether f names a field.
func (f Field) valid() bool { return f >= 0 && f < fieldCount }

// without is m with f unset: its value zeroed, its mark and source cleared.
func (m ModelInfo) without(f Field) ModelInfo {
	if !f.valid() {
		return m
	}
	var zero ModelInfo
	copyField(&m, zero, f)
	m.known &^= 1 << f
	m.sources[f] = ""
	return m
}

// Has reports whether a layer set f.
func (m ModelInfo) Has(f Field) bool { return f.valid() && m.known&(1<<f) != 0 }

// SourceOf names the layer that set f: "models.dev", "listing:<provider>", or
// a caller's layer name. It is "" when f is not set, or was set on a value no
// layer has returned yet.
func (m ModelInfo) SourceOf(f Field) string {
	if !m.Has(f) {
		return ""
	}
	return m.sources[f]
}

// Known returns a copy of m with fields marked as set, by the layer that built
// m. It is how a layer, the SDK's own or a caller's, says which fields it
// knows: a value is never inferred known from being non-zero, so a known zero,
// such as a free price, is marked too (0033-MADR Q11). m is not changed.
func (m ModelInfo) Known(fields ...Field) ModelInfo {
	for _, f := range fields {
		if f.valid() {
			m.known |= 1 << f
			m.sources[f] = m.layer
		}
	}
	return m
}
