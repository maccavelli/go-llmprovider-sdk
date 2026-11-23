package catalog

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/chatgpt"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/nilsource"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/opencoderoute"
)

// A model's limits and the evidence for each (0089-MADR D1 to D3): every
// source read apart, kept, and ranked; a request reads them through
// CachedLimits, a caller setting a session up through LookupLimits.

// Limits is what is known of one model's limits on one provider, with the
// evidence for each (0089-MADR D1). It is a value, safe for concurrent use
// while no goroutine changes it: its slices are the caller's own copies.
type Limits struct {
	// Provider and Model are the provider id, lower case, and the model as
	// the caller named it.
	Provider llmprovider.ProviderID
	Model    string
	// Output is the most the output-limit field may carry. It is never above
	// a known Window: a stated output above it is held to it, and marked
	// Conflicting.
	Output Fact
	// Window is input plus output, where a source states it.
	Window Fact
	// Input is the input limit, where a source states it apart from the
	// window (Claude, Gemini).
	Input Fact
	// Required reports whether the model's wire requires an output limit:
	// the Messages wire, Claude's and OpenCode's messages route.
	Required bool
	// Checked is when the sources were read.
	Checked time.Time
}

// Fact is one limit: the chosen value and every source's (0089-MADR D1). It
// is a value, safe for concurrent use while no goroutine changes it.
type Fact struct {
	// Tokens is the chosen value; 0 when not Known.
	Tokens int
	// Known reports whether a source states the limit. A stated 0 is not a
	// limit: it is kept in Evidence, and leaves the fact unknown.
	Known bool
	// Conflicting reports that two sources state different values, or that
	// the output was held to the window.
	Conflicting bool
	// Evidence is every source that stated the limit: the chosen first, then
	// by rank.
	Evidence []Evidence
}

// Evidence is one source's statement of a fact (0089-MADR D1). It is a value,
// safe for concurrent use.
type Evidence struct {
	// Source names the source: "listing:<provider>" for a provider's listing
	// or its one-model read; "server:<provider>" for the context a local
	// server serves; "models.dev", "models.dev (canonical id)" and
	// "models.dev (family)" for the fetched document by exact id, canonical
	// id and family; "models.dev (bundled <date>)", with ", canonical id" or
	// ", family" before its parenthesis, for the embedded extract; or a
	// caller's layer's Name.
	Source string
	// Tokens is the value the source states.
	Tokens int
	// Seen is when the source was read; zero for a caller's layer, and the
	// extract's date for the extract.
	Seen time.Time
	// Stale reports that the source was read longer ago than its freshness:
	// ten minutes for a listing, a one-model or a served read and the
	// document; always for the extract.
	Stale bool
}

// The ranks of the evidence: a caller's layers above the service, the
// service above the registry; within the service, a server's served context
// above the rest; within the registry, a "models.dev" layer, then the fetched
// document, then the extract, each by exact id, then canonical id, then
// family (0089-MADR D1, D3). A later layer ranks above an earlier one.
const (
	rankCaller        = 10000
	rankServed        = rankService + 1
	rankService       = 5000
	rankRegistryLayer = 4000
	rankDocument      = 3000
	rankBundled       = 2000
)

// The fetched document's source names by stage: its exact id's is
// sourceModelsDev, its canonical id's sourceCanonical.
const sourceFamily = "models.dev (family)"

// limitReadTTL is how long a one-model, served or backend read is fresh: the
// listing's freshness (0089-MADR D2, DL66).
const limitReadTTL = listingMetadataTTL

