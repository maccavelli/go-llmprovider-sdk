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
// it out of Recommended (0041-MADR D3), except CaveatNoTools on Ollama
// (0042-MADR D2). It is a string, safe for concurrent use.
type Caveat string

// The caveats, each read from one fact.
const (
	// CaveatDeprecated is a model whose status is deprecated or retired.
	CaveatDeprecated Caveat = "deprecated"
	// CaveatNoTools is a model that takes no tools. On Ollama it stays in
	// Recommended, after the models that take tools (0042-MADR D2).
	CaveatNoTools Caveat = "no-tools"
	// CaveatNoTextOutput is a model that writes no text.
	CaveatNoTextOutput Caveat = "no-text-output"
	// CaveatExpiring is a model whose published retirement or shutdown date
	// is within 30 days.
	CaveatExpiring Caveat = "expiring"
	// CaveatUnavailable is a model measured to fail at request.
	CaveatUnavailable Caveat = "unavailable"
)

// MenuEntry is one model of Catalog.Menu (0041-MADR D3). It is safe for
// concurrent use only while no goroutine changes it.
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
			dev, devOK = meta.modelFor(cfg, provider, id)
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
// that may write text, by the default rank with their variants grouped,
// where they were in /api/tags order (0061-MADR, amendment "E2 and E4's
// menu, made exact before Phase 4c", decision 2, X2, X3; 0041-MADR A4). A
// model Ollama says writes no text, such as an embedding model, is not
// usable, as on every other provider (0042-MADR D1). Recommended is the menu
// less each entry with a caveat other than CaveatNoTools: the install is all
// there is, so a chat model that takes no tools stays recommended, after
// those that do (0042-MADR D2).
func ollamaMenu(names []string, facts map[string]ModelInfo, created map[string]time.Time, limit int,
	now time.Time) Catalog {
	usable := slices.DeleteFunc(slices.Clone(names), func(id string) bool { return writesNoText(facts[id]) })
	menu := defaultRank(usable, facts, created, func(id string, info ModelInfo) []Caveat {
		return caveatsOf(llmprovider.ProviderOllama, id, info, now)
	})
	var recommended []string
	for _, e := range menu {
		if ollamaRecommends(e) {
			recommended = append(recommended, e.ID)
		}
	}
	return Catalog{Recommended: listlimit.Cut(recommended, limit), Usable: usable, Live: true,
		Menu: cutMenu(menu, limit)}
}

// ollamaRecommends reports whether an Ollama menu entry is recommended: it
// has no caveat but CaveatNoTools (0042-MADR D2).
func ollamaRecommends(e MenuEntry) bool {
	return !slices.ContainsFunc(e.Caveats, func(c Caveat) bool { return c != CaveatNoTools })
}

// compatMenu is a chatcompat listing's catalog (0061-MADR, amendment "E2 and
// E4's menu, made exact before Phase 4c", decision 2, X2, X3): the default
// rank with the variants grouped; the band its first MaxListed uncaveated
// entries, ranked by place; Recommended the uncaveated, in that order.
// Usable stays in the server's order. limit cuts Recommended and Menu, as
// WithLimit does.
func compatMenu(provider llmprovider.ProviderID, usable []string, facts map[string]ModelInfo,
	created map[string]time.Time, limit int, now time.Time) Catalog {
	menu := defaultRank(usable, facts, created, func(id string, info ModelInfo) []Caveat {
		return compatCaveats(provider, id, info, now)
	})
	var recommended []string
	for i := range menu {
		if len(menu[i].Caveats) > 0 {
			continue
		}
		recommended = append(recommended, menu[i].ID)
		if len(recommended) <= MaxListed {
			menu[i].Rank = len(recommended)
		}
	}
	return Catalog{Recommended: listlimit.Cut(recommended, limit), Usable: usable, Live: true,
		Menu: cutMenu(menu, limit)}
}

