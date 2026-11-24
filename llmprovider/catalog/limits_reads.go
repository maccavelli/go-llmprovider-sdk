package catalog

import (
	"bytes"
	"cmp"
	"context"
	_ "embed" // limits_snapshot.json
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/chatgpt"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// The limits a listing states beyond ModelInfo's fields, and the reads
// LookupLimits adds, each kept in its own map of the cache (0089-MADR D2,
// D3, DL66): a listing refresh replaces the listing's extra fields with the
// new listing's, and keeps the reads.

// listingExtra is what a listing states of one model's limits beyond
// ModelInfo's fields.
type listingExtra struct {
	// output is Together's config.max_output_length, or Kilo's
	// per_request_limits.completion_tokens; 0 when not stated.
	output int
	// windows is each live Hugging Face provider's context_length, by
	// provider.
	windows map[string]int
}

// window is the window the extra states for model: its suffixed provider's
// when model names one, else the smallest any live provider states
// (0089-MADR D3, DL10).
func (x listingExtra) window(model string) (int, bool) {
	if len(x.windows) == 0 {
		return 0, false
	}
	if _, suffix, ok := strings.Cut(model, ":"); ok {
		if w, ok := x.windows[suffix]; ok {
			return w, true
		}
	}
	smallest := 0
	for _, w := range x.windows {
		if smallest == 0 || w < smallest {
			smallest = w
		}
	}
	return smallest, true
}

// extrasEntry is one listing's extra fields, and when they were read.
type extrasEntry struct {
	models map[string]listingExtra
	at     time.Time
}

// readKind is one of the reads LookupLimits adds.
type readKind int

const (
	// readOneModel is Claude's GET /v1/models/{model}.
	readOneModel readKind = iota
	// readServed is Ollama's /api/ps.
	readServed
	// readBackend is the ChatGPT backend's listing.
	readBackend
)

// limitReadKey is one read: a listing's key, the model as asked, and the
// kind.
type limitReadKey struct {
	listingKey
	model string
	kind  readKind
}

// limitRead is what one read states, 0 for a limit it does not; none for a
// model the source does not know, kept as a read for its freshness.
type limitRead struct {
	output, input, window int
	// resolved is the id Claude's one-model read resolved model to, for a
	// later refusal's model match (0089-MADR V6).
	resolved string
	none     bool
	at       time.Time
}

// storeExtras replaces key's extra fields with a listing's.
func (c *store) storeExtras(key listingKey, models map[string]listingExtra) {
	c.listingMu.Lock()
	defer c.listingMu.Unlock()
	c.listingExtras[key] = extrasEntry{models: models, at: listingNow()}
}

// extra is model's extra fields under key, when its listing stated any.
func (c *store) extra(key listingKey, model string) (listingExtra, time.Time, bool) {
	c.listingMu.Lock()
	defer c.listingMu.Unlock()
	e := c.listingExtras[key]
	x, ok := e.models[model]
	return x, e.at, ok
}

// storeRead keeps a read made now.
func (c *store) storeRead(k limitReadKey, r limitRead) {
	c.listingMu.Lock()
	defer c.listingMu.Unlock()
	r.at = listingNow()
	c.limitReads[k] = r
}

// read is k's read, of any age.
func (c *store) read(k limitReadKey) (limitRead, bool) {
	c.listingMu.Lock()
	defer c.listingMu.Unlock()
	r, ok := c.limitReads[k]
	return r, ok
}

// freshRead reports whether k's read is younger than limitReadTTL.
func (c *store) freshRead(k limitReadKey) bool {
	r, ok := c.read(k)
	return ok && listingNow().Sub(r.at) < limitReadTTL
}

// listingAt is key's listing's models, of any age, and when they were read.
func (c *store) listingAt(key listingKey) (map[string]ModelInfo, time.Time) {
	c.listingMu.Lock()
	defer c.listingMu.Unlock()
	e := c.listing[key]
	return e.models, e.fetched
}

// listingDialect is the dialect key's chatcompat listing was recognised as,
// or "".
func (c *store) listingDialect(key listingKey) string {
	c.listingMu.Lock()
	defer c.listingMu.Unlock()
	if e := c.listing[key]; e.listing != nil {
		return e.listing.dialect
	}
	return ""
}

// metadataFetchedAt is when cfg's cached document was fetched, or last
// revalidated.
func (c *store) metadataFetchedAt(cfg config) time.Time {
	c.metadataMu.Lock()
	defer c.metadataMu.Unlock()
	return c.metadata[modelMetadataURL(cfg)].fetched
}

// newClaudeRequest is a GET of Anthropic's Models API at endpoint, with the
// key, the version and the workspace a listing sends (0064-MADR D2).
func newClaudeRequest(ctx context.Context, endpoint string, token llmprovider.Token, cfg config) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, err
	}
	cfg.setUserAgent(req)
	token.Apply(req, "x-api-key", "")
	req.Header.Set("anthropic-version", "2023-06-01")
	if cfg.ClaudeWorkspace != "" {
		req.Header.Set(headerClaudeWorkspace, cfg.ClaudeWorkspace)
	}
	return req, nil
}

