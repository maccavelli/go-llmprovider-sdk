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
// any other query is scored by a tiered fuzzy match. Search ranks; it
// does not filter for usability, which is the listing's job.

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

// Match is one ranked Search result.
type Match struct {
	ID    string
	Label string // Label(provider, ID)
	Score int    // higher is better; comparable only within one call
}

// Search ranks models against query, highest score first; equal scores
// keep the input order, the listing's own (0021-MADR C11). Glob queries (*
// matches any run, including "/", and ? one character) score an id match
// above a match of the label only. Other queries match the id exactly, as a
// prefix or substring, then with separators and spaces removed, then the
// label's display name (the label before its first "["), then by token
// prefix. Only when no id matches any of those, and the query has three or
// more characters, does a subsequence of the id match. An empty query returns
// nil. Duplicate ids (compared case-insensitively) are searched once.
func Search(provider llmprovider.ProviderID, models []string, query string) []Match {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}
	candidates := uniqueModelMatches(provider, models)
	if strings.ContainsAny(q, "*?") {
		return sortByScore(globMatches(candidates, q))
	}
	var out []Match
	for _, c := range candidates {
		if score, ok := fuzzyModelScore(c.ID, c.Label, q); ok {
			c.Score = score
			out = append(out, c)
		}
	}
	if len(out) == 0 && len([]rune(q)) >= minSubsequenceQuery {
		for _, c := range candidates {
			if bonus, ok := subsequenceBonus(strings.ToLower(c.ID), q); ok {
				c.Score = scoreSubsequence + bonus
				out = append(out, c)
			}
		}
	}
	return sortByScore(out)
}

// sortByScore orders matches by score, highest first, keeping input order
// among equal scores.
func sortByScore(out []Match) []Match {
	slices.SortStableFunc(out, func(a, b Match) int { return cmp.Compare(b.Score, a.Score) })
	return out
}

// uniqueModelMatches labels each id once, in input order.
func uniqueModelMatches(provider llmprovider.ProviderID, models []string) []Match {
	seen := make(map[string]struct{}, len(models))
	out := make([]Match, 0, len(models))
	for _, id := range models {
		key := strings.ToLower(id)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, Match{ID: id, Label: Label(provider, id)})
	}
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
			c.Score = scoreGlobID
		case re.MatchString(c.Label):
			c.Score = scoreGlobLabel
		default:
			continue
		}
		out = append(out, c)
	}
	return out
}

// fuzzyModelScore returns the tier of the strongest predicate q satisfies
// above the subsequence tier, or false when it satisfies none. The label
// tiers read the display name only: the annotation in brackets would match
// a short query almost always (0021-MADR C11).
func fuzzyModelScore(id, label, q string) (int, bool) {
	lowerID := strings.ToLower(id)
	name := displayName(strings.ToLower(label))
	compactQ := compactModelText(q)
	switch {
	case lowerID == q:
		return scoreExact, true
	case strings.HasPrefix(lowerID, q):
		return scoreIDPrefix, true
	case strings.Contains(lowerID, q):
		return scoreIDSubstring, true
	case compactQ != "" && strings.Contains(compactModelText(lowerID), compactQ):
		return scoreCompactSubstring, true
	case strings.Contains(name, q):
		return scoreLabelSubstring, true
	case tokenPrefixMatch(q, lowerID, name):
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

// tokenPrefixMatch reports whether every token of q prefixes some id or label
// token. A query with no tokens never matches.
func tokenPrefixMatch(q, lowerID, lowerLabel string) bool {
	queryTokens := modelTokens(q)
	if len(queryTokens) == 0 {
		return false
	}
	surface := slices.Concat(modelTokens(lowerID), modelTokens(lowerLabel))
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
