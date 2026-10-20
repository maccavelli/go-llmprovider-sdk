package catalog

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/chatgpt"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// RawListing is one provider's model listing as the provider sent it, with
// none of the SDK's curation (0041-MADR D9). The SDK promises its shape; the
// bytes inside are the provider's, and change when the provider changes them.
// It is safe for concurrent use only while no goroutine changes it.
type RawListing struct {
	// Provider is the provider listed.
	Provider llmprovider.ProviderID
	// Endpoint is the URL listed, without its query string.
	Endpoint string
	// FetchedAt is when the listing was read.
	FetchedAt time.Time
	// Pages is each page's body, byte for byte, in order: its envelope
	// (has_more, nextPageToken, object, and what a provider adds later)
	// included.
	Pages []json.RawMessage
	// Models is each entry of the pages' model arrays, in order.
	Models []RawModel
}

// RawModel is one entry of a provider's model array. It is safe for
// concurrent use only while no goroutine changes it.
type RawModel struct {
	// ID is the provider's own identifier field, unchanged: id for most;
	// name for Gemini, its models/ prefix included, and for Ollama; slug for
	// a ChatGPT session and a ChatGPT plan session. It is empty for an entry
	// with no such field, or one
	// that is not a string; the entry is returned all the same.
	ID string
	// Raw is the entry's bytes, unchanged.
	Raw json.RawMessage
}

// ListRaw lists id's models exactly as the provider sends them (0041-MADR
// D9). It sends the requests List sends, through the same code: the
// credential in the provider's header, WithBaseURL, WithHTTPClient and the
// user agent, OpenCode's session header, Kilo's organization endpoint, every
// page up to the listing's limit, 8 MiB a page, 10 s in all, and one renewal
// of a refused credential. An OpenAI listing with a ChatGPT session lists the
// Codex backend, and one with a ChatGPT plan session reads the "models"
// array of the platform's listing, as the openai provider's ListModels does
// (0043-MADR D5).
//
// It does none of List's curation: no usability filter, ranking, caveat or
// static fallback; no models.dev fetch; no listing cache read or write, so
// CachedFacts and CachedModel never see what it fetched; no de-duplication,
// sorting or id rewriting. Its errors are returned, never degraded: a failed
// page gives a zero RawListing and the classified error, naming the page, and
// so does a page that is not JSON or has no model array where the provider
// has one. A missing src, and a typed-nil one, give an error matching
// llmprovider.ErrInvalidRequest, as List's do (0050-MADR D4).
// Ollama's listing is /api/tags alone: per-model detail calls are not
// listings.
func ListRaw(ctx context.Context, id llmprovider.ProviderID, src llmprovider.TokenSource,
	opts ...llmprovider.Option) (RawListing, error) {
	if !listed[llmprovider.ProviderID(strings.ToLower(string(id)))] {
		return RawListing{}, unlistedProvider(id)
	}
	id = llmprovider.ProviderID(strings.ToLower(string(id)))
	cfg, err := configFor(id, opts)
	if err != nil {
		return RawListing{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, modelListingTimeout)
	defer cancel()
	token, err := listingToken(ctx, src, cfg)
	if err != nil {
		return RawListing{}, err
	}
	raw, err := listRawOnce(ctx, id, src, token, cfg)
	token, renewed, err2 := renewRefused(ctx, src, token, err)
	if err2 != nil {
		return RawListing{}, err2
	}
	if !renewed {
		return raw, err
	}
	return listRawOnce(ctx, id, src, token, cfg)
}

// listRawOnce lists once with token, recording each page as it arrives. A
// caller's client with no transport sends through the SDK's default one, as
// the snapshot command's recorder does, never net/http's shared one (R30).
func listRawOnce(ctx context.Context, id llmprovider.ProviderID, src llmprovider.TokenSource, token llmprovider.Token,
	cfg config) (RawListing, error) {
	rec := &rawRecorder{base: cfg.HTTPClient.Transport}
	if rec.base == nil {
		rec.base = transport.DefaultClient().Transport
	}
	client := *cfg.HTTPClient
	client.Transport = rec
	cfg.HTTPClient = &client
	cfg.sink = nil
	at := time.Now()
	chatGPT := id == llmprovider.ProviderOpenAI && chatgpt.IsSession(src)
	plan := id == llmprovider.ProviderOpenAI && chatgpt.IsPlanSession(src)
	var err error
	switch {
	case chatGPT:
		err = fetchChatGPTRaw(ctx, src, token, cfg)
	case plan:
		err = fetchPlanRaw(ctx, token, cfg)
	case id == llmprovider.ProviderOllama:
		_, err = fetchOllamaTags(ctx, token, cfg)
	default:
		_, err = fetchListing(ctx, id, token, cfg)
	}
	if err != nil {
		// The page that failed is the last one asked for.
		return RawListing{}, fmt.Errorf("model listing: %s page %d: %w", id, max(rec.requests(), 1), err)
	}
	return rec.listing(id, chatGPT || plan, at)
}

// fetchPlanRaw sends a plan session's listing request, the one the openai
// provider sends (internal/chatgpt), and checks its reply as a page.
func fetchPlanRaw(ctx context.Context, token llmprovider.Token, cfg config) error {
	req, err := chatgpt.PlanModelsRequest(ctx, cmp.Or(cfg.BaseURL, openAIListingBase), cfg.userAgent, token)
	if err != nil {
		return err
	}
	resp, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("model listing: plan models: %w", err)
	}
	defer cfg.closeBody(resp)
	if err := llmprovider.ClassifyHTTPError(string(llmprovider.ProviderOpenAI), resp); err != nil {
		return fmt.Errorf("model listing: %w", err)
	}
	var page struct {
		Models []json.RawMessage `json:"models"`
	}
	return decodeLimited(resp.Body, &page, "plan: models")
}

