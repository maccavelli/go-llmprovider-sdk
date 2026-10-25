package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// What each provider's listing says of a model, as a ModelInfo layer
// (0033-MADR D2, as amended on 2026-10-12, and its Kilo row's Phase 1
// amendment). Each converter marks exactly the fields its listing gave.

// usdPerMillionPerXAIUnit converts xAI's price unit, USD cents per 100
// million tokens, to USD per million tokens (measured in
// 0033-PLAN-public-model-metadata.md Phase 1).
const usdPerMillionPerXAIUnit = 1e-4

// supportFlag is one {"supported": bool} leaf of Anthropic's capabilities.
type supportFlag struct {
	Supported bool `json:"supported"`
}

// claudeCapabilities is the part of Anthropic's capabilities object D2 reads.
type claudeCapabilities struct {
	ImageInput        *supportFlag `json:"image_input"`
	PDFInput          *supportFlag `json:"pdf_input"`
	StructuredOutputs *supportFlag `json:"structured_outputs"`
	Thinking          *struct {
		Supported bool `json:"supported"`
		Types     struct {
			Disabled *supportFlag `json:"disabled"`
		} `json:"types"`
	} `json:"thinking"`
	Effort *struct {
		Supported bool         `json:"supported"`
		Low       *supportFlag `json:"low"`
		Medium    *supportFlag `json:"medium"`
		High      *supportFlag `json:"high"`
		XHigh     *supportFlag `json:"xhigh"`
		Max       *supportFlag `json:"max"`
	} `json:"effort"`
}

// dataModel is one entry of an OpenAI-shaped {"data": [...]} listing: OpenAI's
// shutdown date, and xAI's window, efforts and prices.
type dataModel struct {
	ID           string  `json:"id"`
	ShutdownDate *string `json:"shutdown_date"`
	// Created is the Unix time OpenAI and xAI list the model at: a release
	// date, measured (0041-PLAN Phase 2).
	Created int64 `json:"created"`
	// xAI's /v1/models.
	ContextLength *int `json:"context_length"`
	Capabilities  *struct {
		ReasoningEffort []string `json:"reasoning_effort"`
	} `json:"capabilities"`
	PromptPrice           *int64 `json:"prompt_text_token_price"`
	CachedPromptPrice     *int64 `json:"cached_prompt_text_token_price"`
	CompletionPrice       *int64 `json:"completion_text_token_price"`
	PromptPriceLong       *int64 `json:"prompt_text_token_price_long_context"`
	CachedPromptPriceLong *int64 `json:"cached_prompt_text_token_price_long_context"`
	CompletionPriceLong   *int64 `json:"completion_text_token_price_long_context"`
	LongContextThreshold  *int   `json:"long_context_threshold"`
}

// togetherModel is one entry of Together's listing.
type togetherModel struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	ContextLength *int   `json:"context_length"`
	Pricing       *struct {
		Input       *float64 `json:"input"`
		Output      *float64 `json:"output"`
		CachedInput *float64 `json:"cached_input"`
	} `json:"pricing"`
}

// listingLayer is the source name of a provider's listing.
func listingLayer(provider llmprovider.ProviderID) string { return "listing:" + string(provider) }

// infoBuilder builds one ModelInfo, collecting the fields it sets.
type infoBuilder struct {
	info  ModelInfo
	known []Field
}

func newInfo(provider llmprovider.ProviderID, id string) *infoBuilder {
	return &infoBuilder{info: ModelInfo{ID: id, layer: listingLayer(provider)}}
}

func (b *infoBuilder) set(f ...Field) { b.known = append(b.known, f...) }

func (b *infoBuilder) done() ModelInfo { return b.info.Known(b.known...) }

// listingDay is a listing's Unix creation time as a release day, in UTC, and
// false for zero, which dates nothing (0041-MADR D1).
func listingDay(unix int64) (time.Time, bool) {
	if unix <= 0 {
		return time.Time{}, false
	}
	return time.Unix(unix, 0).UTC().Truncate(24 * time.Hour), true
}

// released sets b's Released to day, when there is one.
func (b *infoBuilder) released(day time.Time, ok bool) {
	if ok {
		b.info.Released = day
		b.set(FieldReleased)
	}
}

