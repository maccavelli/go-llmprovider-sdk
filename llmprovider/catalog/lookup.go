package catalog

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Layer is a source of model metadata a caller adds, such as a models.json of
// its own, or a document it persisted (0033-MADR D2, as amended). Model
// returns what the layer says of one model, with the fields it knows marked
// by ModelInfo.Known, and whether it knows the model at all. Model must not
// block or reach the network. Metadata is a Layer. Implementations must be
// safe for concurrent use: every request of a provider given WithModelLayer
// asks the layer, so concurrent requests call Model at once.
type Layer interface {
	// Name is the layer's source name, as SourceOf reports it.
	Name() string
	// Model returns what the layer says of the model id of provider.
	Model(provider llmprovider.ProviderID, id string) (ModelInfo, bool)
}

// WithModelLayer adds l to LookupModel's and CachedModel's layers. A layer
// added later ranks above one added earlier, and every caller layer ranks
// above the provider's listing and models.dev. It is scoped to every
// built-in provider, as WithProfile is (R17, R18).
func WithModelLayer(l Layer) llmprovider.Option {
	return llmprovider.ScopedOptionFor(builtinIDs, "catalog.WithModelLayer", modelLayerOption{l})
}

// listingLayers are the providers whose listing is a layer: those amended D2
// gives fields.
var listingLayers = map[llmprovider.ProviderID]bool{
	llmprovider.ProviderHuggingFace: true, // its release dates (0041-PLAN Phase 2)
	llmprovider.ProviderClaude:      true, llmprovider.ProviderGemini: true, llmprovider.ProviderGrok: true,
	llmprovider.ProviderKilo: true, llmprovider.ProviderTogether: true, llmprovider.ProviderOpenAI: true,
	llmprovider.ProviderOllama: true,
}

// lookupProvider checks provider, matched without case.
func lookupProvider(provider llmprovider.ProviderID) (llmprovider.ProviderID, error) {
	id := llmprovider.ProviderID(strings.ToLower(string(provider)))
	if !slices.Contains(builtinIDs, id) {
		return "", fmt.Errorf("model metadata: %w: %q", llmprovider.ErrInvalidProvider, provider)
	}
	return id, nil
}

