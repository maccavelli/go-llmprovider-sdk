package catalog

import (
	"cmp"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Model search (MADR 0007 §3). A query containing * or ? takes the glob path;
// any other query is scored by a tiered fuzzy match, ranked inside a tier by
// the engine in search.go (0061-MADR, amendment "Model search, rebuilt").
// Search ranks; it does not filter for usability, which is the listing's
// job.

// Fuzzy score tiers: a stronger kind of match always outranks a weaker one.
const (
	scoreExact            = 6000
	scoreIDPrefix         = 5000
	scoreIDSubstring      = 4000
	scoreCompactSubstring = 3500
	scoreLabelSubstring   = 3000
	scoreTokenPrefix      = 2000
	scoreSubsequence      = 1000
	scoreGlobID           = 2
	scoreGlobLabel        = 1
	maxSubsequenceBonus   = 999
	// minSubsequenceQuery is the shortest query the subsequence tier tries.
	minSubsequenceQuery = 3
)

// Match is one ranked Search result. It is safe for concurrent use only while
// no goroutine changes it.
type Match struct {
	ID    string
	Label string // Label(provider, ID)
	Score int    // higher is better; comparable only within one call
	// Why is a short reason for the match, for a picker to show: "exact
	// id", "id prefix", "every word: vision", "near sonnet" (0061-MADR,
	// amendment "Model search, rebuilt", S1).
	Why string
}

// Search ranks models against query, highest score first. Glob queries (*
// matches any run, including "/", and ? one character) score an id match
// above a match of the label only, equal scores in the input order. Other
// queries match, in tiers, the id exactly, as a prefix or substring, then
// with separators and spaces removed, then the label's display name (the
// label before its first "["), then by token prefix, then with every query
// word an id or label word or a prefix of one, words split also between a
// letter and a digit, a size such as 8b kept whole. Only when no model
// matches any of those is each next fallback tried, in turn: every word
// matched, one at least within an edit distance (one edit for a word of four
// to seven runes, two from eight, an adjacent transposition one); some
// words, more first; and, for a query of three or more characters, a
// subsequence of the id. Inside a tier the order is BM25F's over the id and
// the label, then fzf's score, then the input order, Catalog.Usable's: the
// listing's own for most providers, cheapest first for Kilo, fastest first
// for Hugging Face (0021-MADR C11; 0026-MADR F54; 0061-MADR, amendment
// "Model search, rebuilt", S2). Each match's Why says how it matched. An
// empty query returns nil. Duplicate ids (compared case-insensitively) are
// searched once. Catalog.Search searches a listing's facts as well.
func Search(provider llmprovider.ProviderID, models []string, query string) []Match {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}
	return newSearchIndex(provider, len(models), func(i int) (string, *ModelInfo) { return models[i], nil }).query(q)
}

// sortByScore orders matches by score, highest first, keeping input order
// among equal scores.
func sortByScore(out []Match) []Match {
	slices.SortStableFunc(out, func(a, b Match) int { return cmp.Compare(b.Score, a.Score) })
	return out
}

// globMatches keeps, in input order, each candidate whose whole id or whole
// label matches the glob q, scoring an id match above a label-only one.
// Unlike path.Match, * crosses "/".
func globMatches(candidates []Match, q string) []Match {
	var b strings.Builder
	for _, r := range q {
		switch r {
		case '*':
			b.WriteString(".*")
		case '?':
			b.WriteString(".")
		default:
			b.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	pattern := "(?i)^" + b.String() + "$"
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil // unreachable: every literal rune is quoted
	}
	var out []Match
	for _, c := range candidates {
		switch {
		case re.MatchString(c.ID):
			c.Score, c.Why = scoreGlobID, "id pattern"
		case re.MatchString(c.Label):
			c.Score, c.Why = scoreGlobLabel, "name pattern"
		default:
			continue
		}
		out = append(out, c)
	}
	return out
}

// tierScore returns the tier of the strongest predicate the query satisfies
// above scoreAllWords, or false when it satisfies none. The label tiers read
// the display name only: the annotation in brackets would match a short
// query almost always (0021-MADR C11).
func tierScore(d *searchDoc, sq *searchQuery) (int, bool) {
	switch {
	case d.lowerID == sq.q:
		return scoreExact, true
	case strings.HasPrefix(d.lowerID, sq.q):
		return scoreIDPrefix, true
	case strings.Contains(d.lowerID, sq.q):
		return scoreIDSubstring, true
	case sq.compact != "" && strings.Contains(d.compact, sq.compact):
		return scoreCompactSubstring, true
	case strings.Contains(d.name, sq.q):
		return scoreLabelSubstring, true
	case tokensPrefixed(sq.tokens, d.tokens):
		return scoreTokenPrefix, true
	}
	return 0, false
}

// displayName is a label's name, before any bracketed annotation.
func displayName(label string) string {
	name, _, _ := strings.Cut(label, "[")
	return strings.TrimSpace(name)
}

// compactModelText removes the separators "-", ".", "_", "/" and spaces, so
// "gpt41" finds gpt-4.1.
func compactModelText(s string) string {
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune("-._/", r) || unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
}

// isModelSeparator reports whether r splits model-id tokens.
func isModelSeparator(r rune) bool {
	return strings.ContainsRune("-_/.:~", r) || unicode.IsSpace(r)
}

// modelTokens splits s into lowercase tokens on isModelSeparator.
func modelTokens(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), isModelSeparator)
}

// tokensPrefixed reports whether every query token prefixes some id or label
// token. A query with no tokens never matches.
func tokensPrefixed(queryTokens, surface []string) bool {
	if len(queryTokens) == 0 {
		return false
	}
	for _, qt := range queryTokens {
		if !slices.ContainsFunc(surface, func(t string) bool { return strings.HasPrefix(t, qt) }) {
			return false
		}
	}
	return true
}

// subsequenceBonus matches the non-separator runes of q, in order, against
// lowerID (greedy leftmost). Each matched rune earns 10 when it directly
// follows the previous match and 5 at the start of a token.
func subsequenceBonus(lowerID, q string) (int, bool) {
	var needle []rune
	for _, r := range q {
		if !isModelSeparator(r) {
			needle = append(needle, r)
		}
	}
	if len(needle) == 0 {
		return 0, false
	}
	hay := []rune(lowerID)
	j, prev, bonus := 0, -2, 0
	for i, r := range hay {
		if j == len(needle) {
			break
		}
		if r != needle[j] {
			continue
		}
		if i == prev+1 {
			bonus += 10
		}
		if i == 0 || isModelSeparator(hay[i-1]) {
			bonus += 5
		}
		prev = i
		j++
	}
	if j < len(needle) {
		return 0, false
	}
	return min(bonus, maxSubsequenceBonus), true
}