// listingDate reads a YYYY-MM-DD date, as OpenAI's shutdown_date and Kilo's
// expiration_date give one (measured on 2026-10-12).
func listingDate(s string) (time.Time, bool) {
	t, err := time.Parse(time.DateOnly, s)
	return t, err == nil
}

// geminiInfo reads Gemini's limits and thinking flag.
func geminiInfo(id string, m geminiModel) ModelInfo {
	b := newInfo(llmprovider.ProviderGemini, id)
	if m.InputTokenLimit != nil {
		b.info.MaxInputTokens = *m.InputTokenLimit
		b.set(FieldMaxInputTokens)
	}
	if m.OutputTokenLimit != nil {
		b.info.MaxOutputTokens = *m.OutputTokenLimit
		b.set(FieldMaxOutputTokens)
	}
	if m.Thinking != nil {
		b.info.Reasoning.Supported = *m.Thinking
		b.set(FieldReasoning)
	}
	return b.done()
}

// claudeInfo reads Anthropic's limits and capabilities, never its lifecycle.
func claudeInfo(m claudeModel) ModelInfo {
	b := newInfo(llmprovider.ProviderClaude, m.ID)
	if t, err := time.Parse(time.RFC3339, m.CreatedAt); err == nil {
		b.released(listingDay(t.Unix()))
	}
	if m.MaxInputTokens != nil {
		b.info.MaxInputTokens = *m.MaxInputTokens
		b.set(FieldMaxInputTokens)
	}
	if m.MaxTokens != nil {
		b.info.MaxOutputTokens = *m.MaxTokens
		b.set(FieldMaxOutputTokens)
	}
	c := m.Capabilities
	if c == nil {
		return b.done()
	}
	if c.ImageInput != nil || c.PDFInput != nil {
		b.info.Input = []Modality{ModalityText}
		if c.ImageInput != nil && c.ImageInput.Supported {
			b.info.Input = append(b.info.Input, ModalityImage)
		}
		if c.PDFInput != nil && c.PDFInput.Supported {
			b.info.Input = append(b.info.Input, ModalityPDF)
		}
		b.set(FieldInput)
	}
	if c.StructuredOutputs != nil {
		b.info.StructuredOutput = c.StructuredOutputs.Supported
		b.set(FieldStructuredOutput)
	}
	if c.Thinking != nil {
		b.info.Reasoning.Supported = c.Thinking.Supported
		b.set(FieldReasoning)
		if c.Thinking.Types.Disabled != nil {
			b.info.Reasoning.Toggle = c.Thinking.Types.Disabled.Supported
			b.set(FieldReasoningToggle)
		}
	}
	if e := c.Effort; e != nil {
		b.info.Reasoning.Efforts = []llmprovider.Effort{}
		if e.Supported {
			for _, level := range []struct {
				name string
				flag *supportFlag
			}{{"low", e.Low}, {"medium", e.Medium}, {"high", e.High}, {"xhigh", e.XHigh}, {"max", e.Max}} {
				if level.flag != nil && level.flag.Supported {
					b.info.Reasoning.Efforts = append(b.info.Reasoning.Efforts, llmprovider.Effort(level.name))
				}
			}
		}
		b.set(FieldReasoningEfforts)
	}
	return b.done()
}

// dataModelInfo reads an OpenAI-shaped entry: OpenAI's shutdown date, or
// xAI's window, efforts and prices. Other providers' entries carry nothing
// D2 reads.
func dataModelInfo(provider llmprovider.ProviderID, m dataModel) (ModelInfo, bool) {
	b := newInfo(provider, m.ID)
	switch provider {
	case llmprovider.ProviderOpenAI:
		b.released(listingDay(m.Created))
		if m.ShutdownDate != nil {
			if t, ok := listingDate(*m.ShutdownDate); ok {
				b.info.Retires = t
				b.set(FieldRetires)
			}
		}
	case llmprovider.ProviderGrok:
		b.released(listingDay(m.Created))
		grokFields(b, m)
	default:
		return ModelInfo{}, false
	}
	return b.done(), true
}

