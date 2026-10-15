package catalog

import (
	"cmp"
	"context"
	"slices"
	"sync"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/listlimit"
)

// Caveat is a reason a listed model should not be the default. A caveat
// does not drop the model: it sinks it to the end of Catalog.Menu, and keeps
// it out of Recommended (0041-MADR D3).
type Caveat string

// The caveats, each read from one fact.
const (
	// CaveatDeprecated is a model whose status is deprecated or retired.
	CaveatDeprecated Caveat = "deprecated"
	// CaveatNoTools is a model that takes no tools.
	CaveatNoTools Caveat = "no-tools"
	// CaveatNoTextOutput is a model that writes no text.
	CaveatNoTextOutput Caveat = "no-text-output"
	// CaveatExpiring is a model whose published retirement or shutdown date
	// is within 30 days.
	CaveatExpiring Caveat = "expiring"
	// CaveatUnavailable is a model measured to fail at request.
	CaveatUnavailable Caveat = "unavailable"
)

// MenuEntry is one model of Catalog.Menu (0041-MADR D3).
type MenuEntry struct {
	// ID is the model's id, as the provider lists it.
	ID string
	// Info is what is known of the model, merged as CachedFacts merges it:
	// the listing's facts, the models.dev document's, and the caller's
	// layers. SourceOf names the source of each field.
	Info ModelInfo
	// Rank is the model's place in the recommended band, 1 to MaxListed,
	// or 0 outside it. A static catalog ranks each id by its place.
	Rank int
	// Caveats are the reasons the model should not be the default, or nil.
	Caveats []Caveat
}

// expiringWithin is how near a retirement date makes CaveatExpiring.
const expiringWithin = minRankExpiryDays * 24 * time.Hour

// caveatsOf reads model's caveats from its facts (0041-MADR D3).
func caveatsOf(provider llmprovider.ProviderID, model string, info ModelInfo, now time.Time) []Caveat {
	var out []Caveat
	if info.Has(FieldStatus) && (info.Status == StatusDeprecated || info.Status == StatusRetired) {
		out = append(out, CaveatDeprecated)
	}
	if info.Has(FieldTools) && !info.Tools {
		out = append(out, CaveatNoTools)
	}
	if writesNoText(info) {
		out = append(out, CaveatNoTextOutput)
	}
	if info.Has(FieldRetires) && !info.Retires.After(now.Add(expiringWithin)) {
		out = append(out, CaveatExpiring)
	}
	if excludedAtRequest(provider, model) {
		out = append(out, CaveatUnavailable)
	}
	return out
}

// newestFirst orders ids by release date, newest first, a dated id before
// an undated one, then by id (0041-MADR D3, A4).
func newestFirst(ids []string, facts map[string]ModelInfo) {
	slices.SortFunc(ids, func(a, b string) int {
		fa, fb := facts[a], facts[b]
		da, db := fa.Has(FieldReleased), fb.Has(FieldReleased)
		switch {
		case da && db:
			if c := fb.Released.Compare(fa.Released); c != 0 {
				return c
			}
		case da != db:
			if da {
				return -1
			}
			return 1
		}
		return cmp.Compare(a, b)
	})
}

// factsOf merges, for each id, the listing's facts, the document's and the
// caller's layers, as CachedFacts does (0041-MADR D1). doc may be nil.
func factsOf(provider llmprovider.ProviderID, cfg config, ids []string, listing map[string]ModelInfo,
	doc *modelMetadataDoc) map[string]ModelInfo {
	meta := Metadata{doc: doc}
	facts := make(map[string]ModelInfo, len(ids))
	for _, id := range ids {
		var dev ModelInfo
		devOK := false
		if doc != nil {
			dev, devOK = meta.Model(provider, id)
		}
		lst, lstOK := listing[id]
		facts[id] = mergeLayers(provider, id, cfg.layers, dev, devOK, lst, lstOK)
	}
	return facts
}

// entry is id's menu entry.
func entry(provider llmprovider.ProviderID, id string, facts map[string]ModelInfo, rank int, now time.Time) MenuEntry {
	info := facts[id]
	return MenuEntry{ID: id, Info: info, Rank: rank, Caveats: caveatsOf(provider, id, info, now)}
}