// fetchChatGPTRaw sends a ChatGPT session's listing request, the one the
// openai provider sends (internal/chatgpt), and checks its reply as a page.
func fetchChatGPTRaw(ctx context.Context, src llmprovider.TokenSource, token llmprovider.Token, cfg config) error {
	baseURL := chatgpt.BaseURL
	if cfg.BaseURL != "" {
		baseURL = cfg.BaseURL
	}
	req, err := chatgpt.ModelsRequest(ctx, baseURL, cfg.userAgent, token, src)
	if err != nil {
		return err
	}
	resp, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("model listing: chatgpt models: %w", err)
	}
	defer cfg.closeBody(resp)
	if err := llmprovider.ClassifyHTTPError(string(llmprovider.ProviderOpenAI), resp); err != nil {
		return fmt.Errorf("model listing: %w", err)
	}
	var page struct {
		Models []json.RawMessage `json:"models"`
	}
	return decodeLimited(resp.Body, &page, "chatgpt: models")
}

// rawRecorder keeps the body of each successful reply, in order, and hands
// the same bytes on, so the listing's own reading of each page is unchanged.
// It reads at most one byte past the page bound, which the listing then
// refuses.
type rawRecorder struct {
	base  http.RoundTripper
	mu    sync.Mutex
	sent  int
	pages []rawPage
}

// rawPage is one recorded reply: the URL asked for, and the body.
type rawPage struct {
	url  *url.URL
	body []byte
}

func (r *rawRecorder) RoundTrip(req *http.Request) (*http.Response, error) {
	r.mu.Lock()
	r.sent++
	r.mu.Unlock()
	resp, err := r.base.RoundTrip(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return resp, err
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, listingPageLimit+1))
	if cerr := resp.Body.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	r.pages = append(r.pages, rawPage{url: req.URL, body: body})
	r.mu.Unlock()
	resp.Body = io.NopCloser(bytes.NewReader(body))
	return resp, nil
}

// requests is how many requests were sent.
func (r *rawRecorder) requests() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.sent
}

// listing is the recorded pages as a RawListing for id.
func (r *rawRecorder) listing(id llmprovider.ProviderID, chatGPT bool, at time.Time) (RawListing, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.pages) == 0 {
		return RawListing{}, errors.New("model listing: no page was read")
	}
	endpoint := *r.pages[0].url
	endpoint.RawQuery = ""
	out := RawListing{Provider: id, Endpoint: endpoint.String(), FetchedAt: at}
	array, field := rawArray(id, chatGPT)
	for i, p := range r.pages {
		out.Pages = append(out.Pages, json.RawMessage(p.body))
		models, err := rawModels(p.body, array, field)
		if err != nil {
			return RawListing{}, fmt.Errorf("model listing: %s page %d: %w", id, i+1, err)
		}
		out.Models = append(out.Models, models...)
	}
	return out, nil
}

// rawArray is where id's pages keep their model array ("" for a bare array)
// and which field is the provider's own id (0041-MADR D9). chatGPT is set
// for a ChatGPT session and a plan session, which share the shape
// (0043-MADR M6).
func rawArray(id llmprovider.ProviderID, chatGPT bool) (array, field string) {
	switch {
	case chatGPT:
		return "models", "slug"
	case id == llmprovider.ProviderGemini, id == llmprovider.ProviderOllama:
		return "models", "name"
	case id == llmprovider.ProviderTogether:
		return "", "id"
	default:
		return "data", "id"
	}
}

// rawModels splits one page's model array, keeping each entry's bytes.
func rawModels(page []byte, array, field string) ([]RawModel, error) {
	var entries []json.RawMessage
	if array == "" {
		if err := json.Unmarshal(page, &entries); err != nil {
			return nil, fmt.Errorf("the body is not a model array: %w", err)
		}
	} else {
		var envelope map[string]json.RawMessage
		if err := json.Unmarshal(page, &envelope); err != nil {
			return nil, fmt.Errorf("the body is not a JSON object: %w", err)
		}
		raw, ok := envelope[array]
		if !ok {
			return nil, fmt.Errorf("no %q model array", array)
		}
		if err := json.Unmarshal(raw, &entries); err != nil {
			return nil, fmt.Errorf("%q is not an array: %w", array, err)
		}
	}
	models := make([]RawModel, 0, len(entries))
	for _, e := range entries {
		var fields map[string]json.RawMessage
		var modelID string
		if json.Unmarshal(e, &fields) == nil {
			if err := json.Unmarshal(fields[field], &modelID); err != nil {
				modelID = "" // no id field, or not a string: the entry is still the provider's
			}
		}
		models = append(models, RawModel{ID: modelID, Raw: e})
	}
	return models, nil
}