// grokFields reads xAI's window, efforts and prices, converted to USD per
// million tokens. Its long-context prices apply "at or above" the
// threshold, which is the tier Above threshold-1.
func grokFields(b *infoBuilder, m dataModel) {
	if m.ContextLength != nil {
		b.info.ContextWindow = *m.ContextLength
		b.set(FieldContextWindow)
	}
	if m.Capabilities != nil && m.Capabilities.ReasoningEffort != nil {
		b.info.Reasoning.Efforts = make([]llmprovider.Effort, len(m.Capabilities.ReasoningEffort))
		for i, e := range m.Capabilities.ReasoningEffort {
			b.info.Reasoning.Efforts[i] = llmprovider.Effort(e)
		}
		b.set(FieldReasoningEfforts)
	}
	usd := func(v int64) float64 { return float64(v) * usdPerMillionPerXAIUnit }
	if m.PromptPrice == nil || m.CompletionPrice == nil {
		return
	}
	rates := &b.info.Cost.ModelRates
	rates.Input, rates.Output = usd(*m.PromptPrice), usd(*m.CompletionPrice)
	b.set(FieldCostInput, FieldCostOutput, FieldCostTiers)
	if m.CachedPromptPrice != nil {
		rates.CacheRead = usd(*m.CachedPromptPrice)
		b.set(FieldCostCacheRead)
	}
	b.info.Cost.Tiers = []ModelCostTier{}
	if m.LongContextThreshold == nil || m.PromptPriceLong == nil {
		return
	}
	tier := ModelCostTier{Above: *m.LongContextThreshold - 1, ModelRates: *rates}
	tier.Input = usd(*m.PromptPriceLong)
	if m.CompletionPriceLong != nil {
		tier.Output = usd(*m.CompletionPriceLong)
	}
	if m.CachedPromptPriceLong != nil {
		tier.CacheRead = usd(*m.CachedPromptPriceLong)
	}
	b.info.Cost.Tiers = append(b.info.Cost.Tiers, tier)
}

// togetherInfo reads Together's window and prices, in USD per million tokens
// (measured in 0033-PLAN-public-model-metadata.md Phase 1).
func togetherInfo(m togetherModel) ModelInfo {
	b := newInfo(llmprovider.ProviderTogether, m.ID)
	if m.ContextLength != nil {
		b.info.ContextWindow = *m.ContextLength
		b.set(FieldContextWindow)
	}
	if p := m.Pricing; p != nil {
		for f, v := range map[Field]*float64{FieldCostInput: p.Input, FieldCostOutput: p.Output, FieldCostCacheRead: p.CachedInput} {
			if v != nil {
				setRate(&b.info.Cost.ModelRates, f, *v)
				b.set(f)
			}
		}
	}
	return b.done()
}

// kiloInfo reads Kilo's window, output cap, prices and expiration. Its prices
// are per-token strings, converted to USD per million tokens. "-1", the
// variable price of its auto-routing models, and any price that does not
// parse are left unset. Its discount is not applied: the rates are list
// prices.
func kiloInfo(e kiloCatalogEntry) ModelInfo {
	b := newInfo(llmprovider.ProviderKilo, e.ID)
	b.released(listingDay(e.Created))
	if e.ContextLength > 0 {
		b.info.ContextWindow = e.ContextLength
		b.set(FieldContextWindow)
	}
	if e.TopProvider.MaxCompletionTokens != nil {
		b.info.MaxOutputTokens = *e.TopProvider.MaxCompletionTokens
		b.set(FieldMaxOutputTokens)
	}
	for f, s := range map[Field]string{FieldCostInput: e.Pricing.Prompt, FieldCostOutput: e.Pricing.Completion,
		FieldCostCacheRead: e.Pricing.InputCacheRead, FieldCostCacheWrite: e.Pricing.InputCacheWrite,
		FieldCostReasoning: e.Pricing.InternalReasoning} {
		if v, ok := kiloPrice(s); ok {
			setRate(&b.info.Cost.ModelRates, f, v*perMillionTokens)
			b.set(f)
		}
	}
	if t, ok := listingDate(e.ExpirationDate); ok {
		b.info.Retires = t
		b.set(FieldRetires)
	}
	return b.done()
}

// setRate sets the rate f names.
func setRate(r *ModelRates, f Field, v float64) {
	switch f {
	case FieldCostInput:
		r.Input = v
	case FieldCostOutput:
		r.Output = v
	case FieldCostCacheRead:
		r.CacheRead = v
	case FieldCostCacheWrite:
		r.CacheWrite = v
	case FieldCostReasoning:
		r.Reasoning = v
	}
}

