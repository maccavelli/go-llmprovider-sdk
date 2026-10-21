package catalog

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"math"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/kiloendpoint"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/listlimit"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/nilsource"
)

// Listing pagination (MADR 0007 §2): Gemini and Anthropic page their model
// lists, so each fetch requests the maximum page size and follows at most
// maxListingPages pages.
const (
	maxListingPages     = 10
	geminiListPageSize  = "1000"
	claudeListPageLimit = "1000"
)

// listingPageLimit bounds one listing page, from every lister (0021-MADR
// C3). Together's listing, which also lists image, audio and embedding
// models, set the figure.
const listingPageLimit = 8 << 20

// The first-party listings' default bases. Claude's is the host: its lister
// reads /v1/models under it.
const (
	openAIListingBase = "https://api.openai.com/v1"
	claudeListingHost = "https://api.anthropic.com"
	geminiListingBase = "https://generativelanguage.googleapis.com/v1beta"
	xaiListingBase    = "https://api.x.ai/v1"
	ollamaListingBase = "http://localhost:11434"
)

// listingBaseURL is the base id's listing is read from, given base, the
// caller's WithBaseURL: base itself, or the provider's default when it is
// empty; for Claude, claudeListingBase's host. The listing cache is keyed by
// it, so a listing with no base URL and a provider's request, which names
// its provider's default, share one entry (0041-MADR A8).
func listingBaseURL(id llmprovider.ProviderID, base string) string {
	if id == llmprovider.ProviderClaude {
		return claudeListingBase(base)
	}
	if base != "" {
		return base
	}
	switch id {
	case llmprovider.ProviderOpenAI:
		return openAIListingBase
	case llmprovider.ProviderGemini:
		return geminiListingBase
	case llmprovider.ProviderGrok:
		return xaiListingBase
	case llmprovider.ProviderOllama:
		return ollamaListingBase
	case llmprovider.ProviderKilo:
		return kiloendpoint.BaseURL
	case llmprovider.ProviderHuggingFace:
		return huggingFaceBaseURL
	case llmprovider.ProviderTogether:
		return togetherBaseURL
	}
	gateway, err := opencodeBaseURL(id)
	if err != nil {
		return ""
	}
	return gateway
}

// claudeListingBase is the host a Claude listing reads /v1/models from:
// base without /v1, so that the Claude provider's base, which ends in /v1,
// and a host both list (0041-MADR A8); or Anthropic's. The options have
// already trimmed base's trailing slash.
func claudeListingBase(base string) string {
	if base == "" {
		return claudeListingHost
	}
	return strings.TrimSuffix(base, "/v1")
}

// decodeLimited decodes one listing page's JSON value from r into v, reading
// at most listingPageLimit bytes. A longer body is an error naming the limit;
// what labels the errors (0021-MADR C3).
func decodeLimited(r io.Reader, v any, what string) error {
	const limit = listingPageLimit
	raw, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return fmt.Errorf("%s: read: %w", what, err)
	}
	if int64(len(raw)) > limit {
		return fmt.Errorf("%s: the reply is larger than %d MiB", what, limit>>20)
	}
	if err := json.Unmarshal(raw, v); err != nil {
		return fmt.Errorf("%s: decode: %w", what, err)
	}
	return nil
}

// modelListingTimeout bounds one model listing, its metadata fetch included
// (MADR 0009 §2). List, and so every provider's ListModels, applies it
// (MADR 0013 A5).
const modelListingTimeout = 10 * time.Second

// Catalog is the result of one model listing, viewed two ways. A Catalog
// shares nothing with the cache or with another Catalog, so it is safe for
// concurrent use only while no goroutine changes it.
type Catalog struct {
	// Recommended is what List returns: the recommended band, then every
	// other usable id without a caveat, newest first, or its first n with
	// WithLimit(n) (0038-MADR D1, D2, A5; 0041-MADR D3, A4). It is Menu's
	// ids less the caveated ones, except that Ollama's keeps those whose only
	// caveat is CaveatNoTools (0042-MADR D2). It may be empty on a live
	// listing, when every listed model has a caveat; Usable then holds the
	// listing to search (0021-MADR C13).
	Recommended []string
	// Usable is every id the provider's usability filters admit, once each,
	// uncapped, in the order its lister gives: the listing's own for most,
	// cheapest first for Kilo, and fastest first for Hugging Face (MADR 0007's
	// table; 0026-MADR F54). It equals Recommended when Live is false.
	// Ollama's admits each installed model that writes text, or whose output
	// Ollama does not report (0042-MADR D1).
	Usable []string
	// Live reports whether Usable came from the provider's listing rather
	// than the static catalog. A stale listing is live: it came from the
	// provider, at FetchedAt (Source ListStale; 0041-MADR D6).
	Live bool
	// Err is the listing failure: the one that made Live false, or, on a
	// stale listing, the one the last good listing stands in for. It is nil
	// on a listing read now or cached, and when a listing that succeeded
	// yielded no usable id (MADR 0013 C2).
	Err error
	// Menu is every usable model, each with its facts, its rank and its
	// caveats: the band by rank, then the other models without a caveat,
	// newest first by release date, then by id, then the caveated ones, in
	// the same order (0041-MADR D3, A4). Ollama's keeps /api/tags order. On
	// a static catalog it is the static ids, in order. WithLimit(n) cuts it
	// as it cuts Recommended.
	Menu []MenuEntry
	// Source is where the listing came from: the provider's listing, read now
	// or within the listing cache's ten minutes (ListLive), the last good
	// listing after a transient failure (ListStale), or the static catalog
	// (ListStatic) (0041-MADR D6, A6). It is empty on a Catalog List did not
	// build.
	Source ListSource
	// FetchedAt is when the listing was read from the provider: now, or
	// earlier for a cached or stale one. It is zero for the static catalog.
	FetchedAt time.Time
}

// ListSource says where a Catalog's listing came from (0041-MADR D6). It is a
// string, safe for concurrent use.
type ListSource string

// The sources of a listing.
const (
	// ListLive is the provider's listing, read now or within the listing
	// cache's ten minutes.
	ListLive ListSource = "live"
	// ListStale is the last good listing, at most 24 hours old, served after
	// a transient failure: a timeout, a 5xx, a rate limit or a network error.
	ListStale ListSource = "stale"
	// ListStatic is the static catalog, served when no good listing exists.
	ListStatic ListSource = "static"
)

