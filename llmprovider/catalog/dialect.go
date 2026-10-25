package catalog

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// A chatcompat server's dialect (0061-MADR E2; its amendment "E2 and E4's
// menu, made exact before Phase 4c", decision 3, X4): which server's
// /models reply it is, by the marks of each entry, and the one details
// reader that dialect has. A dialect is in the table only with a recorded
// reply of its server (testdata/compat/<dialect>/); a listing that scores
// none is a plain listing.

// dialectThreshold is the share of a dialect's marks a listing must meet:
// every mark, on every entry (decision 3).
const dialectThreshold = 1.0

// compatEntry is one entry of a chatcompat server's GET {base}/models: the
// id E1 lists, and what the dialects' marks and the default rank read.
type compatEntry struct {
	ID      string          `json:"id"`
	Created json.RawMessage `json:"created"`
	OwnedBy *string         `json:"owned_by"`
}

// created is the entry's created time, and whether it is an integer of Unix
// seconds.
func (e compatEntry) created() (time.Time, bool) {
	var unix int64
	if len(e.Created) == 0 || json.Unmarshal(e.Created, &unix) != nil || unix <= 0 {
		return time.Time{}, false
	}
	return time.Unix(unix, 0).UTC(), true
}

// dialect is one server's marks and details reader.
type dialect struct {
	name  string
	marks []func(compatEntry) bool
	// details reads the server's facts of each listed model and emits them
	// as the listing's, SourceOf listing:<id> (X4).
	details func(ctx context.Context, id llmprovider.ProviderID, token llmprovider.Token, cfg config, ids []string)
}

// dialects are the recorded dialects.
var dialects = []dialect{
	{
		name: "ollama",
		marks: []func(compatEntry) bool{
			func(e compatEntry) bool { return strings.Contains(e.ID, ":") },
			func(e compatEntry) bool { _, ok := e.created(); return ok },
			func(e compatEntry) bool { return e.OwnedBy != nil },
		},
		details: ollamaDialectDetails,
	},
}

// dialectScore is the share of d's marks entries meet, 0 for none.
func dialectScore(d dialect, entries []compatEntry) float64 {
	if len(entries) == 0 || len(d.marks) == 0 {
		return 0
	}
	met := 0
	for _, e := range entries {
		for _, mark := range d.marks {
			if mark(e) {
				met++
			}
		}
	}
	return float64(met) / float64(len(entries)*len(d.marks))
}

// dialectOf is the dialect entries score at its threshold, the first in the
// table, and whether one does.
func dialectOf(entries []compatEntry) (dialect, bool) {
	for _, d := range dialects {
		if dialectScore(d, entries) >= dialectThreshold {
			return d, true
		}
	}
	return dialect{}, false
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
func fetchCompatEntries(ctx context.Context, id llmprovider.ProviderID, token llmprovider.Token, cfg config) ([]compatEntry, error) {
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
		Data []compatEntry `json:"data"`
	}
	if err := decodeLimited(resp.Body, &result, string(id)+": models"); err != nil {
		return nil, err
	}
	return result.Data, nil
}