// LookupModel returns what is known of model, merged field by field from its
// layers: the caller's (WithModelLayer), the provider's listing when src is
// not nil, and models.dev unless WithoutModelMetadata turns it off
// (0033-MADR D2, D3). It fetches what it needs, bounded as a listing is (R40).
//
// A model no layer knows is not an error: its fields are all unset, as Has
// reports. A source that fails is reported in the error, beside what the
// other layers set, which the caller may use either way.
func LookupModel(ctx context.Context, provider llmprovider.ProviderID, src llmprovider.TokenSource, model string,
	opts ...llmprovider.Option) (ModelInfo, error) {
	id, err := lookupProvider(provider)
	if err != nil {
		return ModelInfo{}, err
	}
	cfg, err := configFor(id, opts)
	if err != nil {
		return ModelInfo{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, modelListingTimeout)
	defer cancel()
	var errs []error
	var dev, lst ModelInfo
	var devOK, lstOK bool
	if !cfg.metadataOff {
		doc, err := loadModelMetadata(ctx, cfg)
		if err != nil {
			errs = append(errs, err)
		} else {
			dev, devOK = Metadata{doc: doc}.Model(id, model)
		}
	}
	if src != nil && listingLayers[id] {
		lst, lstOK, err = listingModel(ctx, id, src, model, cfg)
		if err != nil {
			errs = append(errs, fmt.Errorf("model metadata: listing %s: %w", id, err))
		}
	}
	return mergeLayers(id, model, cfg.layers, dev, devOK, lst, lstOK), errors.Join(errs...)
}

// CachedModel is LookupModel without the wait: what the caches and the
// caller's layers hold now, and whether anything is known of model. It starts
// any fetch that is missing or stale, so a later call finds it. It asks src
// for its token as the source gives it.
func CachedModel(ctx context.Context, provider llmprovider.ProviderID, src llmprovider.TokenSource, model string,
	opts ...llmprovider.Option) (ModelInfo, bool) {
	id, err := lookupProvider(provider)
	if err != nil {
		return ModelInfo{}, false
	}
	cfg, err := configFor(id, opts)
	if err != nil {
		return ModelInfo{}, false
	}
	var dev, lst ModelInfo
	var devOK, lstOK bool
	if m, ok := cachedMetadata(ctx, cfg); ok {
		dev, devOK = m.Model(id, model)
	}
	if src != nil && listingLayers[id] {
		if token, err := src.Token(ctx); err == nil {
			key := listingKeyFor(id, cfg, token)
			models, fresh := cfg.store.cachedListing(key)
			lst, lstOK = models[model]
			if !fresh || (id == llmprovider.ProviderOllama && !lstOK) {
				cfg.store.startListingFetch(ctx, id, src, model, cfg, key)
			}
		}
	}
	info := mergeLayers(id, model, cfg.layers, dev, devOK, lst, lstOK)
	return info, info.known != 0
}

// CachedFacts is CachedModel without the listing fetch: what a request may
// read of model at request time (0041-MADR D1). It merges, field by field,
// the caller's layers, a listing a List or ListModels call already read for
// the provider, base URL and credential opts give, where no base URL is the
// provider's default (0041-MADR A8), and the cached models.dev
// document, and reports whether anything is known of model. It starts no
// fetch: catalog.List, ListModels and LookupModel warm the caches it reads
// (0041-MADR A3). WithoutModelMetadata leaves the document out. It asks the
// credential opts give for its token, to find the listing: a provider passes
// its request's own token, as WithAPIKey(token.Value) after its options, so
// that its token source is not asked a second time.
func CachedFacts(ctx context.Context, provider llmprovider.ProviderID, model string, opts ...llmprovider.Option) (ModelInfo,
	bool) {
	id, err := lookupProvider(provider)
	if err != nil {
		return ModelInfo{}, false
	}
	cfg, err := configFor(id, opts)
	if err != nil {
		return ModelInfo{}, false
	}
	var dev, lst ModelInfo
	var devOK, lstOK bool
	if m, ok := peekMetadata(cfg); ok {
		dev, devOK = m.Model(id, model)
	}
	if cfg.tokens != nil && listingLayers[id] {
		if token, err := cfg.tokens.Token(ctx); err == nil {
			models, _ := cfg.store.cachedListing(listingKeyFor(id, cfg, token))
			lst, lstOK = models[model]
		}
	}
	info := mergeLayers(id, model, cfg.layers, dev, devOK, lst, lstOK)
	return info, info.known != 0
}

// listingModel reads one model's metadata from its provider's listing, from
// the cache while it is fresh.
func listingModel(ctx context.Context, id llmprovider.ProviderID, src llmprovider.TokenSource, model string,
	cfg config) (ModelInfo, bool, error) {
	token, err := src.Token(ctx)
	if err != nil {
		return ModelInfo{}, false, fmt.Errorf("acquire token: %w", err)
	}
	if models, fresh := cfg.store.cachedListing(listingKeyFor(id, cfg, token)); fresh {
		if info, ok := models[model]; ok || id != llmprovider.ProviderOllama {
			return info, ok, nil
		}
	}
	return fetchListingModel(ctx, id, src, model, cfg)
}

// mergeLayers merges, field by field, models.dev, then the listing, then each
// caller layer in order: a field a later layer sets replaces the earlier
// one's (0033-MADR D2).
func mergeLayers(id llmprovider.ProviderID, model string, layers []Layer, dev ModelInfo, devOK bool,
	lst ModelInfo, lstOK bool) ModelInfo {
	info := ModelInfo{ID: model}
	if devOK {
		overlay(&info, dev, "")
	}
	if lstOK {
		overlay(&info, lst, "")
	}
	for _, l := range layers {
		if v, ok := l.Model(id, model); ok {
			overlay(&info, v, l.Name())
		}
	}
	return info
}

// overlay copies each field src has onto dst, with its source: src's own, or
// name when name is set. Slices are copied, so dst shares nothing with src.
func overlay(dst *ModelInfo, src ModelInfo, name string) {
	for f := range fieldCount {
		if !src.Has(f) {
			continue
		}
		copyField(dst, src, f)
		dst.known |= 1 << f
		dst.sources[f] = src.sources[f]
		if name != "" {
			dst.sources[f] = name
		}
	}
}

// copyField copies field f's value from src to dst.
func copyField(dst *ModelInfo, src ModelInfo, f Field) {
	switch f {
	case FieldName:
		dst.Name = src.Name
	case FieldFamily:
		dst.Family = src.Family
	case FieldContextWindow:
		dst.ContextWindow = src.ContextWindow
	case FieldMaxInputTokens:
		dst.MaxInputTokens = src.MaxInputTokens
	case FieldMaxOutputTokens:
		dst.MaxOutputTokens = src.MaxOutputTokens
	case FieldInput:
		dst.Input = slices.Clone(src.Input)
	case FieldOutput:
		dst.Output = slices.Clone(src.Output)
	case FieldTools:
		dst.Tools = src.Tools
	case FieldTemperature:
		dst.Temperature = src.Temperature
	case FieldStructuredOutput:
		dst.StructuredOutput = src.StructuredOutput
	case FieldReasoning:
		dst.Reasoning.Supported = src.Reasoning.Supported
	case FieldReasoningToggle:
		dst.Reasoning.Toggle = src.Reasoning.Toggle
	case FieldReasoningEfforts:
		dst.Reasoning.Efforts = slices.Clone(src.Reasoning.Efforts)
	case FieldReasoningBudget:
		dst.Reasoning.BudgetMin, dst.Reasoning.BudgetMax = src.Reasoning.BudgetMin, src.Reasoning.BudgetMax
	case FieldCostInput, FieldCostOutput, FieldCostCacheRead, FieldCostCacheWrite, FieldCostReasoning:
		setRate(&dst.Cost.ModelRates, f, rate(src.Cost.ModelRates, f))
	case FieldCostTiers:
		dst.Cost.Tiers = slices.Clone(src.Cost.Tiers)
	case FieldReleased:
		dst.Released = src.Released
	case FieldStatus:
		dst.Status = src.Status
	case FieldRetires:
		dst.Retires = src.Retires
	case FieldReasoningReplay:
		dst.Reasoning.ReplayField = src.Reasoning.ReplayField
	case FieldCanonicalID:
		dst.CanonicalID = src.CanonicalID
	}
}

// rate is the rate f names.
func rate(r ModelRates, f Field) float64 {
	switch f {
	case FieldCostInput:
		return r.Input
	case FieldCostOutput:
		return r.Output
	case FieldCostCacheRead:
		return r.CacheRead
	case FieldCostCacheWrite:
		return r.CacheWrite
	default:
		return r.Reasoning
	}
}