// List performs one model listing for id and returns both the curated
// recommendation and every usable id (MADR 0007 §1). src is the credential: a
// key is llmprovider.NewStaticToken(key). It curates the listing against the
// static catalog so configure UIs never show huge unusable lists (embeddings,
// TTS, Live, image, dated previews, …), spends no generation, and is bounded
// at 10 s.
//
// opts are llmprovider's common options, with WithProfile, WithLimit,
// WithWorkload, WithLastGood and WithKiloOrganization.
//
// A listing read within the last ten minutes, for the same provider, base
// URL, Kilo organization and credential, is ranked again with no request,
// and callers at once share one fetch, and its result, a failure included;
// a caller that is cancelled ends only its own wait (0041-MADR D6, A6;
// 0053-MADR D4). A transient
// failure (a timeout, a 5xx, a rate limit, a network error) serves the last
// good listing, at most 24 hours old: Catalog.Source is ListStale, Live
// true, and Err the failure. Any other failed listing degrades to the
// static catalog with a nil error and Live false, and its classified
// failure in Catalog.Err. Ollama is listed every time and never served
// stale. Only Ollama, an unknown provider, a missing source, a token
// failure or a refused option return an error. A missing src, and a typed-nil
// one such as a nil *llmprovider.CommandToken held in a non-nil TokenSource,
// give an error matching llmprovider.ErrInvalidRequest (0050-MADR D4). A
// listing refused for its credential renews src once, when src is an
// llmprovider.InvalidatingSource, and lists again (0026-MADR F36).
func List(ctx context.Context, id llmprovider.ProviderID, src llmprovider.TokenSource, opts ...llmprovider.Option) (Catalog, error) {
	// The id first: an option scoped to the built-in ids would otherwise
	// refuse an unknown one with ErrInvalidRequest (0015-PLAN S12b).
	if !listed[llmprovider.ProviderID(strings.ToLower(string(id)))] {
		return Catalog{}, unlistedProvider(id)
	}
	// Ids are matched without case, options included (0020-MADR F35).
	id = llmprovider.ProviderID(strings.ToLower(string(id)))
	cfg, err := configFor(id, opts)
	if err != nil {
		return Catalog{}, err
	}

	cat, _, err := listAndRecord(ctx, id, src, cfg)
	return cat, err
}

// listAndRecord is List's listing, after its id and options are checked: it
// lists, renews a refused credential once, and returns what the listing said
// of each model besides the catalog (0033-MADR D2).
func listAndRecord(ctx context.Context, id llmprovider.ProviderID, src llmprovider.TokenSource, cfg config) (Catalog, map[string]ModelInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, modelListingTimeout)
	defer cancel()
	token, err := listingToken(ctx, src, cfg)
	if err != nil {
		return Catalog{}, nil, err
	}
	cat, models, err := listOnce(ctx, id, token, cfg)
	// A listing that still fails after the renewal keeps the
	// degrade-to-static contract.
	token, renewed, err2 := renewRefused(ctx, src, token, cmp.Or(err, cat.Err))
	if err2 != nil {
		return Catalog{}, nil, err2
	}
	if !renewed {
		return cat, models, err
	}
	return listOnce(ctx, id, token, cfg)
}

// listingToken is src's token for a listing. List and ListRaw share it.
func listingToken(ctx context.Context, src llmprovider.TokenSource, cfg config) (llmprovider.Token, error) {
	// A missing source and a typed nil, such as a nil *CommandToken in a
	// non-nil TokenSource, are both the caller's to fix (0050-MADR D4).
	if src == nil {
		return llmprovider.Token{}, fmt.Errorf("%w: model listing: TokenSource is required", llmprovider.ErrInvalidRequest)
	}
	if nilsource.Is(src) {
		return llmprovider.Token{}, fmt.Errorf("%w: model listing: a nil %T is not a TokenSource", llmprovider.ErrInvalidRequest, src)
	}
	// A session with no client refreshes through the listing's, so a
	// refresh the listing causes keeps the caller's transport (0026-MADR
	// F10). It is given as a default: a provider passes its own client here,
	// default or not, and a caller's client given later replaces it (F37).
	if user, ok := src.(defaultClientUser); ok {
		user.UseDefaultHTTPClient(cfg.HTTPClient)
	}
	token, err := src.Token(ctx)
	if err != nil {
		return llmprovider.Token{}, fmt.Errorf("model listing: acquire token: %w", err)
	}
	return token, nil
}

// renewRefused renews src's token once when failure says the credential was
// refused, as Generate renews it (0026-MADR F36), and reports whether it did.
// List and ListRaw share it.
func renewRefused(ctx context.Context, src llmprovider.TokenSource, token llmprovider.Token,
	failure error) (llmprovider.Token, bool, error) {
	source, ok := src.(llmprovider.InvalidatingSource)
	if !ok || !credentialRefused(failure) {
		return token, false, nil
	}
	if invalidator, ok := src.(llmprovider.TokenInvalidator); ok {
		invalidator.InvalidateToken(token)
	} else {
		source.Invalidate()
	}
	renewed, err := src.Token(ctx)
	if err != nil {
		return token, false, fmt.Errorf("model listing: acquire token: %w", err)
	}
	return renewed, true, nil
}

// listOnce lists with token, and returns what the listing says of each
// model besides the catalog. Ollama lists every time (listOllamaOnce). Every
// other provider lists through the listing cache (0041-MADR D6, A6): a
// listing read within listingMetadataTTL is built again with no request;
// otherwise one fetch per key runs at a time, and a transient failure
// serves the last good listing, read within lastGoodWindow, or WithLastGood's.
func listOnce(ctx context.Context, id llmprovider.ProviderID, token llmprovider.Token, cfg config) (Catalog, map[string]ModelInfo, error) {
	id = llmprovider.ProviderID(strings.ToLower(string(id)))
	if !listed[id] {
		return Catalog{}, nil, unlistedProvider(id)
	}
	if id == llmprovider.ProviderOllama {
		return listOllamaOnce(ctx, token, cfg)
	}
	key := listingKeyFor(id, cfg, token)
	var meta func() modelMetadataResult
	if id != llmprovider.ProviderKilo {
		// The document is read alongside the listing; Kilo's ranking reads
		// its own listing (MADR 0009 §2).
		meta = startModelMetadata(ctx, cfg)
	}
	f, models, at, failure := cachedOrFetched(ctx, id, token, cfg, key)
	source := ListLive
	if failure != nil && transientListingFailure(failure) {
		if last, lastAt, ok := cfg.store.lastGoodListing(key, cfg); ok {
			f, at, source = last, lastAt, ListStale
			models, _ = cfg.store.cachedListing(key)
		}
	}
	l := buildListing(id, f, cfg, meta)
	if source == ListLive {
		l.err = failure
	}
	cat := l.catalog(id, token, cfg, models)
	switch {
	case source == ListStale:
		cat.Source, cat.FetchedAt, cat.Err = ListStale, at, failure
	case cat.Live:
		cat.Source, cat.FetchedAt = ListLive, at
	default:
		cat.Source = ListStatic
	}
	return cat, models, nil
}

