package catalog

import (
	"context"
	"maps"
	"slices"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// The listings' metadata, cached in-process so that CachedModel never waits
// for a listing (0033-MADR D3, as amended on 2026-10-12). It lives as the
// models.dev cache does, and is keyed as the probe cache is: by provider,
// base URL, and a fingerprint of the credential, never the credential.

// listingMetadataTTL is how long a listing's metadata is fresh.
const listingMetadataTTL = 10 * time.Minute

// listingKey identifies one listing: who asked, where, for which Kilo
// organization, with which credential (0041-MADR A6).
type listingKey struct {
	provider llmprovider.ProviderID
	baseURL  string
	// organization is the Kilo organization listed, empty for the personal
	// account and for every other provider.
	organization string
	// credential is transport.Fingerprint's, never the credential.
	credential string
}

// listingKeyFor is the key of id's listing under cfg with token. Its base is
// the one the listing reads, so no base URL and the provider's default are
// one key (0041-MADR A8).
func listingKeyFor(id llmprovider.ProviderID, cfg config, token llmprovider.Token) listingKey {
	return listingKey{provider: id, baseURL: listingBaseURL(id, cfg.BaseURL), organization: cfg.KiloOrganization,
		credential: transport.Fingerprint(token.Value)}
}

// listingEntry is one listing's metadata by model id, and when it was read;
// and the listing itself, as fetched, and when (0041-MADR D6, A6).
type listingEntry struct {
	models  map[string]ModelInfo
	fetched time.Time
	// listing is nil until a listing with a usable id is fetched.
	listing  *fetchedListing
	listedAt time.Time
}

// lastGoodWindow is how old a listing may be and still be served after a
// transient failure (0041-MADR D6).
const lastGoodWindow = 24 * time.Hour

// listingFetchKey is one background fetch: a listing, and for Ollama, whose
// metadata is read one model at a time, the model.
type listingFetchKey struct {
	listingKey
	model string
}

// listingNow is the caches' clock. Tests move it.
var listingNow = time.Now

// storeListing records a listing's metadata. With merge, the models are added
// to a fresh entry rather than replacing it, as Ollama's are, one by one.
func (c *store) storeListing(key listingKey, models map[string]ModelInfo, merge bool) {
	c.listingMu.Lock()
	defer c.listingMu.Unlock()
	now := listingNow()
	e := c.listing[key]
	if !merge || e.models == nil || now.Sub(e.fetched) >= listingMetadataTTL {
		// The listing itself is kept: it has its own times.
		c.listing[key] = listingEntry{models: maps.Clone(models), fetched: now, listing: e.listing,
			listedAt: e.listedAt}
		return
	}
	next := maps.Clone(e.models)
	maps.Copy(next, models)
	c.listing[key] = listingEntry{models: next, fetched: e.fetched}
}

// cachedListing returns a listing's metadata, and whether it is fresh. Stale
// metadata is still returned, for the caller to use while it refreshes.
func (c *store) cachedListing(key listingKey) (map[string]ModelInfo, bool) {
	c.listingMu.Lock()
	defer c.listingMu.Unlock()
	e := c.listing[key]
	return e.models, e.models != nil && listingNow().Sub(e.fetched) < listingMetadataTTL
}

// storeFetched records a listing as fetched, and its models' metadata when
// it has any, and returns when it was read. A listing with no usable id is
// not kept: it is no last good listing (0041-MADR D6).
func (c *store) storeFetched(key listingKey, f fetchedListing, models map[string]ModelInfo) time.Time {
	if len(models) > 0 {
		c.storeListing(key, models, false)
	}
	c.listingMu.Lock()
	defer c.listingMu.Unlock()
	now := listingNow()
	if len(uniqueIDs(f.usable)) == 0 {
		return now
	}
	e := c.listing[key]
	e.listing, e.listedAt = &fetchedListing{usable: slices.Clone(f.usable), kilo: slices.Clone(f.kilo)}, now
	c.listing[key] = e
	return now
}

// freshListing is key's listing while it is younger than
// listingMetadataTTL, and when it was read.
func (c *store) freshListing(key listingKey) (fetchedListing, time.Time, bool) {
	c.listingMu.Lock()
	defer c.listingMu.Unlock()
	e := c.listing[key]
	if e.listing == nil || listingNow().Sub(e.listedAt) >= listingMetadataTTL {
		return fetchedListing{}, time.Time{}, false
	}
	return *e.listing, e.listedAt, true
}

// lastGoodListing is the listing a transient failure serves: key's cached
// one, read within lastGoodWindow, else the one WithLastGood hands in,
// within the same window (0041-MADR D6).
func (c *store) lastGoodListing(key listingKey, cfg config) (fetchedListing, time.Time, bool) {
	c.listingMu.Lock()
	e := c.listing[key]
	now := listingNow()
	c.listingMu.Unlock()
	if e.listing != nil && now.Sub(e.listedAt) < lastGoodWindow {
		return *e.listing, e.listedAt, true
	}
	if len(cfg.lastGood.ids) > 0 && !cfg.lastGood.at.IsZero() && now.Sub(cfg.lastGood.at) < lastGoodWindow {
		return fetchedListing{usable: slices.Clone(cfg.lastGood.ids)}, cfg.lastGood.at, true
	}
	return fetchedListing{}, time.Time{}, false
}

// claimListing makes the caller key's one fetcher, with done to call when
// its fetch ends; or, when another caller is fetching key, returns wait,
// closed when that fetch ends (0041-MADR D6).
func (c *store) claimListing(key listingKey) (done func(), wait <-chan struct{}) {
	c.listingMu.Lock()
	defer c.listingMu.Unlock()
	if ch, ok := c.listingClaims[key]; ok {
		return nil, ch
	}
	ch := make(chan struct{})
	c.listingClaims[key] = ch
	return func() {
		c.listingMu.Lock()
		delete(c.listingClaims, key)
		c.listingMu.Unlock()
		close(ch)
	}, nil
}

// startListingFetch refreshes a listing in the background, detached from ctx
// and bounded as a listing is, unless the same fetch is already running.
func (c *store) startListingFetch(ctx context.Context, id llmprovider.ProviderID, src llmprovider.TokenSource, model string, cfg config, key listingKey) {
	fk := listingFetchKey{listingKey: key}
	if id == llmprovider.ProviderOllama {
		fk.model = model
	}
	c.listingMu.Lock()
	if c.listingInFlight[fk] {
		c.listingMu.Unlock()
		return
	}
	c.listingInFlight[fk] = true
	c.listingMu.Unlock()
	fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), modelListingTimeout)
	c.listingFetching.Go(func() {
		defer cancel()
		defer func() {
			c.listingMu.Lock()
			delete(c.listingInFlight, fk)
			c.listingMu.Unlock()
		}()
		// Nothing waits for this fetch, so its failure goes only to the
		// caller's logger (R31); the next lookup tries again.
		if _, _, err := fetchListingModel(fetchCtx, id, src, model, cfg); err != nil && cfg.logger != nil {
			cfg.logger.Debug("catalog: background listing", "provider", string(id), "error", err)
		}
	})
}

// fetchListingModel reads one model's metadata from its provider: Ollama's
// /api/show for the model, or the provider's listing, which List and this
// share. What it reads is cached.
func fetchListingModel(ctx context.Context, id llmprovider.ProviderID, src llmprovider.TokenSource, model string,
	cfg config) (ModelInfo, bool, error) {
	if id == llmprovider.ProviderOllama {
		token, err := src.Token(ctx)
		if err != nil {
			return ModelInfo{}, false, err
		}
		info, err := fetchOllamaShow(ctx, token, cfg, model)
		if err != nil {
			return ModelInfo{}, false, err
		}
		cfg.store.storeListing(listingKeyFor(id, cfg, token), map[string]ModelInfo{model: info}, true)
		return info, true, nil
	}
	cat, models, err := listAndRecord(ctx, id, src, cfg)
	if err == nil {
		err = cat.Err
	}
	info, ok := models[model]
	return info, ok, err
}
