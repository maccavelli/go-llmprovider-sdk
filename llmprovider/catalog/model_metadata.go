package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Model metadata for the open catalogs (MADR 0009 §2): a models.dev-format
// document, by default OpenCode's, fetched concurrently with a listing and
// cached in-process.
const (
	defaultModelMetadataURL = "https://models.opencode.ai/api.json"
	envModelMetadataURL     = "LLMPROVIDER_MODELS_METADATA_URL"
	envDisableModelMetadata = "LLMPROVIDER_DISABLE_MODELS_METADATA"
	modelMetadataTTL        = 10 * time.Minute
	// modelMetadataRetryAfter is how long a failed fetch is remembered before
	// the next load tries again (MADR 0013 A6).
	modelMetadataRetryAfter = time.Minute
	// metadataLookupTimeout bounds the lookup a generation request makes
	// (OpenCode's chat route) whatever the caller's context (MADR 0013 A6).
	metadataLookupTimeout = 5 * time.Second

	metadataKeyZen = "opencode"
	metadataKeyGo  = "opencode-go"
	metadataKeyHF  = "huggingface"
	// metadataKeyTogether is models.dev's key for Together AI (measured
	// 2026-09-30; 0017-REPORT).
	metadataKeyTogether = "togetherai"
)

// errModelMetadataDisabled reports that the environment turned the fetch off.
var errModelMetadataDisabled = errors.New("model metadata: disabled by WithoutModelMetadata")

// modelCost is a models.dev price, USD per million tokens.
type modelCost struct {
	Input  float64 `json:"input"`
	Output float64 `json:"output"`
}

// modelMetadata is the subset of one models.dev model entry the ranker reads.
type modelMetadata struct {
	Name      string     `json:"name"`
	Family    string     `json:"family"`
	Reasoning *bool      `json:"reasoning"`
	Cost      *modelCost `json:"cost"`
	Limit     struct {
		Context int `json:"context"`
	} `json:"limit"`
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

// modelReasoningOption is one models.dev reasoning_options entry.
type modelReasoningOption struct {
	Type   string   `json:"type"`
	Values []string `json:"values"`
}

// modelMetadataSection is one provider's entry in the document.
type modelMetadataSection struct {
	Models map[string]modelMetadata `json:"models"`
}

// modelMetadataDoc maps a document key to its models by id.
type modelMetadataDoc map[string]map[string]modelMetadata

// reasoningEfforts returns the effort values the document lists for one
// model's reasoning_options, or nil.
func (d modelMetadataDoc) reasoningEfforts(provider llmprovider.ProviderID, model string) []string {
	for _, o := range d[modelMetadataKey(provider)][model].ReasoningOptions {
		if o.Type == jsonKeyEffort {
			// The cache is the process's; the caller gets its own copy
			// (R29; 0026-MADR F53).
			return slices.Clone(o.Values)
		}
	}
	return nil
}

// interleavedField returns the message field a model expects its prior
// reasoning replayed under ("reasoning_content"), or "" when it declares none
// (MADR 0012 §2, O5).
func (d modelMetadataDoc) interleavedField(provider llmprovider.ProviderID, model string) string {
	var declared struct {
		Field string `json:"field"`
	}
	if json.Unmarshal(d[modelMetadataKey(provider)][model].Interleaved, &declared) != nil {
		return ""
	}
	return declared.Field
}

// npm returns a model's provider.npm, the AI SDK package OpenCode's client
// routes it by, and false when the document does not list the model (MADR
// 0012 §3.1).
func (d modelMetadataDoc) npm(provider llmprovider.ProviderID, model string) (string, bool) {
	m, ok := d[modelMetadataKey(provider)][model]
	if !ok {
		return "", false
	}
	return m.Provider.NPM, true
}

// Metadata is the model metadata document as a provider reads it for one
// request: reasoning efforts, the interleaved-reasoning field and the AI SDK
// package, per model (MADR 0009 §2).
//
// Moved here from llmprovider (0015-PLAN S8, commit 2).
type Metadata struct{ doc modelMetadataDoc }

// LookupMetadata returns the document at url, or the default document when
// url is empty, fetched through client and cached as the listing caches it.
// A nil client means the default client, as an omitted WithHTTPClient does.
// The lookup waits at most 5 s, whatever ctx allows, and a failed fetch is not
// retried for a minute (MADR 0013 A6). Lookups share one fetch per URL, which
// carries on, for up to 10 s, after a lookup gives up; past the cache's TTL a
// lookup returns the cached document and the fetch refreshes it (0021-MADR
// C1, C2).
//
// Moved here from llmprovider (0015-PLAN S8, commit 2).
func LookupMetadata(ctx context.Context, url string, client *http.Client) (Metadata, error) {
	ctx, cancel := context.WithTimeout(ctx, metadataLookupTimeout)
	defer cancel()
	cfg := defaultConfig()
	cfg.ModelMetadataURL = url
	if client != nil {
		cfg.HTTPClient = client
	}
	doc, err := loadModelMetadata(ctx, cfg)
	return Metadata{doc: doc}, err
}

// LookupMetadataWith is LookupMetadata for the provider id, with its options:
// the document is WithModelMetadataURL's, fetched through WithHTTPClient's
// client, and the request names the caller's WithClientInfo application and
// WithSessionID session, as the provider's own requests do (0026-MADR F33).
// An option LookupMetadata has no use for is ignored; one scoped to another
// provider is refused, as List refuses it.
func LookupMetadataWith(ctx context.Context, id llmprovider.ProviderID, opts ...llmprovider.Option) (Metadata, error) {
	cfg, err := configFor(id, opts)
	if err != nil {
		return Metadata{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, metadataLookupTimeout)
	defer cancel()
	doc, err := loadModelMetadata(ctx, cfg)
	return Metadata{doc: doc}, err
}

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
	}
	return ""
}