// listOllamaOnce is Ollama's listing, outside the listing cache: it lists every
// time, never serves a stale listing, and returns its errors (0041-MADR A6).
// Its /api/show replies are cached, as before (0041-MADR A4).
func listOllamaOnce(ctx context.Context, token llmprovider.Token, cfg config) (Catalog, map[string]ModelInfo, error) {
	models := map[string]ModelInfo{}
	cfg.sink = func(info ModelInfo) { models[info.ID] = info }
	l, err := ollamaCatalog(ctx, token, cfg)
	if err != nil {
		return Catalog{}, models, err
	}
	undatedConstant(models)
	cat := l.catalog(llmprovider.ProviderOllama, token, cfg, models)
	cat.Source, cat.FetchedAt = ListLive, listingNow()
	if cat.Err == nil && cat.Live && len(models) > 0 {
		cfg.store.storeListing(listingKeyFor(llmprovider.ProviderOllama, cfg, token), models, false)
	}
	return cat, models, nil
}

// cachedOrFetched is key's listing: the cached one while it is fresh, else
// the one fetch per key in flight, started by the first caller (0041-MADR
// D6). The fetch is detached from every caller and bounded as a listing is,
// and each caller waits for it under its own ctx and receives its result, a
// failure included, so N callers cost one request and a caller that is
// cancelled ends only its own wait (0053-MADR D4). models is what the listing
// says of each model, and at is when it was read. failure is the fetch's
// failure, or ctx's while waiting; f is then empty.
func cachedOrFetched(ctx context.Context, id llmprovider.ProviderID, token llmprovider.Token, cfg config,
	key listingKey) (f fetchedListing, models map[string]ModelInfo, at time.Time, failure error) {
	if f, at, ok := cfg.store.freshListing(key); ok {
		models, _ := cfg.store.cachedListing(key)
		return f, models, at, nil
	}
	call, leader := cfg.store.claimListing(key)
	if leader {
		fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), modelListingTimeout)
		cfg.store.listingFetching.Go(func() {
			defer cancel()
			call.f, call.models, call.at, call.err = fetchClaimed(fetchCtx, id, token, cfg, key)
			cfg.store.finishListing(key, call)
		})
	}
	select {
	case <-call.done:
		// Each caller keeps its own copy of what the models say.
		return call.f, maps.Clone(call.models), call.at, call.err
	case <-ctx.Done():
		return fetchedListing{}, nil, time.Time{}, ctx.Err()
	}
}

// fetchClaimed fetches the listing a call's leader claimed, and records it
// when it succeeds.
func fetchClaimed(ctx context.Context, id llmprovider.ProviderID, token llmprovider.Token, cfg config,
	key listingKey) (fetchedListing, map[string]ModelInfo, time.Time, error) {
	models := map[string]ModelInfo{}
	cfg.sink = func(info ModelInfo) { models[info.ID] = info }
	f, failure := fetchListing(ctx, id, token, cfg)
	if failure != nil {
		return fetchedListing{}, models, time.Time{}, failure
	}
	undatedConstant(models)
	return f, models, cfg.store.storeFetched(key, f, models), nil
}

// transientListingFailure reports whether a listing's failure may serve the
// last good listing: a timeout, a 5xx or 408, a rate limit, or a network
// error (0041-MADR D6). A refused credential, a refused request, an
// exhausted quota and anything unclassified fall to the static list.
func transientListingFailure(err error) bool {
	var netErr net.Error
	return errors.Is(err, llmprovider.ErrProviderUnavailable) || errors.Is(err, llmprovider.ErrRateLimited) ||
		errors.Is(err, context.DeadlineExceeded) || errors.As(err, &netErr)
}

// undatedConstant drops the Released of a listing whose dated rows all
// share one date, as Zen's and Go's do: such a date is when the listing was
// built, not when a model was released (0041-MADR D1). It needs two dated
// rows to tell.
func undatedConstant(models map[string]ModelInfo) {
	var first time.Time
	dated, same := 0, true
	for _, m := range models {
		if !m.Has(FieldReleased) {
			continue
		}
		if dated == 0 {
			first = m.Released
		} else if !m.Released.Equal(first) {
			same = false
		}
		dated++
	}
	if dated < 2 || !same {
		return
	}
	for id, m := range models {
		if m.Has(FieldReleased) {
			models[id] = m.without(FieldReleased)
		}
	}
}

// defaultClientUser is a source that refreshes through a client it is given,
// such as *auth.OAuthSession.
type defaultClientUser interface {
	UseDefaultHTTPClient(*http.Client)
}

// credentialRefused reports whether a listing's failure says its credential
// was refused, by internal/wire's Reauth rule: an *APIError whose kind is
// ErrAuthFailure, whatever its status, or a 401 with no kind (0026-MADR F6,
// F36).
func credentialRefused(err error) bool {
	apiErr, ok := errors.AsType[*llmprovider.APIError](err)
	switch {
	case !ok:
		return false
	case apiErr.Kind == nil:
		return apiErr.Status == http.StatusUnauthorized
	default:
		return apiErr.Kind == llmprovider.ErrAuthFailure //nolint:errorlint // the kind itself, not what it wraps
	}
}

// curator curates a listing's usable ids, best first, to at most limit, or
// every one it recommends when limit is zero or less.
type curator func(usable []string, limit int) []string

// listingResult is one lister's result, before its catalog is built: the usable
// ids, the listing's failure, the static catalog, the curator that ranks
// the band, and the document the curator ranks with (0041-MADR D3).
type listingResult struct {
	usable []string
	err    error
	static []string
	// curate is nil for Ollama, which has no band.
	curate curator
	// meta is nil where the lister fetches no document, as Kilo's does not.
	meta func() modelMetadataResult
	// listed is what an Ollama listing read of each model, its /api/tags
	// entry under its /api/show reply, cached or not; nil for the other
	// listers, whose facts are the ones they emit (0042-MADR D1).
	listed map[string]ModelInfo
}

// catalog applies the degrade-to-static contract every lister except
// Ollama has always had: a failed fetch, or one that yields no usable id,
// substitutes the static catalog. Otherwise it builds the menu from the
// listing's facts, the document and the caller's layers. A listing whose
// every model has a caveat keeps the live listing: the user searches it
// (0021-MADR C13).
func (l listingResult) catalog(id llmprovider.ProviderID, token llmprovider.Token, cfg config,
	models map[string]ModelInfo) Catalog {
	now := rankingNow()
	if l.curate == nil {
		return ollamaMenu(l.usable, factsOf(id, cfg, l.usable, l.listed, nil), cfg.limit, now)
	}
	usable := uniqueIDs(l.usable)
	if l.err != nil || len(usable) == 0 {
		// What facts are cached: the static path starts no wait.
		cached, _ := cfg.store.cachedListing(listingKeyFor(id, cfg, token))
		var doc *modelMetadataDoc
		if m, ok := peekMetadata(cfg); ok {
			doc = m.doc
		}
		cat := staticMenu(id, l.static, factsOf(id, cfg, l.static, cached, doc), cfg.limit, now)
		cat.Err = l.err
		return cat
	}
	var doc *modelMetadataDoc
	if l.meta != nil {
		doc = l.meta().doc
	}
	if m, ok := peekMetadata(cfg); doc == nil && ok {
		doc = m.doc
	}
	return menuOf(id, usable, factsOf(id, cfg, usable, models, doc), l.curate, cfg.limit, now)
}