// readClaudeModel reads one model by GET /v1/models/{model}, which takes an
// id or an alias, and keeps what it states, and the id it resolved to; a 404
// is kept as none (0089-MADR D2, DL2, DL66).
func readClaudeModel(ctx context.Context, token llmprovider.Token, cfg config, key listingKey, model string) error {
	k := limitReadKey{key, model, readOneModel}
	req, err := newClaudeRequest(ctx, claudeListingBase(cfg.BaseURL)+"/v1/models/"+url.PathEscape(model), token, cfg)
	if err != nil {
		return fmt.Errorf("model limits: claude model: %w", err)
	}
	resp, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("model limits: claude model: %w", transport.WrapOffline(ctx, err, llmprovider.ErrOffline))
	}
	defer cfg.closeBody(resp)
	if resp.StatusCode == http.StatusNotFound {
		cfg.store.storeRead(k, limitRead{none: true})
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("model limits: claude model: %w", llmprovider.ClassifyHTTPError(string(llmprovider.ProviderClaude), resp))
	}
	var m claudeModel
	if err := decodeLimited(resp.Body, &m, "claude: model"); err != nil {
		return fmt.Errorf("model limits: %w", err)
	}
	r := limitRead{resolved: m.ID}
	if m.MaxTokens != nil {
		r.output = *m.MaxTokens
	}
	if m.MaxInputTokens != nil {
		r.input = *m.MaxInputTokens
	}
	cfg.store.storeRead(k, r)
	// The resolved id reaches the provider's refusal check through the
	// cache's memo (0089-MADR V6, through internal/cacheref).
	cfg.memo.KeepResolved(memoKey(llmprovider.ProviderClaude, model, cfg, false), m.ID)
	return nil
}

// ollamaBase is a built-in Ollama's base: base, or the local default, as
// /api/show reads it.
func ollamaBase(base string) string {
	if base == "" {
		return ollamaListingBase
	}
	return strings.TrimRight(base, "/")
}

// ollamaRoot is a chatcompat Ollama server's root: its base URL less a
// trailing /v1, as its dialect's details are read (0061-MADR X4).
func ollamaRoot(base string) string {
	return strings.TrimSuffix(strings.TrimRight(base, "/"), "/v1")
}

// readServedContext reads Ollama's GET {base}/api/ps, the context each
// loaded model is served with, and keeps it for each loaded model, and none
// for model when it is not loaded (0089-MADR D2, DL12, DL13). A server
// without /api/ps (404) is read as none loaded.
func readServedContext(ctx context.Context, id llmprovider.ProviderID, token llmprovider.Token, cfg config, base string,
	key listingKey, model string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/ps", http.NoBody)
	if err != nil {
		return fmt.Errorf("model limits: ollama ps: %w", err)
	}
	cfg.setUserAgent(req)
	if token.Header != "" {
		token.Apply(req, "", "")
	}
	resp, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("model limits: ollama ps: %w", transport.WrapOffline(ctx, err, llmprovider.ErrOffline))
	}
	defer cfg.closeBody(resp)
	var ps struct {
		Models []struct {
			Name          string `json:"name"`
			Model         string `json:"model"`
			ContextLength int    `json:"context_length"`
		} `json:"models"`
	}
	switch {
	case resp.StatusCode == http.StatusNotFound:
	case resp.StatusCode != http.StatusOK:
		return fmt.Errorf("model limits: ollama ps: %w", llmprovider.ClassifyHTTPError(string(id), resp))
	default:
		if err := decodeLimited(resp.Body, &ps, "ollama: ps"); err != nil {
			return fmt.Errorf("model limits: %w", err)
		}
	}
	found := false
	for _, m := range ps.Models {
		if m.ContextLength <= 0 {
			continue
		}
		for _, name := range slices.Compact([]string{m.Name, m.Model, strings.TrimSuffix(m.Name, ":latest")}) {
			if name != "" {
				cfg.store.storeRead(limitReadKey{key, name, readServed}, limitRead{window: m.ContextLength})
				found = found || name == model
			}
		}
	}
	if !found {
		cfg.store.storeRead(limitReadKey{key, model, readServed}, limitRead{none: true})
	}
	return nil
}