// noTextTokens are the id tokens of a model that writes no text: an
// embedding, a reranker, speech (X3).
var noTextTokens = []string{denyEmbed, "embedding", "embeddings", "rerank", "reranker", denyWhisper, denyTTS}

// compatCaveats is a chatcompat model's caveats: caveatsOf's, and
// CaveatNoTextOutput for a model whose facts do not state its output and
// whose id has a token of noTextTokens (X3). The built-in Ollama reads no
// name (0042-MADR D1; PLAN deviation 8).
func compatCaveats(provider llmprovider.ProviderID, id string, info ModelInfo, now time.Time) []Caveat {
	out := caveatsOf(provider, id, info, now)
	if !info.Has(FieldOutput) && slices.ContainsFunc(modelTokens(id), func(t string) bool {
		return slices.Contains(noTextTokens, t)
	}) {
		out = append(out, CaveatNoTextOutput)
	}
	return out
}

// defaultRank orders an unranked listing (X3): the uncaveated before the
// caveated; then the larger context window, a known one first; then the
// newer created, a known one first; then the listing's order. Within the
// uncaveated, and within the caveated, the variants of one model then sit
// together (X2). No dialect recorded says which model is loaded, so that
// key is not read. caveats is the listing's caveats of a model.
func defaultRank(ids []string, facts map[string]ModelInfo, created map[string]time.Time,
	caveats func(id string, info ModelInfo) []Caveat) []MenuEntry {
	entries := make([]MenuEntry, len(ids))
	for i, id := range ids {
		info := facts[id]
		entries[i] = MenuEntry{ID: id, Info: info, Caveats: caveats(id, info)}
	}
	window := func(e MenuEntry) int {
		if e.Info.Has(FieldContextWindow) {
			return e.Info.ContextWindow
		}
		return 0
	}
	slices.SortStableFunc(entries, func(a, b MenuEntry) int {
		ta, oka := created[a.ID]
		tb, okb := created[b.ID]
		var byCreated int
		switch {
		case oka && okb:
			byCreated = tb.Compare(ta)
		case oka != okb:
			byCreated = boolOrder(okb, oka)
		}
		return cmp.Or(boolOrder(len(a.Caveats) > 0, len(b.Caveats) > 0), cmp.Compare(window(b), window(a)), byCreated)
	})
	split := slices.IndexFunc(entries, func(e MenuEntry) bool { return len(e.Caveats) > 0 })
	if split < 0 {
		split = len(entries)
	}
	place := make(map[string]int, len(ids))
	for i, id := range ids {
		place[id] = i
	}
	return slices.Concat(groupVariants(entries[:split], place), groupVariants(entries[split:], place))
}

// boolOrder orders false before true.
func boolOrder(a, b bool) int {
	switch {
	case a == b:
		return 0
	case a:
		return 1
	}
	return -1
}

// groupVariants keeps entries in order, except that each model's variants,
// those of one variantKey, sit at the first one's place, the plainest
// first, then by place, their listing's order (X2). An id with no key is a
// group of its own.
func groupVariants(entries []MenuEntry, place map[string]int) []MenuEntry {
	keys := make([]string, len(entries))
	for i, e := range entries {
		keys[i] = variantKey(e.ID)
	}
	out := make([]MenuEntry, 0, len(entries))
	done := make([]bool, len(entries))
	for i := range entries {
		if done[i] {
			continue
		}
		group := []MenuEntry{entries[i]}
		done[i] = true
		for j := i + 1; j < len(entries) && keys[i] != ""; j++ {
			if !done[j] && keys[j] == keys[i] {
				group = append(group, entries[j])
				done[j] = true
			}
		}
		slices.SortStableFunc(group, func(a, b MenuEntry) int {
			return cmp.Or(cmp.Compare(plainness(a.ID), plainness(b.ID)), cmp.Compare(place[a.ID], place[b.ID]))
		})
		out = append(out, group...)
	}
	return out
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