// uniqueIDs drops repeated ids, keeping the first of each (MADR 0013 A1).
func uniqueIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, dup := seen[id]; !dup {
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
}

// staticCatalog wraps a caller-owned copy of a static catalog, cut to limit
// in both views, which stay equal (0038-MADR D3).
func staticCatalog(static []string, limit int) Catalog {
	static = listlimit.Cut(static, limit)
	return Catalog{Recommended: static, Usable: slices.Clone(static), Live: false}
}

// firstPartyUsable is a first-party provider's usable listed models.
func firstPartyUsable(ctx context.Context, p llmprovider.ProviderID, token llmprovider.Token, cfg config) ([]string, error) {
	switch p {
	case llmprovider.ProviderOpenAI:
		return fetchOpenAIUsable(ctx, token, cfg)
	case llmprovider.ProviderClaude:
		return fetchClaudeUsable(ctx, token, cfg)
	case llmprovider.ProviderGemini:
		return fetchGeminiUsable(ctx, token, cfg)
	default:
		return fetchGrokUsable(ctx, token, cfg)
	}
}

// firstPartyCurator is a first-party provider's name curator: its whole
// ranking before 0041-MADR D4, and its fallback with no document since.
func firstPartyCurator(p llmprovider.ProviderID) curator {
	switch p {
	case llmprovider.ProviderOpenAI:
		return curateOpenAI
	case llmprovider.ProviderClaude:
		return curateClaude
	case llmprovider.ProviderGemini:
		return curateGemini
	default:
		return curateGrok
	}
}

// fetchedListing is what one provider's fetch returned: all its build needs
// (0041-MADR A6). It is what the listing cache keeps.
type fetchedListing struct {
	// usable is the listing's usable ids, in the lister's order.
	usable []string
	// kilo is Kilo's entries, whose prices and dates its ranking reads.
	kilo []kiloCatalogEntry
}

// fetchListing sends one provider's listing requests, Ollama's aside, and
// returns what buildListing needs (0041-MADR D6, A6). The caller owns the
// timeout. Every lister sends token as it describes itself (0016-MADR A6).
// A failed fetch degrades to the static catalog rather than failing: see
// listingResult.catalog.
func fetchListing(ctx context.Context, id llmprovider.ProviderID, token llmprovider.Token, cfg config) (fetchedListing, error) {
	var usable []string
	var err error
	switch id {
	case llmprovider.ProviderGemini, llmprovider.ProviderOpenAI, llmprovider.ProviderClaude, llmprovider.ProviderGrok:
		usable, err = firstPartyUsable(ctx, id, token, cfg)
	case llmprovider.ProviderOpencodeZen, llmprovider.ProviderOpencodeGo:
		usable, err = fetchOpencodeUsable(ctx, id, token, cfg)
	case llmprovider.ProviderHuggingFace:
		usable, err = fetchHuggingFaceUsable(ctx, token, cfg)
	case llmprovider.ProviderKilo:
		entries, err := fetchKiloCatalog(ctx, token, cfg)
		return fetchedListing{usable: kiloUsable(entries), kilo: entries}, err
	case llmprovider.ProviderTogether:
		usable, err = fetchTogetherUsable(ctx, token, cfg)
	default:
		return fetchedListing{}, unlistedProvider(id)
	}
	return fetchedListing{usable: usable}, err
}

// buildListing is one provider's listingResult from f: a listing fetched
// now, cached, or kept as the last good one. It sends no listing request:
// the caller's profile, limit, workload and layers rank it as they rank a
// live listing (0041-MADR D6, A6). meta is the models.dev document's, and
// nil for Kilo, which ranks from its own entries (MADR 0009 §2).
func buildListing(id llmprovider.ProviderID, f fetchedListing, cfg config, meta func() modelMetadataResult) listingResult {
	l := listingResult{usable: f.usable, static: Static(id), meta: meta}
	switch id {
	case llmprovider.ProviderGemini, llmprovider.ProviderOpenAI, llmprovider.ProviderClaude, llmprovider.ProviderGrok:
		// Ranked by facts, as every provider is; the name curator ranks with
		// no document (0041-MADR D4).
		l.curate = metadataCurate(id, cfg.ModelProfile, cfg.workload, meta, firstPartyCurator(id))
	case llmprovider.ProviderOpencodeZen, llmprovider.ProviderOpencodeGo:
		// The OpenCode listing carries no routing or capability metadata, so
		// the curation leaves out, by name, what a request would fail on
		// (0021-MADR D3); curateFromCatalog does so for every provider
		// (0038-MADR D1).
		l.curate = metadataCurate(id, cfg.ModelProfile, cfg.workload, meta, func(usable []string, limit int) []string {
			return curateFromCatalog(id, staticOpencodeCatalog(id), usable, isUsableOpencodeModel, rankOpencodeModel, limit)
		})
	case llmprovider.ProviderHuggingFace:
		l.curate = metadataCurate(id, cfg.ModelProfile, cfg.workload, meta, curateHuggingFace)
	case llmprovider.ProviderKilo:
		l.curate = kiloCurate(f.kilo, cfg.ModelProfile, cfg.workload)
	case llmprovider.ProviderTogether:
		l.curate = metadataCurate(id, cfg.ModelProfile, cfg.workload, meta, curateTogether)
	}
	return l
}

// listed holds the ids List lists.
var listed = map[llmprovider.ProviderID]bool{
	llmprovider.ProviderGemini: true, llmprovider.ProviderOpenAI: true, llmprovider.ProviderClaude: true,
	llmprovider.ProviderGrok: true, llmprovider.ProviderOpencodeZen: true, llmprovider.ProviderOpencodeGo: true,
	llmprovider.ProviderHuggingFace: true, llmprovider.ProviderKilo: true, llmprovider.ProviderTogether: true,
	llmprovider.ProviderOllama: true,
}

// unlistedProvider is List's error for an id it does not list. It matches
// ErrUnsupported, so a caller such as wizard can list the provider another
// way (0015-MADR, amendment "the wizard lists a provider through its own
// ListModels").
func unlistedProvider(id llmprovider.ProviderID) error {
	return fmt.Errorf("model listing: unsupported provider %q: %w", id, llmprovider.ErrUnsupported)
}

// geminiModelsPage is one page of Gemini's GET {base}/models.
type geminiModelsPage struct {
	Models        []geminiModel `json:"models"`
	NextPageToken string        `json:"nextPageToken"`
}

// geminiModel is one entry of Gemini's listing: its id and methods, and the
// limits and thinking flag 0033-MADR D2 reads.
type geminiModel struct {
	Name                       string   `json:"name"`
	SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
	InputTokenLimit            *int     `json:"inputTokenLimit"`
	OutputTokenLimit           *int     `json:"outputTokenLimit"`
	Thinking                   *bool    `json:"thinking"`
}