// readBackendWindows reads the ChatGPT backend's listing, or a plan
// session's, the one the openai provider sends (internal/chatgpt), and keeps
// each model's context_window, and none for model when it does not list it
// (0089-MADR D3, DL8).
func readBackendWindows(ctx context.Context, src llmprovider.TokenSource, token llmprovider.Token, cfg config, key listingKey,
	model string) error {
	var req *http.Request
	var err error
	if chatgpt.IsSession(src) {
		req, err = chatgpt.ModelsRequestAs(ctx, cmp.Or(cfg.BaseURL, chatgpt.BaseURL), cfg.userAgent, cfg.clientName, token, src)
	} else {
		req, err = chatgpt.PlanModelsRequest(ctx, cmp.Or(cfg.BaseURL, openAIListingBase), cfg.userAgent, token)
	}
	if err != nil {
		return fmt.Errorf("model limits: %w", err)
	}
	resp, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("model limits: chatgpt models: %w", transport.WrapOffline(ctx, err, llmprovider.ErrOffline))
	}
	defer cfg.closeBody(resp)
	if err := llmprovider.ClassifyHTTPError(string(llmprovider.ProviderOpenAI), resp); err != nil {
		return fmt.Errorf("model limits: chatgpt models: %w", err)
	}
	var page struct {
		Models []struct {
			Slug          string `json:"slug"`
			ContextWindow int    `json:"context_window"`
		} `json:"models"`
	}
	if err := decodeLimited(resp.Body, &page, "chatgpt: models"); err != nil {
		return fmt.Errorf("model limits: %w", err)
	}
	found := false
	for _, m := range page.Models {
		if m.Slug == "" || m.ContextWindow <= 0 {
			continue
		}
		cfg.store.storeRead(limitReadKey{key, m.Slug, readBackend}, limitRead{window: m.ContextWindow})
		found = found || m.Slug == model
	}
	if !found {
		cfg.store.storeRead(limitReadKey{key, model, readBackend}, limitRead{none: true})
	}
	return nil
}

// limitsSnapshot is the embedded extract: each model's limit, by models.dev
// section, from a public models.dev document, written by
// tools/limitsnapshot.py at each release (0089-MADR D3, DL70).
//
//go:embed limits_snapshot.json
var limitsSnapshot []byte

// limitsExtract is a parsed extract: its document and its date.
type limitsExtract struct {
	meta Metadata
	date time.Time
}

// names are the extract's source names by stage: exact, canonical id,
// family.
func (e limitsExtract) names() [3]string {
	d := e.date.Format(time.DateOnly)
	return [3]string{"models.dev (bundled " + d + ")", "models.dev (bundled " + d + ", canonical id)",
		"models.dev (bundled " + d + ", family)"}
}

// parseLimitsExtract reads an extract. The embedded one is a build defect
// when it does not parse, and panics; TestDiscover_EmbeddedExtract_0089 pins
// it.
func parseLimitsExtract(raw []byte) limitsExtract {
	var file struct {
		Date     string          `json:"date"`
		Sections json.RawMessage `json:"sections"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		panic(fmt.Sprintf("catalog: limits_snapshot.json: %v", err))
	}
	date, err := time.Parse(time.DateOnly, file.Date)
	if err != nil {
		panic(fmt.Sprintf("catalog: limits_snapshot.json: date: %v", err))
	}
	doc, err := decodeModelMetadata(bytes.NewReader(file.Sections))
	if err == nil {
		err = doc.decodeAll()
	}
	if err != nil {
		panic(fmt.Sprintf("catalog: limits_snapshot.json: %v", err))
	}
	return limitsExtract{meta: Metadata{doc: doc}, date: date}
}

// bundledExtract is the embedded extract, parsed once. Tests replace it.
var bundledExtract = sync.OnceValue(func() limitsExtract { return parseLimitsExtract(limitsSnapshot) })