// modelMetadataCacheEntry is one URL's last good document and last failure.
type modelMetadataCacheEntry struct {
	doc     modelMetadataDoc // nil until a fetch succeeds; kept when a refresh fails
	fetched time.Time        // when doc was fetched
	failed  time.Time        // when the last fetch failed; zero after a success
	err     error            // that failure
}

// metadataFetch is one URL's fetch in flight, which every lookup of that URL
// waits for (0021-MADR C2). doc and err are set before done is closed.
type metadataFetch struct {
	done chan struct{}
	doc  modelMetadataDoc
	err  error
}

var (
	modelMetadataMu    sync.Mutex
	modelMetadataCache = map[string]modelMetadataCacheEntry{}
	// modelMetadataFetches holds each URL's fetch in flight, under
	// modelMetadataMu.
	modelMetadataFetches = map[string]*metadataFetch{}
	// metadataFetching counts the fetches in flight, so that tests can wait
	// for a background refresh.
	metadataFetching sync.WaitGroup
	// metadataNow is the cache's clock. Tests move it.
	metadataNow = time.Now
)

// metadataLimit bounds the metadata document (0021-MADR C3).
const metadataLimit = 32 << 20

// metadataFetchTimeout bounds one fetch. The fetch is detached from the
// lookups that wait for it, so it has its own bound (0021-MADR C1).
const metadataFetchTimeout = 10 * time.Second

// modelMetadataURL resolves the document URL: the option, else OpenCode's.
func modelMetadataURL(cfg config) string {
	if cfg.ModelMetadataURL != "" {
		return cfg.ModelMetadataURL
	}
	return defaultModelMetadataURL
}

// OptionsFromEnv returns the options LLMPROVIDER_MODELS_METADATA_URL and
// LLMPROVIDER_DISABLE_MODELS_METADATA set: WithModelMetadataURL for a URL, and
// WithoutModelMetadata for a true boolean ("1", "true", …). It is the only way
// either variable takes effect: the library reads no environment variable
// unless a caller asks (0015-MADR D9). An unset variable adds no option.
func OptionsFromEnv() []llmprovider.Option {
	var opts []llmprovider.Option
	if v := os.Getenv(envModelMetadataURL); v != "" {
		opts = append(opts, llmprovider.WithModelMetadataURL(v))
	}
	if v, err := strconv.ParseBool(os.Getenv(envDisableModelMetadata)); err == nil && v {
		opts = append(opts, llmprovider.WithoutModelMetadata())
	}
	return opts
}

// loadModelMetadata returns the document, from the in-process cache while it
// is younger than modelMetadataTTL. A failure is remembered for
// modelMetadataRetryAfter, and while it is, loads answer from the cache
// without fetching: the stale document when there is one, else the failure
// (MADR 0013 A6).
//
// One fetch per URL runs at a time, detached from every caller's context
// and bounded by metadataFetchTimeout; each load waits for it under its own
// context. Past the TTL the stale document is returned at once and the
// fetch refreshes it in the background. Every failure of the fetch is
// remembered; a caller's own context ending never is (0021-MADR C1, C2).
func loadModelMetadata(ctx context.Context, cfg config) (modelMetadataDoc, error) {
	if cfg.metadataOff {
		return nil, errModelMetadataDisabled
	}
	url := modelMetadataURL(cfg)
	modelMetadataMu.Lock()
	e := modelMetadataCache[url]
	now := metadataNow()
	switch {
	case e.doc != nil && now.Sub(e.fetched) < modelMetadataTTL:
		modelMetadataMu.Unlock()
		return e.doc, nil
	case !e.failed.IsZero() && now.Sub(e.failed) < modelMetadataRetryAfter:
		modelMetadataMu.Unlock()
		return e.cached()
	}
	fetch := startMetadataFetch(ctx, url, cfg)
	modelMetadataMu.Unlock()
	if e.doc != nil {
		return e.doc, nil
	}
	select {
	case <-fetch.done:
		return fetch.doc, fetch.err
	case <-ctx.Done():
		return nil, fmt.Errorf("model metadata: %w", ctx.Err())
	}
}