// fetchGeminiUsable returns the usable Gemini text models in listing order,
// following nextPageToken for at most maxListingPages pages (MADR 0007 §2).
// Any page failure, or running out of pages, fails the whole listing.
func fetchGeminiUsable(ctx context.Context, token llmprovider.Token, cfg config) ([]string, error) {
	baseURL := geminiListingBase
	if cfg.BaseURL != "" {
		baseURL = cfg.BaseURL
	}

	var available []string
	pageToken := ""
	for page := 1; page <= maxListingPages; page++ {
		query := url.Values{"pageSize": {geminiListPageSize}}
		if pageToken != "" {
			query.Set("pageToken", pageToken)
		}
		endpoint := baseURL + "/models?" + query.Encode()
		result, err := fetchGeminiPage(ctx, endpoint, token, cfg)
		if err != nil {
			return nil, err
		}
		for _, m := range result.Models {
			id := strings.TrimPrefix(m.Name, "models/")
			cfg.emit(geminiInfo(id, m))
			if isUsableGeminiTextModel(id, m.SupportedGenerationMethods) {
				available = append(available, id)
			}
		}
		if result.NextPageToken == "" {
			return available, nil
		}
		pageToken = result.NextPageToken
	}
	return nil, fmt.Errorf("model listing: more than %d pages", maxListingPages)
}

// fetchGeminiPage performs one listing request. The key travels in the
// x-goog-api-key header, or the token's own, never the query string.
func fetchGeminiPage(ctx context.Context, endpoint string, token llmprovider.Token, cfg config) (geminiModelsPage, error) {
	var result geminiModelsPage
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, http.NoBody)
	if err != nil {
		return result, err
	}
	cfg.setUserAgent(req)
	token.Apply(req, "x-goog-api-key", "")

	resp, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return result, err
	}
	defer cfg.closeBody(resp)

	if resp.StatusCode != http.StatusOK {
		return result, llmprovider.ClassifyHTTPError(string(llmprovider.ProviderGemini), resp)
	}
	err = decodeLimited(resp.Body, &result, "gemini: models")
	return result, err
}

func curateGemini(usable []string, limit int) []string {
	return curateFromCatalog(llmprovider.ProviderGemini, staticGemini, usable, func(s string) bool {
		return isUsableGeminiTextModel(s, []string{methodGenerateContent})
	}, rankGeminiModel, limit)
}

// fetchOpenAIUsable returns the usable OpenAI chat models in listing order.
func fetchOpenAIUsable(ctx context.Context, token llmprovider.Token, cfg config) ([]string, error) {
	baseURL := openAIListingBase
	if cfg.BaseURL != "" {
		baseURL = cfg.BaseURL
	}
	header, value := tokenHeader(token, headerAuthorization, bearerScheme)
	ids, err := fetchDataIDs(ctx, baseURL+"/models", header, value, cfg, llmprovider.ProviderOpenAI)
	if err != nil {
		return nil, err
	}
	return filterIDs(ids, isUsableOpenAIChatModel), nil
}

func curateOpenAI(usable []string, limit int) []string {
	return curateFromCatalog(llmprovider.ProviderOpenAI, staticOpenAI, usable, isUsableOpenAIChatModel, rankOpenAIModel, limit)
}

// claudeModelsPage is one page of Anthropic's GET /v1/models.
type claudeModelsPage struct {
	Data    []claudeModel `json:"data"`
	HasMore bool          `json:"has_more"`
	LastID  string        `json:"last_id"`
}

// claudeModel is one entry of Anthropic's listing: its id, and the limits and
// capabilities 0033-MADR D2 reads. Its lifecycle is not read: it lags the
// deprecations page (0033-MADR, "sources that lag").
type claudeModel struct {
	ID string `json:"id"`
	// CreatedAt is when Anthropic lists the model as released, RFC 3339.
	CreatedAt      string              `json:"created_at"`
	MaxInputTokens *int                `json:"max_input_tokens"`
	MaxTokens      *int                `json:"max_tokens"`
	Capabilities   *claudeCapabilities `json:"capabilities"`
}

// fetchClaudeUsable returns the usable Claude text models in listing order,
// following has_more/last_id for at most maxListingPages pages (MADR 0007 §2).
// Any page failure, or running out of pages, fails the whole listing.
func fetchClaudeUsable(ctx context.Context, token llmprovider.Token, cfg config) ([]string, error) {
	baseURL := claudeListingBase(cfg.BaseURL)

	var available []string
	afterID := ""
	for page := 1; page <= maxListingPages; page++ {
		query := url.Values{"limit": {claudeListPageLimit}}
		if afterID != "" {
			query.Set("after_id", afterID)
		}
		result, err := fetchClaudePage(ctx, baseURL+"/v1/models?"+query.Encode(), token, cfg)
		if err != nil {
			return nil, err
		}
		for _, m := range result.Data {
			cfg.emit(claudeInfo(m))
			if isUsableClaudeTextModel(m.ID) {
				available = append(available, m.ID)
			}
		}
		if !result.HasMore {
			return available, nil
		}
		if result.LastID == "" {
			return nil, errors.New("model listing: has_more without last_id")
		}
		afterID = result.LastID
	}
	return nil, fmt.Errorf("model listing: more than %d pages", maxListingPages)
}

// fetchClaudePage performs one listing request.
func fetchClaudePage(ctx context.Context, endpoint string, token llmprovider.Token, cfg config) (claudeModelsPage, error) {
	var result claudeModelsPage
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, http.NoBody)
	if err != nil {
		return result, err
	}
	cfg.setUserAgent(req)
	token.Apply(req, "x-api-key", "")
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return result, err
	}
	defer cfg.closeBody(resp)

	if resp.StatusCode != http.StatusOK {
		// Older keys / regional proxies may not support Models API.
		return result, llmprovider.ClassifyHTTPError(string(llmprovider.ProviderClaude), resp)
	}
	err = decodeLimited(resp.Body, &result, "claude: models")
	return result, err
}

func curateClaude(usable []string, limit int) []string {
	return curateFromCatalog(llmprovider.ProviderClaude, staticClaude, usable, isUsableClaudeTextModel, rankClaudeModel, limit)
}

