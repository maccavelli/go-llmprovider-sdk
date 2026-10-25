package wizard

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// Search-then-select model flow (MADR 0007 §4–§5). Every interaction uses the
// existing Prompter methods; Prompter itself does not change.
//
// otherModelLabel is the trailing escape hatch on the model menu. A live
// listing can lag a newly released model, and the catalog is curated rather
// than exhaustive, so the user must always be able to name a model directly.
const (
	otherModelLabel      = "Other (enter a model id)"
	searchAgainLabel     = "Search again"
	useQueryLabel        = "Use %s as the model id"
	noRecommendedNotice  = "no %s model meets the profile; search the listing for one"
	currentModelDetail   = "current"
	savedFallbackDetail  = "saved"
	chooseModelTitle     = "Choose a %s model:"
	searchResultsTitle   = "Choose a %s model (%s for %q):"
	searchModelsPrompt   = "Search models: a name, a family, or a feature such as vision, tools, 128k (blank for the recommended)"
	searchFallbackPrompt = "Search fallback models (blank for recommended)"
	chooseFallbacksTitle = "Choose fallback models (optional):"
	searchFallbacksTitle = "Choose fallback models:"
	moreFallbacksPrompt  = "Search for more fallback models?"
	moreMatchesLabel     = "More (%d of %d shown)"
	maxSearchResults     = 20
	// searchPage is how many matches the model menu shows at once, before
	// its More row (0061-MADR, amendment "Model search, rebuilt", S4).
	searchPage = 10
)

// selectModel asks for a search query. A blank query shows the recommended
// menu; any other query searches every usable model, by its facts too on a
// listing List built, and offers the matches ten at a time, each with its
// facts and why it matched, under a title that counts them, with More while
// any are unshown, then Search again and Other (0061-MADR, amendment "Model
// search, rebuilt", S4). A query with no match offers itself as the model
// id, or Search again. Each round checks ctx, so a prompter that accepts
// every default stops when ctx ends (R40; 0026-MADR F9).
func selectModel(ctx context.Context, p Prompter, d llmprovider.Descriptor, cat catalog.Catalog, o Options) (string, error) {
	title := fmt.Sprintf(chooseModelTitle, d.Label)
	for {
		if err := ctx.Err(); err != nil {
			return "", fmt.Errorf("select model: %w", err)
		}
		q, err := p.Input(searchModelsPrompt, "")
		if err != nil {
			return "", fmt.Errorf("search models: %w", err)
		}
		q = strings.TrimSpace(q)
		if q == "" && len(cat.Recommended) == 0 {
			// A live listing in which nothing meets the profile (0021-MADR
			// C13): the user searches it. A blank query offers the current
			// model, if any, and Other, never the same prompt again, so
			// accepting defaults ends here (0026-MADR F9).
			p.Notify(LevelWarn, noRecommendedNotice, d.Label)
			return selectRecommended(p, d, nil, o)
		}
		if q == "" {
			return selectRecommended(p, d, cat.Recommended, o)
		}
		matches := searchCatalog(d, cat, nil, q)
		if len(matches) == 0 {
			// A model the listing does not have may still be served: offer
			// the query itself (0021-MADR Z11). Search again is the default.
			p.Notify(LevelWarn, "no %s models match %q", d.Label, q)
			idx, err := choose(p, title, []Choice{{Label: fmt.Sprintf(useQueryLabel, q)}, {Label: searchAgainLabel}}, 1)
			if err != nil {
				return "", fmt.Errorf("select model: %w", err)
			}
			if idx == 0 {
				return q, nil
			}
			continue
		}
		id, other, err := pageMatches(ctx, p, fmt.Sprintf(searchResultsTitle, d.Label, matchCount(len(matches)), q),
			matches, menuFacts(cat))
		switch {
		case err != nil:
			return "", err
		case other:
			return enterModelID(p, d.ID, o)
		case id != "":
			return id, nil
		}
	}
}