// Ollama's capability names, as its types/model/capability.go spells them
// (0042-MADR, section 2).
const (
	ollamaCompletion = "completion"
	ollamaImage      = "image"
	ollamaThinking   = "thinking"
	ollamaTools      = "tools"
	ollamaVision     = "vision"
)

// ollamaShowReply is the part of Ollama's POST /api/show reply D2 reads.
type ollamaShowReply struct {
	ModelInfo    map[string]json.RawMessage `json:"model_info"`
	Capabilities []string                   `json:"capabilities"`
}

// fetchOllamaShow asks a local Ollama about one model: its context window,
// from model_info["<general.architecture>.context_length"], and its
// capabilities (tools, vision, thinking, and what it writes). Like the
// listing, it sends a credential only where the token names its own header.
func fetchOllamaShow(ctx context.Context, token llmprovider.Token, cfg config, model string) (ModelInfo, error) {
	baseURL := "http://localhost:11434"
	if cfg.BaseURL != "" {
		baseURL = strings.TrimRight(cfg.BaseURL, "/")
	}
	body, err := json.Marshal(map[string]string{"model": model})
	if err != nil {
		return ModelInfo{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/show", bytes.NewReader(body))
	if err != nil {
		return ModelInfo{}, err
	}
	cfg.setUserAgent(req)
	req.Header.Set("Content-Type", "application/json")
	if token.Header != "" {
		token.Apply(req, "", "")
	}
	resp, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return ModelInfo{}, fmt.Errorf("%w: could not reach Ollama at %s: %w", llmprovider.ErrProviderUnavailable, baseURL,
			transport.WrapOffline(ctx, err, llmprovider.ErrOffline))
	}
	defer cfg.closeBody(resp)
	if resp.StatusCode != http.StatusOK {
		return ModelInfo{}, llmprovider.ClassifyHTTPError(string(llmprovider.ProviderOllama), resp)
	}
	var show ollamaShowReply
	if err := decodeLimited(resp.Body, &show, "ollama: show"); err != nil {
		return ModelInfo{}, err
	}
	return ollamaInfo(model, show), nil
}

// ollamaInfo reads one /api/show reply. A missing or empty capability list
// says nothing: Ollama omits an empty one (0042-MADR D1).
func ollamaInfo(model string, show ollamaShowReply) ModelInfo {
	b := newInfo(llmprovider.ProviderOllama, model)
	var arch string
	if json.Unmarshal(show.ModelInfo["general.architecture"], &arch) == nil && arch != "" {
		var window int
		if json.Unmarshal(show.ModelInfo[arch+".context_length"], &window) == nil && window > 0 {
			b.info.ContextWindow = window
			b.set(FieldContextWindow)
		}
	}
	if caps := show.Capabilities; len(caps) > 0 {
		b.info.Tools = slices.Contains(caps, ollamaTools)
		b.info.Input = []Modality{ModalityText}
		if slices.Contains(caps, ollamaVision) {
			b.info.Input = append(b.info.Input, ModalityImage)
		}
		b.info.Output = ollamaOutputs(caps)
		b.info.Reasoning.Supported = slices.Contains(caps, ollamaThinking)
		b.set(FieldTools, FieldInput, FieldOutput, FieldReasoning)
	}
	return b.done()
}

// ollamaOutputs is what a model writes, from its capabilities: text when it
// completes, an image when it makes one. An embedding or a decision model
// writes neither (0042-MADR D1).
func ollamaOutputs(capabilities []string) []Modality {
	var out []Modality
	if slices.Contains(capabilities, ollamaCompletion) {
		out = append(out, ModalityText)
	}
	if slices.Contains(capabilities, ollamaImage) {
		out = append(out, ModalityImage)
	}
	return out
}

// ollamaTagInfo is what one /api/tags entry says of its model: only what it
// writes. Its tools, input and reasoning come from /api/show alone, where the
// two endpoints were measured to differ (0042-MADR D1).
func ollamaTagInfo(t ollamaTag) ModelInfo {
	b := newInfo(llmprovider.ProviderOllama, t.Name)
	if len(t.Capabilities) > 0 {
		b.info.Output = ollamaOutputs(t.Capabilities)
		b.set(FieldOutput)
	}
	return b.done()
}
