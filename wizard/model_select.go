package wizard

import (
	"context"
	"fmt"
	"slices"
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
	chooseModelTitle     = "Choose a %s model:"
	searchModelsPrompt   = "Search models (blank for recommended)"
	searchFallbackPrompt = "Search fallback models (blank for recommended)"
	chooseFallbacksTitle = "Choose fallback models (optional):"
	searchFallbacksTitle = "Choose fallback models:"
	moreFallbacksPrompt  = "Search for more fallback models?"
	maxSearchResults     = 20
)

// selectModel asks for a search query. A blank query shows the recommended
// menu; any other query searches every usable model and offers the numbered
// matches, Search again, and Other. A query with no match offers itself as
// the model id, or Search again. Each round checks ctx, so a prompter that
// accepts every default stops when ctx ends (R40; 0026-MADR F9).
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
		matches := catalog.Search(d.ID, cat.Usable, q)
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
		shown := capMatches(p, matches)
		choices := append(matchChoices(shown), Choice{Label: searchAgainLabel}, Choice{Label: otherModelLabel})
		idx, err := choose(p, title, choices, 0)
		if err != nil {
			return "", fmt.Errorf("select model: %w", err)
		}
		switch {
		case idx < len(shown):
			return shown[idx].ID, nil
		case idx == len(shown):
			continue
		default:
			return enterModelID(p, d.ID, o)
		}
	}
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
// saved fallbacks a round offers are preselected (0020-MADR F50). Each round
// checks ctx (0026-MADR F9).
func selectFallbacks(ctx context.Context, p Prompter, d llmprovider.Descriptor, cat catalog.Catalog, primary string, saved []string) ([]string, error) {
	var chosen []string
	for {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("select fallbacks: %w", err)
		}
		exclude := excludedIDs(primary, chosen)
		recs, usable := without(cat.Recommended, exclude), without(cat.Usable, exclude)
		if len(recs) == 0 && len(usable) == 0 {
			return chosen, nil
		}
		q, err := p.Input(searchFallbackPrompt, "")
		if err != nil {
			return nil, fmt.Errorf("search fallback models: %w", err)
		}
		q = strings.TrimSpace(q)
		if q == "" {
			if len(recs) == 0 {
				return chosen, nil
			}
			idxs, err := p.MultiSelect(chooseFallbacksTitle, modelChoices(d.ID, recs), indicesOf(recs, saved))
			if err != nil {
				return nil, fmt.Errorf("select fallbacks: %w", err)
			}
			chosen = appendPicks(chosen, recs, idxs)
			return chosen, nil // a blank round ends the loop
		}
		matches := catalog.Search(d.ID, usable, q)
		if len(matches) == 0 {
			p.Notify(LevelWarn, "no %s models match %q", d.Label, q)
			continue
		}
		shown := capMatches(p, matches)
		ids := matchIDs(shown)
		idxs, err := p.MultiSelect(searchFallbacksTitle, matchChoices(shown), indicesOf(ids, saved))
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
// drops any.
func capMatches(p Prompter, matches []catalog.Match) []catalog.Match {
	if len(matches) <= maxSearchResults {
		return matches
	}
	p.Notify(LevelInfo, "showing %d of %d matches; refine the search to narrow them", maxSearchResults, len(matches))
	return matches[:maxSearchResults]
}

// matchChoices renders matches as menu rows.
func matchChoices(matches []catalog.Match) []Choice {
	out := make([]Choice, 0, len(matches))
	for _, m := range matches {
		out = append(out, Choice{Label: m.Label})
	}
	return out
}

// matchIDs returns the ids of matches, in order.
func matchIDs(matches []catalog.Match) []string {
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		out = append(out, m.ID)
	}
	return out
}