// pageMatches offers matches ten at a time, the first row the default: the
// strongest match, the exact id when one was typed. More shows the next ten;
// Search again returns no id; Other returns other.
func pageMatches(ctx context.Context, p Prompter, title string, matches []catalog.Match,
	facts map[string]catalog.ModelInfo) (id string, other bool, err error) {
	for start := 0; ; {
		if err := ctx.Err(); err != nil {
			return "", false, fmt.Errorf("select model: %w", err)
		}
		end := min(start+searchPage, len(matches))
		page := matches[start:end]
		choices := matchChoices(page, facts)
		more := end < len(matches)
		if more {
			choices = append(choices, Choice{Label: fmt.Sprintf(moreMatchesLabel, end, len(matches))})
		}
		choices = append(choices, Choice{Label: searchAgainLabel}, Choice{Label: otherModelLabel})
		idx, err := choose(p, title, choices, 0)
		if err != nil {
			return "", false, fmt.Errorf("select model: %w", err)
		}
		switch {
		case idx < len(page):
			return page[idx].ID, false, nil
		case more && idx == len(page):
			start = end
		case idx == len(choices)-2:
			return "", false, nil
		default:
			return "", true, nil
		}
	}
}

// matchCount is "1 match" or "n matches".
func matchCount(n int) string {
	if n == 1 {
		return "1 match"
	}
	return strconv.Itoa(n) + " matches"
}

// searchCatalog searches cat for q, less the ids in exclude: by Catalog.Search,
// facts included, on a listing List built, which names its Source; else by
// catalog.Search, over Usable's ids with the provider's labels (0061-MADR,
// amendment "Model search, rebuilt", S1 and "S1 amended in execution").
func searchCatalog(d llmprovider.Descriptor, cat catalog.Catalog, exclude map[string]struct{}, q string) []catalog.Match {
	if cat.Source == "" {
		return catalog.Search(d.ID, without(cat.Usable, exclude), q)
	}
	if len(exclude) > 0 {
		cat.Usable = without(cat.Usable, exclude)
		cat.Menu = slices.DeleteFunc(slices.Clone(cat.Menu), func(e catalog.MenuEntry) bool {
			_, skip := exclude[strings.ToLower(e.ID)]
			return skip
		})
	}
	return cat.Search(q)
}

// menuFacts is each Menu entry's facts, by its id in lower case, the first
// of an id kept.
func menuFacts(cat catalog.Catalog) map[string]catalog.ModelInfo {
	out := make(map[string]catalog.ModelInfo, len(cat.Menu))
	for _, e := range cat.Menu {
		if _, seen := out[strings.ToLower(e.ID)]; !seen {
			out[strings.ToLower(e.ID)] = e.Info
		}
	}
	return out
}

// selectRecommended shows the curated menu. A previously configured model for
// the same provider that is not recommended is listed after the recommended
// rows, marked current, and is the default, so pressing Enter keeps it.
func selectRecommended(p Prompter, d llmprovider.Descriptor, models []string, o Options) (string, error) {
	choices := modelChoices(d.ID, models)
	// The saved model is the default only for its own provider (MADR 0007
	// §4.3; 0026-MADR F46).
	defaultIdx, listed := 0, false
	for i, m := range models {
		if o.Existing.Provider == d.ID && m == o.Existing.Model {
			defaultIdx, listed = i, true
		}
	}
	current := ""
	if !listed && o.Existing.Provider == d.ID && o.Existing.Model != "" {
		current = o.Existing.Model
		choices = append(choices, Choice{Label: catalog.Label(d.ID, current), Detail: currentModelDetail})
		defaultIdx = len(models)
	}
	choices = append(choices, Choice{Label: otherModelLabel})
	idx, err := choose(p, fmt.Sprintf(chooseModelTitle, d.Label), choices, defaultIdx)
	if err != nil {
		return "", fmt.Errorf("select model: %w", err)
	}
	switch {
	case idx < len(models):
		return models[idx], nil
	case current != "" && idx == len(models):
		return current, nil
	default:
		return enterModelID(p, d.ID, o)
	}
}

// enterModelID is the Other escape hatch: the user types a model id. The
// saved model is the default only for its own provider (MADR 0007 §4.3), and a
// blank id is refused (MADR 0013 C4, C7).
func enterModelID(p Prompter, provider llmprovider.ProviderID, o Options) (string, error) {
	manual, err := p.Input("Model id", existingModel(o, provider))
	if err != nil {
		return "", fmt.Errorf("enter model: %w", err)
	}
	manual = strings.TrimSpace(manual)
	if manual == "" {
		return "", fmt.Errorf("wizard: no model entered")
	}
	return manual, nil
}

// existingFallbacks returns the saved fallbacks when they belong to
// provider, else nil.
func existingFallbacks(o Options, provider llmprovider.ProviderID) []string {
	if o.Existing.Provider == provider {
		return o.Existing.Fallbacks
	}
	return nil
}