// menuOf is a live listing's catalog (0041-MADR D3, A4). The band is
// curate's six of the uncaveated ids, by rank. The other uncaveated ids
// follow, newest first, then the caveated, the same way. Recommended is the
// menu less its caveated entries. limit cuts both, as WithLimit does.
func menuOf(provider llmprovider.ProviderID, usable []string, facts map[string]ModelInfo, curate curator, limit int,
	now time.Time) Catalog {
	var uncaveated, sunk []string
	for _, id := range usable {
		if len(caveatsOf(provider, id, facts[id], now)) == 0 {
			uncaveated = append(uncaveated, id)
		} else {
			sunk = append(sunk, id)
		}
	}
	// The curator ranks only the uncaveated, so that no caveated id is in
	// the band. A curator gives distinct ids of those it is given, at most
	// MaxListed of them.
	band := curate(slices.Clone(uncaveated), MaxListed)
	rest := slices.DeleteFunc(slices.Clone(uncaveated), func(id string) bool { return slices.Contains(band, id) })
	newestFirst(rest, facts)
	newestFirst(sunk, facts)
	menu := make([]MenuEntry, 0, len(usable))
	for i, id := range band {
		menu = append(menu, entry(provider, id, facts, i+1, now))
	}
	for _, id := range slices.Concat(rest, sunk) {
		menu = append(menu, entry(provider, id, facts, 0, now))
	}
	return Catalog{Recommended: listlimit.Cut(slices.Concat(band, rest), limit), Usable: usable, Live: true,
		Menu: cutMenu(menu, limit)}
}

// staticMenu is a static catalog with its menu: the static ids in their
// order, each ranked by its place, with what facts are cached. Recommended
// stays equal to Usable (0038-MADR D3).
func staticMenu(provider llmprovider.ProviderID, static []string, facts map[string]ModelInfo, limit int,
	now time.Time) Catalog {
	cat := staticCatalog(static, limit)
	cat.Menu = make([]MenuEntry, 0, len(cat.Usable))
	for i, id := range cat.Usable {
		cat.Menu = append(cat.Menu, entry(provider, id, facts, i+1, now))
	}
	return cat
}

// writesNoText reports whether info says the model writes no text: its
// output is known, and text is not in it (0041-MADR D3; 0042-MADR D1).
func writesNoText(info ModelInfo) bool {
	return info.Has(FieldOutput) && !slices.Contains(info.Output, ModalityText)
}

// ollamaMenu is an Ollama listing's catalog: no band; the installed models
// that may write text, in /api/tags order, the caveated after the others and
// out of Recommended (0041-MADR A4). A model Ollama says writes no text, such
// as an embedding model, is not usable, as on every other provider
// (0042-MADR D1).
func ollamaMenu(names []string, facts map[string]ModelInfo, limit int, now time.Time) Catalog {
	usable := slices.DeleteFunc(slices.Clone(names), func(id string) bool { return writesNoText(facts[id]) })
	var uncaveated, sunk []MenuEntry
	for _, id := range usable {
		e := entry(llmprovider.ProviderOllama, id, facts, 0, now)
		if len(e.Caveats) == 0 {
			uncaveated = append(uncaveated, e)
		} else {
			sunk = append(sunk, e)
		}
	}
	return Catalog{Recommended: listlimit.Cut(menuIDsOf(uncaveated), limit), Usable: usable, Live: true,
		Menu: cutMenu(slices.Concat(uncaveated, sunk), limit)}
}

// menuIDsOf is the entries' ids, in order.
func menuIDsOf(entries []MenuEntry) []string {
	ids := make([]string, len(entries))
	for i, e := range entries {
		ids[i] = e.ID
	}
	return ids
}

// cutMenu is menu's first n, or all of it when n is zero or less.
func cutMenu(menu []MenuEntry, n int) []MenuEntry {
	if n > 0 && len(menu) > n {
		return menu[:n:n]
	}
	return menu
}

// ollamaShowConcurrency is how many /api/show requests a listing sends at a
// time.
const ollamaShowConcurrency = 4

// ollamaShows reads /api/show for each installed model not already cached,
// a few at a time, under ctx. cached is the fresh cached replies. A failed
// reply leaves its model out, with no caveat (0041-MADR A4).
func ollamaShows(ctx context.Context, token llmprovider.Token, cfg config, names []string,
	cached map[string]ModelInfo) map[string]ModelInfo {
	out := make(map[string]ModelInfo, len(names))
	var mu sync.Mutex
	var wg sync.WaitGroup
	slots := make(chan struct{}, ollamaShowConcurrency)
	for _, name := range names {
		if info, ok := cached[name]; ok {
			out[name] = info
			continue
		}
		wg.Go(func() {
			slots <- struct{}{}
			defer func() { <-slots }()
			info, err := fetchOllamaShow(ctx, token, cfg, name)
			if err != nil {
				return
			}
			mu.Lock()
			out[name] = info
			mu.Unlock()
		})
	}
	wg.Wait()
	return out
}