// ollamaCatalog lists every installed model, and reads /api/show for each
// one whose reply is not cached, for its caveats (0041-MADR A4). Only the
// replies are emitted, and so cached. A model's /api/tags capabilities say
// what it writes where its reply does not, for this listing alone, so a
// failed reply is asked again the next time (0042-MADR D1). Ollama has no
// static catalog, so its errors are returned rather than degraded, and an
// empty install is a successful (Live) listing.
func ollamaCatalog(ctx context.Context, token llmprovider.Token, cfg config) (listingResult, error) {
	tags, err := fetchOllamaTags(ctx, token, cfg)
	if err != nil {
		return listingResult{}, err
	}
	var names []string
	digests := make(map[string]string, len(tags))
	for _, t := range tags {
		names = append(names, t.Name)
		digests[t.Name] = t.Digest
	}
	// Each reply keeps its own age and digest: a listing reuses one only
	// while it is younger than listingMetadataTTL and its model's digest is
	// unchanged, and stores only the replies it fetched (0053-MADR D5).
	key := listingKeyFor(llmprovider.ProviderOllama, cfg, token)
	cached := cfg.store.freshShows(key, digests)
	shows := ollamaShows(ctx, token, cfg, names, cached)
	for name, info := range shows {
		if _, reused := cached[name]; !reused {
			cfg.store.storeShow(key, name, info, digests[name])
		}
	}
	listed := make(map[string]ModelInfo, len(tags))
	for _, t := range tags {
		info := ollamaTagInfo(t)
		if show, ok := shows[t.Name]; ok {
			cfg.emit(show)
			overlay(&info, show, "")
		}
		listed[t.Name] = info
	}
	// Recommended never shares Usable's array (0020-MADR F34).
	return listingResult{usable: names, listed: listed}, nil
}

// ollamaTag is one /api/tags entry: the model's name, its digest, which a
// cached /api/show reply is checked against (0053-MADR D5), and the
// capabilities Ollama v0.30.0 and later list beside it.
type ollamaTag struct {
	Name         string   `json:"name"`
	Digest       string   `json:"digest"`
	Capabilities []string `json:"capabilities"`
}

// fetchOllamaTags returns every installed model in /api/tags order, with its
// capabilities where Ollama lists them (0042-MADR D1). Ollama takes no
// credential, so a token is sent only where it names its own Header (R16;
// 0016-MADR D2, A6).
func fetchOllamaTags(ctx context.Context, token llmprovider.Token, cfg config) ([]ollamaTag, error) {
	baseURL := ollamaListingBase
	if cfg.BaseURL != "" {
		baseURL = cfg.BaseURL
	}

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/api/tags", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	cfg.setUserAgent(req)
	if token.Header != "" {
		token.Apply(req, "", "")
	}

	resp, err := cfg.HTTPClient.Do(req)
	if err != nil {
		// An unreachable server is unavailable, as any send failure is
		// (0026-MADR F35).
		return nil, fmt.Errorf("%w: could not reach Ollama at %s: %w", llmprovider.ErrProviderUnavailable, baseURL, err)
	}
	defer cfg.closeBody(resp)

	if resp.StatusCode != http.StatusOK {
		return nil, llmprovider.ClassifyHTTPError(string(llmprovider.ProviderOllama), resp)
	}

	var result struct {
		Models []ollamaTag `json:"models"`
	}
	if err := decodeLimited(resp.Body, &result, "ollama: tags"); err != nil {
		return nil, fmt.Errorf("failed to parse Ollama response: %w", err)
	}
	return result.Models, nil
}

// ValidateOllamaURL checks if an Ollama instance is reachable at the given URL
// by calling GET /api/version, and classifies a failure as the Ollama listing
// does: a server it cannot reach gives an error matching
// llmprovider.ErrProviderUnavailable, wrapping the transport's error, which
// matches ctx's when ctx ends; a status other than 200, an
// *llmprovider.APIError; a URL no request can be made to, a plain error.
// ValidateOllamaURLWith takes options, such as the HTTP client and the
// caller's identity.
func ValidateOllamaURL(ctx context.Context, baseURL string) error {
	return checkOllamaURL(ctx, baseURL, defaultConfig())
}

// ValidateOllamaURLWith is ValidateOllamaURL with llmprovider's common
// options: WithHTTPClient, WithClientInfo (0020-MADR F37, Q5 a and its
// amendment). An option it refuses gives an error matching
// llmprovider.ErrInvalidRequest; its other errors are ValidateOllamaURL's.
func ValidateOllamaURLWith(ctx context.Context, baseURL string, opts ...llmprovider.Option) error {
	cfg, err := configFor(llmprovider.ProviderOllama, opts)
	if err != nil {
		return err
	}
	return checkOllamaURL(ctx, baseURL, cfg)
}

func checkOllamaURL(ctx context.Context, baseURL string, cfg config) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// Trailing slashes go, as ResolveOptions trims a base URL for the
	// listing (0020-MADR F30; 0026-MADR F52).
	baseURL = strings.TrimRight(baseURL, "/")

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/api/version", http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	cfg.setUserAgent(req)

	resp, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: could not reach Ollama at %s: %w", llmprovider.ErrProviderUnavailable, baseURL, err)
	}
	defer config{}.closeBody(resp)

	if resp.StatusCode != http.StatusOK {
		// Classified as the listing's is (0049-MADR D2c).
		return llmprovider.ClassifyHTTPError(string(llmprovider.ProviderOllama), resp)
	}
	return nil
}

// fetchGrokUsable returns the usable Grok models in listing order.
func fetchGrokUsable(ctx context.Context, token llmprovider.Token, cfg config) ([]string, error) {
	baseURL := xaiListingBase
	if cfg.BaseURL != "" {
		baseURL = cfg.BaseURL
	}
	header, value := tokenHeader(token, headerAuthorization, bearerScheme)
	ids, err := fetchDataIDs(ctx, baseURL+"/models", header, value, cfg, llmprovider.ProviderGrok)
	if err != nil {
		return nil, err
	}
	return filterIDs(ids, isUsableGrokModel), nil
}

func curateGrok(usable []string, limit int) []string {
	return curateFromCatalog(llmprovider.ProviderGrok, staticGrok, usable, isUsableGrokModel, rankGrokModel, limit)
}

// fetchDataIDs performs GET endpoint and decodes a {"data":[{"id":…}]} body.
// The credential is sent as header: value when value is non-empty.
func fetchDataIDs(ctx context.Context, endpoint, header, value string, cfg config, provider llmprovider.ProviderID) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, http.NoBody)
	if err != nil {
		return nil, err
	}
	cfg.setUserAgent(req)
	if provider == serviceOpencode {
		req.Header.Set(opencodeSessionHeader, cfg.session) // 0013 B7
	}
	if value != "" {
		req.Header.Set(header, value)
	}

	resp, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer cfg.closeBody(resp)

	if resp.StatusCode != http.StatusOK {
		return nil, llmprovider.ClassifyHTTPError(string(provider), resp)
	}

	var result struct {
		Data []dataModel `json:"data"`
	}
	if err := decodeLimited(resp.Body, &result, string(provider)+": models"); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(result.Data))
	for _, m := range result.Data {
		ids = append(ids, m.ID)
		if info, ok := dataModelInfo(provider, m); ok {
			cfg.emit(info)
		}
	}
	return ids, nil
}

// filterIDs keeps the ids usable admits, preserving order.
func filterIDs(ids []string, usable func(string) bool) []string {
	var out []string
	for _, id := range ids {
		if usable(id) {
			out = append(out, id)
		}
	}
	return out
}