// LookupLimits reads every source of model's limits on provider, bounded as
// LookupModel is, warms the caches a request reads, and returns what they
// state (0089-MADR D2). It reads what LookupModel reads, and: for Claude, a
// model its listing does not name, by GET /v1/models/{model}, which takes an
// alias; for Ollama, and a chatcompat server of Ollama's dialect, the context
// /api/ps serves a loaded model; for OpenAI with a ChatGPT session or plan
// session, the backend's listing in place of OpenAI's. A one-model or served
// read is kept apart from the listing, so a listing refresh keeps it, and a
// 404 is kept as none for the same ten minutes. The embedded models.dev
// extract answers too, below the fetched document, so offline the layers and
// the extract still state what they know.
//
// A Claude workspace is read from catalog.WithClaudeWorkspace, as every
// listing reads it: a caller whose provider takes claude.WithWorkspace passes
// catalog.WithClaudeWorkspace with the same id, as the provider does for its
// own listing, so that its requests find what this stored (0089-MADR DL75).
//
// A model no source knows is not an error: each fact is unknown. A failing
// source is reported in the error, joined, beside what the others gave.
// Errors: ErrInvalidProvider for an id LookupModel refuses; the closed-cache
// error LookupModel returns, matching llmprovider.ErrInvalidRequest; one
// matching ErrInvalidRequest for a typed-nil src; ctx's error; and each
// source's own: an *llmprovider.APIError, or a plain error for a transport
// failure.
func LookupLimits(ctx context.Context, provider llmprovider.ProviderID, src llmprovider.TokenSource, model string,
	opts ...llmprovider.Option) (Limits, error) {
	id, cfg, err := lookupConfig(provider, opts)
	if err != nil {
		return Limits{Model: model}, err
	}
	if err := cfg.store.refuseClosed(); err != nil {
		return Limits{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, modelListingTimeout)
	defer cancel()
	var errs []error
	if !cfg.metadataOff {
		if _, err := loadModelMetadata(ctx, cfg); err != nil {
			errs = append(errs, err)
		}
	}
	var key listingKey
	keyed := false
	switch {
	case nilsource.Is(src):
		errs = append(errs, fmt.Errorf("model limits: listing %s: %w: a nil %T is not a TokenSource", id,
			llmprovider.ErrInvalidRequest, src))
	case src != nil && hasListingLayer(id, cfg):
		key, keyed, errs = discoverService(ctx, id, src, model, cfg, errs)
	}
	doc, docOK := peekMetadata(cfg)
	return assembleLimits(id, model, cfg, doc, docOK, key, keyed), errors.Join(errs...)
}

// CachedLimits is LookupLimits without a fetch, as CachedFacts is: what the
// caches, the caller's layers and the extract hold now. It starts none,
// unlike CachedModel, and finds the listing and the reads by the credential
// opts give, as CachedFacts does. It is what a request reads. An id
// LookupLimits refuses gives Limits with nothing known.
func CachedLimits(ctx context.Context, provider llmprovider.ProviderID, model string, opts ...llmprovider.Option) Limits {
	id, cfg, err := lookupConfig(provider, opts)
	if err != nil {
		return Limits{Provider: llmprovider.ProviderID(strings.ToLower(string(provider))), Model: model}
	}
	var key listingKey
	keyed := false
	if cfg.tokens != nil && !nilsource.Is(cfg.tokens) && hasListingLayer(id, cfg) {
		nameSession(cfg.tokens, cfg)
		if token, err := cfg.tokens.Token(ctx); err == nil {
			key, keyed = listingKeyFor(id, cfg, token), true
		}
	}
	doc, docOK := peekMetadata(cfg)
	return assembleLimits(id, model, cfg, doc, docOK, key, keyed)
}

// discoverService reads the service's sources of model: the listing, through
// the listing cache, and the reads D2 adds, each unless a fresh one is
// stored. It returns the listing's key, and errs with each failure added.
func discoverService(ctx context.Context, id llmprovider.ProviderID, src llmprovider.TokenSource, model string, cfg config,
	errs []error) (listingKey, bool, []error) {
	nameSession(src, cfg)
	token, err := src.Token(ctx)
	if err != nil {
		return listingKey{}, false, append(errs, fmt.Errorf("model limits: listing %s: acquire token: %w", id, err))
	}
	key := listingKeyFor(id, cfg, token)
	if id == llmprovider.ProviderOpenAI && (chatgpt.IsSession(src) || chatgpt.IsPlanSession(src)) {
		if !cfg.store.freshRead(limitReadKey{key, model, readBackend}) {
			if err := readBackendWindows(ctx, src, token, cfg, key, model); err != nil {
				errs = append(errs, err)
			}
		}
		return key, true, errs
	}
	_, listed, err := listingModel(ctx, id, src, model, cfg)
	if err != nil {
		errs = append(errs, fmt.Errorf("model limits: listing %s: %w", id, err))
	}
	if ctx.Err() != nil {
		return key, true, errs
	}
	switch {
	case id == llmprovider.ProviderClaude:
		if err == nil && !listed && !cfg.store.freshRead(limitReadKey{key, model, readOneModel}) {
			errs = appendErr(errs, readClaudeModel(ctx, token, cfg, key, model))
		}
	case id == llmprovider.ProviderOllama:
		if !cfg.store.freshRead(limitReadKey{key, model, readServed}) {
			errs = appendErr(errs, readServedContext(ctx, id, token, cfg, ollamaBase(cfg.BaseURL), key, model))
		}
	case cfg.compat && cfg.store.listingDialect(key) == "ollama":
		if !cfg.store.freshRead(limitReadKey{key, model, readServed}) {
			errs = appendErr(errs, readServedContext(ctx, id, token, cfg, ollamaRoot(cfg.BaseURL), key, model))
		}
	}
	return key, true, errs
}

// appendErr is errs with err, when err is not nil.
func appendErr(errs []error, err error) []error {
	if err != nil {
		return append(errs, err)
	}
	return errs
}

// factBuilder collects one fact's evidence, each with its rank.
type factBuilder struct {
	cands []rankedEvidence
}

// rankedEvidence is one statement and its rank.
type rankedEvidence struct {
	Evidence
	rank int
}

// add records a statement of tokens by src at rank.
func (b *factBuilder) add(rank int, src string, tokens int, seen time.Time, stale bool) {
	b.cands = append(b.cands, rankedEvidence{Evidence: Evidence{Source: src, Tokens: tokens, Seen: seen, Stale: stale}, rank: rank})
}

// fact is the chosen value, the highest rank stating more than 0, and every
// statement: the chosen first, then by rank, highest first (0089-MADR D1).
func (b *factBuilder) fact() Fact {
	cands := slices.Clone(b.cands)
	slices.SortStableFunc(cands, func(x, y rankedEvidence) int { return cmp.Compare(y.rank, x.rank) })
	var f Fact
	chosen, values := -1, map[int]bool{}
	for i, c := range cands {
		if c.Tokens <= 0 {
			continue
		}
		values[c.Tokens] = true
		if chosen < 0 {
			chosen = i
		}
	}
	if chosen >= 0 {
		f.Tokens, f.Known, f.Conflicting = cands[chosen].Tokens, true, len(values) > 1
		cands = append(append([]rankedEvidence{cands[chosen]}, cands[:chosen]...), cands[chosen+1:]...)
	}
	for _, c := range cands {
		f.Evidence = append(f.Evidence, c.Evidence)
	}
	return f
}

// limitFacts is the three facts' builders.
type limitFacts struct {
	output, window, input factBuilder
}

// addInfo records what info states of the three limits, by src at rank.
func (b *limitFacts) addInfo(info ModelInfo, rank int, src string, seen time.Time, stale bool) {
	if info.Has(FieldMaxOutputTokens) {
		b.output.add(rank, src, info.MaxOutputTokens, seen, stale)
	}
	if info.Has(FieldContextWindow) {
		b.window.add(rank, src, info.ContextWindow, seen, stale)
	}
	if info.Has(FieldMaxInputTokens) {
		b.input.add(rank, src, info.MaxInputTokens, seen, stale)
	}
}

// assembleLimits is model's Limits from what the caches, the layers, the
// document doc and the extract hold; key is the listing's when keyed.
func assembleLimits(id llmprovider.ProviderID, model string, cfg config, doc Metadata, docOK bool, key listingKey,
	keyed bool) Limits {
	var b limitFacts
	for i, l := range cfg.layers {
		v, ok := l.Model(id, model)
		if !ok {
			continue
		}
		if l.Name() == sourceModelsDev {
			// A models.dev document is registry evidence wherever it came
			// from (0089-MADR D1, DL15).
			b.addInfo(v, rankRegistryLayer+i, l.Name(), time.Time{}, false)
			continue
		}
		b.addInfo(v, rankCaller+i, l.Name(), time.Time{}, false)
	}
	if keyed {
		serviceEvidence(&b, id, model, cfg, key)
	}
	if docOK {
		at := cfg.store.metadataFetchedAt(cfg)
		stale := metadataNow().Sub(at) >= modelMetadataTTL
		registryEvidence(&b, doc, cfg, id, model, rankDocument,
			[3]string{sourceModelsDev, sourceCanonical, sourceFamily}, at, stale)
	}
	ext := bundledExtract()
	registryEvidence(&b, ext.meta, cfg, id, model, rankBundled, ext.names(), ext.date, true)
	l := Limits{Provider: id, Model: model, Output: b.output.fact(), Window: b.window.fact(), Input: b.input.fact(),
		Required: requiresLimit(id, model, doc, docOK), Checked: listingNow()}
	if l.Output.Known && l.Window.Known && l.Output.Tokens > l.Window.Tokens {
		// The output is never above the window (0088 ST91; 0089-MADR DL19).
		l.Output.Tokens, l.Output.Conflicting = l.Window.Tokens, true
	}
	return l
}

// serviceEvidence records what the listing, its extra fields and the reads
// state of model under key (0089-MADR D3).
func serviceEvidence(b *limitFacts, id llmprovider.ProviderID, model string, cfg config, key listingKey) {
	now := listingNow()
	stale := func(at time.Time) bool { return now.Sub(at) >= listingMetadataTTL }
	src := listingLayer(id)
	models, at := cfg.store.listingAt(key)
	info, listed := models[model]
	extra, extraAt, hasExtra := cfg.store.extra(key, listingBaseID(id, model))
	if listed {
		// An extra field states the output where the listing's own does not
		// (Together's), and holds it to the smaller where both do (Kilo's
		// per_request_limits; 0089-MADR DL4, DL9).
		if hasExtra && extra.output > 0 && (!info.Has(FieldMaxOutputTokens) || extra.output < info.MaxOutputTokens) {
			info.MaxOutputTokens = extra.output
			info = info.Known(FieldMaxOutputTokens)
		}
		b.addInfo(info, rankService, src, at, stale(at))
	}
	if w, ok := extra.window(model); hasExtra && ok {
		b.window.add(rankService, src, w, extraAt, stale(extraAt))
	}
	for _, kind := range []readKind{readOneModel, readBackend, readServed} {
		r, ok := cfg.store.read(limitReadKey{key, model, kind})
		if !ok || r.none {
			continue
		}
		if kind == readServed {
			b.window.add(rankServed, "server:"+string(id), r.window, r.at, stale(r.at))
			continue
		}
		if r.output > 0 {
			b.output.add(rankService, src, r.output, r.at, stale(r.at))
		}
		if r.input > 0 {
			b.input.add(rankService, src, r.input, r.at, stale(r.at))
		}
		if r.window > 0 {
			b.window.add(rankService, src, r.window, r.at, stale(r.at))
		}
	}
}

// registryEvidence records what m states of model, for each fact the first
// stage that states it: the exact id, the canonical id, then the family
// (0089-MADR D3, DL71). A chatcompat server's model is read by canonical id
// alone, as modelFor reads it. names are the stages' source names.
func registryEvidence(b *limitFacts, m Metadata, cfg config, id llmprovider.ProviderID, model string, rank int,
	names [3]string, seen time.Time, stale bool) {
	if m.doc == nil {
		return
	}
	type stage struct {
		info ModelInfo
		name string
	}
	var stages []stage
	if !cfg.compat {
		if info, ok := m.Model(id, model); ok {
			stages = append(stages, stage{info, names[0]})
		}
	}
	// A provider with no models.dev section, Ollama, has no registry
	// evidence: its server states its window (0089-MADR DL13).
	if info, ok := m.canonicalModel(model); ok && (cfg.compat || modelMetadataKey(id) != "") {
		stages = append(stages, stage{info, names[1]})
	}
	if family, ok := familyID(model); ok && !cfg.compat {
		if info, ok := m.Model(id, family); ok {
			stages = append(stages, stage{info, names[2]})
		}
	}
	for _, f := range []Field{FieldMaxOutputTokens, FieldContextWindow, FieldMaxInputTokens} {
		for j, s := range stages {
			if !s.info.Has(f) {
				continue
			}
			one := ModelInfo{layer: s.name}
			copyField(&one, s.info, f)
			one = one.Known(f)
			// Within a source, an earlier stage ranks above a later one.
			b.addInfo(one, rank-j, s.name, seen, stale)
			break
		}
	}
}

// datedSuffix is a dated model id's date, -YYYYMMDD.
var datedSuffix = regexp.MustCompile(`-\d{8}$`)

// familyID is model without its routing suffix after ":" or its dated
// suffix, and whether it had one (0089-MADR D3).
func familyID(model string) (string, bool) {
	if i := strings.LastIndex(model, ":"); i > 0 {
		return model[:i], true
	}
	if loc := datedSuffix.FindStringIndex(model); len(loc) == 2 && loc[0] > 0 {
		return model[:loc[0]], true
	}
	return "", false
}

// listingBaseID is the id a listing's extra fields are kept under: Hugging
// Face's without its ":provider" or ":policy" suffix.
func listingBaseID(id llmprovider.ProviderID, model string) string {
	if id == llmprovider.ProviderHuggingFace {
		if base, _, ok := strings.Cut(model, ":"); ok {
			return base
		}
	}
	return model
}

// requiresLimit reports whether model's wire requires an output limit:
// Claude's, and OpenCode's messages route, read from the document's
// provider.npm when it lists the model, else from OpenCode's route table,
// as the provider routes a request (0089-MADR DL30, DL74).
func requiresLimit(id llmprovider.ProviderID, model string, doc Metadata, docOK bool) bool {
	switch id {
	case llmprovider.ProviderClaude:
		return true
	case llmprovider.ProviderOpencodeZen, llmprovider.ProviderOpencodeGo:
		route := opencoderoute.Of(id, model)
		if docOK {
			if npm, ok := doc.NPM(id, model); ok {
				route = opencoderoute.ForNPM(npm)
			}
		}
		return route == opencoderoute.Messages
	}
	return false
}