// existingModel returns the saved model when it belongs to provider, else "".
func existingModel(o Options, provider llmprovider.ProviderID) string {
	if o.Existing.Provider == provider {
		return o.Existing.Model
	}
	return ""
}

// selectFallbacks offers fallback models, never the primary or one already
// chosen. A blank query offers the remaining recommended models and ends the
// loop; a search round offers the matches and asks whether to search again.
// The result is nil when no MultiSelect was shown, and non-nil (possibly empty)
// once one was, which is the shape this function has always returned. The
// saved fallbacks a round offers are preselected (0020-MADR F50); a saved one
// that is not recommended is offered in the blank round after the recommended
// rows, marked saved, so that a blank answer keeps it, and the loop ends early
// only when nothing at all is left (0059-MADR W7). Each round checks ctx
// (0026-MADR F9).
func selectFallbacks(ctx context.Context, p Prompter, d llmprovider.Descriptor, cat catalog.Catalog, primary string, saved []string) ([]string, error) {
	var chosen []string
	for {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("select fallbacks: %w", err)
		}
		exclude := excludedIDs(primary, chosen)
		recs, usable := without(cat.Recommended, exclude), without(cat.Usable, exclude)
		extra := savedBeyond(saved, recs, exclude)
		if len(recs) == 0 && len(usable) == 0 && len(extra) == 0 {
			return chosen, nil
		}
		q, err := p.Input(searchFallbackPrompt, "")
		if err != nil {
			return nil, fmt.Errorf("search fallback models: %w", err)
		}
		q = strings.TrimSpace(q)
		if q == "" {
			if len(recs) == 0 && len(extra) == 0 {
				return chosen, nil
			}
			choices := modelChoices(d.ID, recs)
			for _, m := range extra {
				choices = append(choices, Choice{Label: catalog.Label(d.ID, m), Detail: savedFallbackDetail})
			}
			ids := append(slices.Clone(recs), extra...)
			idxs, err := p.MultiSelect(chooseFallbacksTitle, choices, indicesOf(ids, saved))
			if err != nil {
				return nil, fmt.Errorf("select fallbacks: %w", err)
			}
			chosen = appendPicks(chosen, ids, idxs)
			return chosen, nil // a blank round ends the loop
		}
		matches := searchCatalog(d, cat, exclude, q)
		if len(matches) == 0 {
			p.Notify(LevelWarn, "no %s models match %q", d.Label, q)
			continue
		}
		shown := capMatches(p, matches)
		ids := matchIDs(shown)
		idxs, err := p.MultiSelect(searchFallbacksTitle, matchChoices(shown, menuFacts(cat)), indicesOf(ids, saved))
		if err != nil {
			return nil, fmt.Errorf("select fallbacks: %w", err)
		}
		chosen = appendPicks(chosen, ids, idxs)
		more, err := p.Confirm(moreFallbacksPrompt, false)
		if err != nil {
			return nil, fmt.Errorf("confirm more fallbacks: %w", err)
		}
		if !more {
			return chosen, nil
		}
	}
}

// savedBeyond returns the saved fallbacks that are neither excluded nor among
// recs, once each, compared without case (0059-MADR W7).
func savedBeyond(saved, recs []string, exclude map[string]struct{}) []string {
	var out []string
	for _, s := range saved {
		same := func(m string) bool { return strings.EqualFold(m, s) }
		if _, skip := exclude[strings.ToLower(s)]; skip || strings.TrimSpace(s) == "" ||
			slices.ContainsFunc(recs, same) || slices.ContainsFunc(out, same) {
			continue
		}
		out = append(out, s)
	}
	return out
}

// excludedIDs is the set a fallback round must not offer, keyed by lower-case
// id because SearchModels compares ids case-insensitively (MADR 0013 C6).
func excludedIDs(primary string, chosen []string) map[string]struct{} {
	exclude := make(map[string]struct{}, len(chosen)+1)
	exclude[strings.ToLower(primary)] = struct{}{}
	for _, c := range chosen {
		exclude[strings.ToLower(c)] = struct{}{}
	}
	return exclude
}

// without returns a new slice of the models not in exclude, in order.
func without(models []string, exclude map[string]struct{}) []string {
	out := make([]string, 0, len(models))
	for _, m := range models {
		if _, skip := exclude[strings.ToLower(m)]; !skip {
			out = append(out, m)
		}
	}
	return out
}