// fetchOpencodeUsable returns the usable gateway models in listing order.
// The OpenCode /models endpoint is PUBLIC — it answers 200 with no
// credentials (verified 2026-08-28) — so the credential is sent only when
// there is one, and an empty key is not an error. The listing carries no
// routing or capability metadata (every entry reports owned_by "opencode"),
// so route selection cannot be derived from it; see providers/opencode.
func fetchOpencodeUsable(ctx context.Context, gateway llmprovider.ProviderID, token llmprovider.Token, cfg config) ([]string, error) {
	baseURL, err := opencodeBaseURL(gateway)
	if err != nil {
		return nil, err
	}
	if cfg.BaseURL != "" {
		baseURL = strings.TrimRight(cfg.BaseURL, "/")
	}
	header, value := headerAuthorization, ""
	if token.Value != "" {
		header, value = tokenHeader(token, headerAuthorization, bearerScheme)
	}
	ids, err := fetchDataIDs(ctx, baseURL+"/models", header, value, cfg, "opencode")
	if err != nil {
		return nil, err
	}
	return filterIDs(ids, isUsableOpencodeModel), nil
}

// onlyText reports whether a modality list is exactly ["text"].
func onlyText(mods []string) bool { return len(mods) == 1 && mods[0] == jsonKeyText }

// hasText reports whether a modality list includes "text". A model that also
// accepts images or files still serves a text prompt (MADR 0007 §1b).
func hasText(mods []string) bool { return slices.Contains(mods, jsonKeyText) }