// startMetadataFetch returns url's fetch in flight, starting one when there
// is none. modelMetadataMu is held.
func startMetadataFetch(ctx context.Context, url string, cfg config) *metadataFetch {
	if fetch := modelMetadataFetches[url]; fetch != nil {
		return fetch
	}
	fetch := &metadataFetch{done: make(chan struct{})}
	modelMetadataFetches[url] = fetch
	fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), metadataFetchTimeout)
	metadataFetching.Add(1)
	go func() {
		defer metadataFetching.Done()
		defer cancel()
		doc, err := fetchModelMetadata(fetchCtx, url, cfg)
		modelMetadataMu.Lock()
		e := modelMetadataCache[url]
		if err != nil {
			e.failed, e.err = metadataNow(), err
		} else {
			e = modelMetadataCacheEntry{doc: doc, fetched: metadataNow()}
		}
		modelMetadataCache[url] = e
		delete(modelMetadataFetches, url)
		fetch.doc, fetch.err = e.cached()
		modelMetadataMu.Unlock()
		close(fetch.done)
	}()
	return fetch
}

// cached answers from a cache entry after a failure: the stale document when
// there is one, else the failure.
func (e modelMetadataCacheEntry) cached() (modelMetadataDoc, error) {
	if e.doc != nil {
		return e.doc, nil
	}
	return nil, e.err
}

// fetchModelMetadata performs one GET. Go's transport requests gzip itself.
func fetchModelMetadata(ctx context.Context, url string, cfg config) (modelMetadataDoc, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("model metadata: %w", err)
	}
	cfg.setUserAgent(req) // the caller's identity (0020-MADR F37)
	resp, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("model metadata: %w", err)
	}
	defer cfg.closeBody(resp)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("model metadata: %s returned HTTP %d", url, resp.StatusCode)
	}
	return decodeModelMetadata(resp.Body)
}

// decodeModelMetadata keeps the sections the ranking reads: MADR 0009's
// three and Together's (0017-MADR D1; 0020-MADR F4). A section without
// models is treated as absent.
func decodeModelMetadata(r io.Reader) (modelMetadataDoc, error) {
	var raw struct {
		Zen      *modelMetadataSection `json:"opencode"`
		Go       *modelMetadataSection `json:"opencode-go"`
		HF       *modelMetadataSection `json:"huggingface"`
		Together *modelMetadataSection `json:"togetherai"`
	}
	if err := decodeLimited(r, metadataLimit, &raw, "model metadata"); err != nil {
		return nil, err
	}
	doc := modelMetadataDoc{}
	for key, s := range map[string]*modelMetadataSection{
		metadataKeyZen: raw.Zen, metadataKeyGo: raw.Go, metadataKeyHF: raw.HF, metadataKeyTogether: raw.Together,
	} {
		if s != nil && s.Models != nil {
			doc[key] = s.Models
		}
	}
	return doc, nil
}

// modelMetadataResult carries one fetch's outcome across a goroutine.
type modelMetadataResult struct {
	doc modelMetadataDoc
	err error
}

// startModelMetadata fetches the document concurrently with a listing, under
// the listing's context (MADR 0009 §2). The channel receives exactly one value.
func startModelMetadata(ctx context.Context, cfg config) <-chan modelMetadataResult {
	ch := make(chan modelMetadataResult, 1)
	go func() {
		doc, err := loadModelMetadata(ctx, cfg)
		ch <- modelMetadataResult{doc: doc, err: err}
	}()
	return ch
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
	return c
}

// metadataCurate ranks a provider's usable models with the metadata document
// (MADR 0009 §2). A failed or disabled fetch, or a document without the
// provider's key, returns fallback's curation unchanged.
func metadataCurate(provider llmprovider.ProviderID, profile Profile, meta <-chan modelMetadataResult, fallback func([]string) []string) func([]string) []string {
	return func(usable []string) []string {
		res := <-meta
		models, ok := res.doc[modelMetadataKey(provider)]
		if res.err != nil || !ok {
			return fallback(usable)
		}
		now := rankingNow()
		cands := make([]rankCandidate, 0, len(usable))
		for _, id := range usable {
			m, covered := models[id]
			cands = append(cands, metadataCandidate(id, m, covered, now))
		}
		return rankRecommended(profile, provider, cands, append(fallback(usable), usable...))
	}
}