// appendPicks appends ids[i] for each valid index, once each (MADR 0013 C1).
// chosen becomes non-nil even when nothing was picked, because a MultiSelect
// was shown.
func appendPicks(chosen, ids []string, idxs []int) []string {
	if chosen == nil {
		chosen = []string{}
	}
	for _, i := range idxs {
		if i >= 0 && i < len(ids) && !slices.ContainsFunc(chosen, func(c string) bool {
			return strings.EqualFold(c, ids[i])
		}) {
			chosen = append(chosen, ids[i])
		}
	}
	return chosen
}

// indicesOf returns the positions in models of the saved ids, compared
// case-insensitively (MADR 0013 C6); nil when none is there.
func indicesOf(models, saved []string) []int {
	var out []int
	for i, m := range models {
		if slices.ContainsFunc(saved, func(s string) bool { return strings.EqualFold(s, m) }) {
			out = append(out, i)
		}
	}
	return out
}

// capMatches keeps the first maxSearchResults matches and says so when it
// drops any. Only the fallbacks' MultiSelect cuts: a pick returns indices,
// not a page to turn to; the model menu pages (pageMatches).
func capMatches(p Prompter, matches []catalog.Match) []catalog.Match {
	if len(matches) <= maxSearchResults {
		return matches
	}
	p.Notify(LevelInfo, "showing %d of %d matches; refine the search to narrow them", maxSearchResults, len(matches))
	return matches[:maxSearchResults]
}

// matchChoices renders matches as menu rows, each with its facts and why it
// matched as its Detail.
func matchChoices(matches []catalog.Match, facts map[string]catalog.ModelInfo) []Choice {
	out := make([]Choice, 0, len(matches))
	for _, m := range matches {
		out = append(out, Choice{Label: m.Label, Detail: matchDetail(facts[strings.ToLower(m.ID)], m.Why)})
	}
	return out
}

// matchDetail is a search row's Detail, the parts known, in one order: the
// context window ("200k ctx"), the prices ("$3 / $15 per M", or "free"), the
// input it reads besides text and whether it takes tools and reasons
// ("vision · tools · reasoning"), then why it matched (0061-MADR, amendment
// "Model search, rebuilt", S4).
func matchDetail(info catalog.ModelInfo, why string) string {
	var parts []string
	if info.Has(catalog.FieldContextWindow) && info.ContextWindow > 0 {
		parts = append(parts, tokenCount(info.ContextWindow)+" ctx")
	}
	if info.Has(catalog.FieldCostInput) && info.Has(catalog.FieldCostOutput) {
		if info.Cost.Input == 0 && info.Cost.Output == 0 {
			parts = append(parts, "free")
		} else {
			parts = append(parts, fmt.Sprintf("$%s / $%s per M", price(info.Cost.Input), price(info.Cost.Output)))
		}
	}
	if info.Has(catalog.FieldInput) {
		for _, m := range info.Input {
			switch m {
			case catalog.ModalityImage:
				parts = append(parts, "vision")
			case catalog.ModalityAudio, catalog.ModalityVideo, catalog.ModalityPDF:
				parts = append(parts, string(m))
			}
		}
	}
	if info.Has(catalog.FieldTools) && info.Tools {
		parts = append(parts, "tools")
	}
	if info.Has(catalog.FieldReasoning) && info.Reasoning.Supported {
		parts = append(parts, "reasoning")
	}
	if why != "" {
		parts = append(parts, why)
	}
	return strings.Join(parts, " · ")
}

// tokenCount is n tokens in short: in binary units when n is a whole number
// of them, as windows such as 131072 (128k) are, else in decimal ones.
func tokenCount(n int) string {
	switch {
	case n%(1<<20) == 0:
		return strconv.Itoa(n>>20) + "m"
	case n%1_000_000 == 0:
		return strconv.Itoa(n/1_000_000) + "m"
	case n%1024 == 0:
		return strconv.Itoa(n/1024) + "k"
	case n >= 1000:
		return strconv.Itoa(n/1000) + "k"
	}
	return strconv.Itoa(n)
}

// price is a price per million tokens, in as few digits as it takes.
func price(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }

// matchIDs returns the ids of matches, in order.
func matchIDs(matches []catalog.Match) []string {
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, m.ID)
	}
	return out
}