// fetchHuggingFaceUsable returns the usable router models, fastest first: input
// must include text and output must be exactly text (MADR 0007 §1b), and at
// least one provider offering must be live. The endpoint is PUBLIC (200 with
// no credential, verified 2026-08-29), so the credential is optional.
//
// Unlike the other providers, ranking here uses measured figures rather than
// name heuristics: the listing reports throughput (tokens/sec) and
// first_token_latency_ms per provider offering. The sorted order is handed to
// curateFromCatalog with a nil rankFn, which preserves it.
func fetchHuggingFaceUsable(ctx context.Context, token llmprovider.Token, cfg config) ([]string, error) {
	baseURL := huggingFaceBaseURL
	if cfg.BaseURL != "" {
		baseURL = strings.TrimRight(cfg.BaseURL, "/")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/models", http.NoBody)
	if err != nil {
		return nil, err
	}
	cfg.setUserAgent(req)
	if token.Value != "" {
		token.Apply(req, headerAuthorization, bearerScheme)
	}

	resp, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer cfg.closeBody(resp)

	if resp.StatusCode != http.StatusOK {
		return nil, llmprovider.ClassifyHTTPError(string(llmprovider.ProviderHuggingFace), resp)
	}

	var result struct {
		Data []struct {
			ID           string `json:"id"`
			Created      int64  `json:"created"`
			Architecture struct {
				InputModalities  []string `json:"input_modalities"`
				OutputModalities []string `json:"output_modalities"`
			} `json:"architecture"`
			Providers []struct {
				Status              string  `json:"status"`
				SupportsTools       bool    `json:"supports_tools"`
				Throughput          float64 `json:"throughput"`
				FirstTokenLatencyMs float64 `json:"first_token_latency_ms"`
			} `json:"providers"`
		} `json:"data"`
	}
	if err := decodeLimited(resp.Body, &result, "huggingface: models"); err != nil {
		return nil, err
	}

	type scored struct {
		id   string
		tps  float64
		ttft float64
	}
	var ranked []scored
	for _, m := range result.Data {
		// Its created is a release date, measured (0041-PLAN Phase 2).
		b := newInfo(llmprovider.ProviderHuggingFace, m.ID)
		b.released(listingDay(m.Created))
		cfg.emit(b.done())
		if !hasText(m.Architecture.InputModalities) || !onlyText(m.Architecture.OutputModalities) {
			continue
		}
		if !isUsableHuggingFaceModel(m.ID) {
			continue
		}
		best := scored{id: m.ID, ttft: math.MaxFloat64}
		live := false
		for _, pr := range m.Providers {
			if pr.Status != "live" {
				continue
			}
			live = true
			if pr.Throughput > best.tps {
				best.tps = pr.Throughput
			}
			if pr.FirstTokenLatencyMs > 0 && pr.FirstTokenLatencyMs < best.ttft {
				best.ttft = pr.FirstTokenLatencyMs
			}
		}
		if live {
			ranked = append(ranked, best)
		}
	}
	// Fastest first; ties broken by lowest time-to-first-token.
	slices.SortStableFunc(ranked, func(a, b scored) int {
		switch {
		case a.tps > b.tps:
			return -1
		case a.tps < b.tps:
			return 1
		case a.ttft < b.ttft:
			return -1
		case a.ttft > b.ttft:
			return 1
		}
		return 0
	})
	available := make([]string, 0, len(ranked))
	for _, r := range ranked {
		available = append(available, r.id)
	}
	return available, nil
}

// fetchTogetherUsable returns Together's chat models in listing order.
// GET {base}/models answers with a bare JSON array, not an OpenAI
// {"data": [...]} envelope, and lists every model type; only "chat" models
// can serve a Chat Completions request (0017-REPORT, Together section).
func fetchTogetherUsable(ctx context.Context, token llmprovider.Token, cfg config) ([]string, error) {
	baseURL := togetherBaseURL
	if cfg.BaseURL != "" {
		baseURL = strings.TrimRight(cfg.BaseURL, "/")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/models", http.NoBody)
	if err != nil {
		return nil, err
	}
	cfg.setUserAgent(req)
	token.Apply(req, headerAuthorization, bearerScheme)

	resp, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer cfg.closeBody(resp)
	if resp.StatusCode != http.StatusOK {
		return nil, llmprovider.ClassifyHTTPError(string(llmprovider.ProviderTogether), resp)
	}

	var models []togetherModel
	if err := decodeLimited(resp.Body, &models, "together: models"); err != nil {
		return nil, err
	}
	usable := make([]string, 0, len(models))
	for _, m := range models {
		if m.ID != "" {
			cfg.emit(togetherInfo(m))
		}
		if m.Type == "chat" && m.ID != "" {
			usable = append(usable, m.ID)
		}
	}
	return usable, nil
}

// curateTogether is the fallback curation when the metadata document is
// unavailable: the static catalog's order, then the listing's.
func curateTogether(usable []string, limit int) []string {
	return curateFromCatalog(llmprovider.ProviderTogether, staticTogether, usable, nil, nil, limit)
}

// curateHuggingFace passes a nil rankFn, which preserves the metadata order.
func curateHuggingFace(usable []string, limit int) []string {
	return curateFromCatalog(llmprovider.ProviderHuggingFace, staticHuggingFace, usable, isUsableHuggingFaceModel, nil, limit)
}

// kiloCatalogEntry is the subset of Kilo's OpenRouter-shaped catalog entry this
// package reads. Shared by listKiloModels, KiloModelCapabilities and the
// ranker (MADR 0009 §2).
type kiloCatalogEntry struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Created      int64  `json:"created"`
	Architecture struct {
		InputModalities  []string `json:"input_modalities"`
		OutputModalities []string `json:"output_modalities"`
	} `json:"architecture"`
	Pricing struct {
		Prompt            string `json:"prompt"`
		Completion        string `json:"completion"`
		InputCacheRead    string `json:"input_cache_read"`
		InputCacheWrite   string `json:"input_cache_write"`
		InternalReasoning string `json:"internal_reasoning"`
	} `json:"pricing"`
	TopProvider struct {
		MaxCompletionTokens *int `json:"max_completion_tokens"`
	} `json:"top_provider"`
	ContextLength         int                `json:"context_length"`
	ExpirationDate        string             `json:"expiration_date"`
	PreferredIndex        *int               `json:"preferredIndex"`
	TerminalBench         *kiloTerminalBench `json:"terminalBench"`
	SupportedParameters   []string           `json:"supported_parameters"`
	MayTrainOnYourPrompts bool               `json:"mayTrainOnYourPrompts"`
}

// kiloTerminalBench is the benchmark block Kilo publishes for some models.
type kiloTerminalBench struct {
	OverallScore *float64 `json:"overallScore"`
}

// kiloPriceRank parses Kilo's string pricing into a sortable value. A price
// kiloPrice does not know, negative, non-finite or unparseable, means
// "variable" (the kilo-auto tiers report "-1") and sorts last rather than
// first (0021-MADR C9).
func kiloPriceRank(s string) float64 {
	if v, ok := kiloPrice(s); ok {
		return v
	}
	return math.MaxFloat64
}

// fetchKiloCatalog performs the shared GET {base}/models, or an organization's
// listing (resolveKiloEndpoints). The public endpoint answers with no
// credential (verified 2026-08-29), so the token may be empty.
func fetchKiloCatalog(ctx context.Context, token llmprovider.Token, cfg config) ([]kiloCatalogEntry, error) {
	endpoints := kiloendpoint.Resolve(cfg.BaseURL, token.Value, cfg.KiloOrganization)
	req, err := http.NewRequestWithContext(ctx, "GET", endpoints.Models, http.NoBody)
	if err != nil {
		return nil, err
	}
	cfg.setUserAgent(req)
	if token.Value != "" {
		token.Apply(req, headerAuthorization, bearerScheme)
	}
	if endpoints.Org != "" {
		req.Header.Set(kiloendpoint.OrganizationHeader, endpoints.Org)
	}
	resp, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer cfg.closeBody(resp)
	if resp.StatusCode != http.StatusOK {
		return nil, llmprovider.ClassifyHTTPError(string(llmprovider.ProviderKilo), resp)
	}
	var result struct {
		Data []kiloCatalogEntry `json:"data"`
	}
	if err := decodeLimited(resp.Body, &result, "kilo: models"); err != nil {
		return nil, err
	}
	for _, e := range result.Data {
		cfg.emit(kiloInfo(e))
	}
	return result.Data, nil
}

// The Kilo catalog's curation. Two documented traps are handled in kiloUsable:
//
//  1. pricing.completion is a STRING and is "-1" for the variable-priced
//     kilo-auto/{frontier,balanced,efficient} tiers. A naive ascending sort would
//     rank the most expensive tiers as cheaper than free, so kiloPriceRank sorts
//     negative and unparseable prices LAST.
//  2. Kilo model ids may legitimately END in ":free" (tencent/hy3:free), so
//     nothing is stripped at the colon. splitHuggingFaceModelPolicy must not be
//     used here.
//
// Models flagged mayTrainOnYourPrompts are excluded. That is a POLICY decision,
// not a capability filter — see isUsableKiloModel's comment.

// kiloUsable returns the usable Kilo models, cheapest first: input must include
// text and output must be exactly text (MADR 0007 §1b), tools must be supported,
// and the training policy applies.
func kiloUsable(entries []kiloCatalogEntry) []string {
	type priced struct {
		id    string
		price float64
	}
	var ranked []priced
	for _, m := range entries {
		if !hasText(m.Architecture.InputModalities) || !onlyText(m.Architecture.OutputModalities) {
			continue
		}
		if m.MayTrainOnYourPrompts { // POLICY — see isUsableKiloModel
			continue
		}
		if !slices.Contains(m.SupportedParameters, jsonKeyTools) {
			continue
		}
		if !isUsableKiloModel(m.ID) {
			continue
		}
		ranked = append(ranked, priced{id: m.ID, price: kiloPriceRank(m.Pricing.Completion)})
	}
	// Cheapest first; "-1" and unparseable prices sort last via kiloPriceRank.
	slices.SortStableFunc(ranked, func(a, b priced) int {
		switch {
		case a.price < b.price:
			return -1
		case a.price > b.price:
			return 1
		}
		return 0
	})
	available := make([]string, 0, len(ranked))
	for _, r := range ranked {
		available = append(available, r.id)
	}
	return available
}

// curateKilo passes a nil rankFn, which preserves the price ordering.
func curateKilo(usable []string, limit int) []string {
	return curateFromCatalog(llmprovider.ProviderKilo, staticKilo, usable, isUsableKiloModel, nil, limit)
}

// kiloCurate ranks Kilo's usable models from the listing's own metadata
// (MADR 0009 §2). When fewer than limit are eligible, the rest come from
// curateKilo's order, then the usable list.
func kiloCurate(entries []kiloCatalogEntry, profile Profile, w workload) curator {
	return func(usable []string, limit int) []string {
		byID := make(map[string]kiloCatalogEntry, len(entries))
		for _, e := range entries {
			byID[e.ID] = e
		}
		now := rankingNow()
		cands := make([]rankCandidate, 0, len(usable))
		for _, id := range usable {
			e, ok := byID[id]
			if !ok {
				e.ID = id
			}
			cands = append(cands, kiloCandidate(e, now).withTurn(w))
		}
		return rankRecommended(profile, llmprovider.ProviderKilo, cands, append(curateKilo(usable, limit), usable...), limit)
	}
}

// KiloModelCapabilities returns the supported_parameters published for one Kilo
// model, for use with kilo.WithCapabilities. The catalog endpoint is public, so
// apiKey may be empty. An empty list is a valid answer; an error means the
// catalog was unreachable or the model is absent from it.
func KiloModelCapabilities(ctx context.Context, apiKey, model string, opts ...llmprovider.Option) ([]string, error) {
	cfg, err := configFor(llmprovider.ProviderKilo, opts)
	if err != nil {
		return nil, err
	}
	entries, err := fetchKiloCatalog(ctx, llmprovider.Token{Value: apiKey}, cfg)
	if err != nil {
		return nil, err
	}
	for _, m := range entries {
		if m.ID == model {
			return m.SupportedParameters, nil
		}
	}
	return nil, fmt.Errorf("%w: kilo model %q not in catalog", llmprovider.ErrInvalidRequest, model)
}
