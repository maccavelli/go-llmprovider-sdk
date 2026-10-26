package catalog

import (
	"context"
	"net/http"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/compatdialect"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// A chatcompat server's dialect (0061-MADR E2; its amendment "E2 and E4's
// menu, made exact before Phase 4c", decision 3, X4): internal/compatdialect
// tells which server's /models reply it is; this file reads the one details
// endpoint each dialect has.

// dialectDetails reads a dialect's server's facts of each listed model, and
// emits them as the listing's, SourceOf listing:<id> (X4), by dialect name.
var dialectDetails = map[string]func(ctx context.Context, id llmprovider.ProviderID, token llmprovider.Token, cfg config,
	ids []string){
	"ollama": ollamaDialectDetails,
}

// ollamaDialectDetails reads an Ollama server's /api/tags and /api/show at its
// root, the base URL less a trailing /v1, through the digest-cached reader
// the built-in Ollama's listing uses (X4), and emits each model's facts as
// the listing's. A failed read emits nothing: the listing stands without
// its details.
func ollamaDialectDetails(ctx context.Context, id llmprovider.ProviderID, token llmprovider.Token, cfg config, ids []string) {
	root := cfg
	root.BaseURL = strings.TrimSuffix(strings.TrimRight(cfg.BaseURL, "/"), "/v1")
	tags, err := fetchOllamaTags(ctx, token, root)
	if err != nil {
		return
	}
	digests := make(map[string]string, len(tags))
	var names []string
	for _, t := range tags {
		digests[t.Name] = t.Digest
		names = append(names, t.Name)
	}
	key := listingKeyFor(id, cfg, token)
	cached := cfg.store.freshShows(key, digests)
	shows := ollamaShows(ctx, token, root, names, cached)
	for name, info := range shows {
		if _, reused := cached[name]; !reused {
			cfg.store.storeShow(key, name, info, digests[name])
		}
	}
	listed := make(map[string]bool, len(ids))
	for _, m := range ids {
		listed[m] = true
	}
	for _, t := range tags {
		if !listed[t.Name] {
			continue
		}
		info := newInfo(id, t.Name).done()
		overlay(&info, ollamaTagInfo(t), listingLayer(id))
		if show, ok := shows[t.Name]; ok {
			overlay(&info, show, listingLayer(id))
		}
		cfg.emit(info)
	}
}

// fetchCompatEntries is a chatcompat server's GET {base}/models: each entry,
// in the server's order, with no Authorization header for an empty key, as
// the provider sends none (0061-MADR D4; its amendment "chatcompat,
// provider-agnostic", E1).
func fetchCompatEntries(ctx context.Context, id llmprovider.ProviderID, token llmprovider.Token, cfg config) ([]compatdialect.Entry,
	error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.BaseURL+"/models", http.NoBody)
	if err != nil {
		return nil, err
	}
	cfg.setUserAgent(req)
	if token.Value != "" {
		header, value := bearerHeader(token)
		req.Header.Set(header, value)
	}
	resp, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, transport.WrapOffline(ctx, err, llmprovider.ErrOffline)
	}
	defer cfg.closeBody(resp)
	if resp.StatusCode != http.StatusOK {
		return nil, llmprovider.ClassifyHTTPError(string(id), resp)
	}
	var result struct {
		Data []compatdialect.Entry `json:"data"`
	}
	if err := decodeLimited(resp.Body, &result, string(id)+": models"); err != nil {
		return nil, err
	}
	return result.Data, nil
}
