package catalog

import (
	"cmp"
	"slices"
	"strings"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// modelVersion is the version parseVersion reads from a model id: the family
// name before it, its major, and its minor with the minor's digit count, 0
// when the id names no minor (0070-MADR D1).
type modelVersion struct {
	family               string
	major, minor, digits int
}

// versionFamily is one family's versions of one major, such as grok 4 or
// gemini 3.
type versionFamily struct {
	name  string
	major int
}

// parseVersion reads the first version after an id's family name: N, N.M or
// N-M, where N is one digit and M one or two, so a date or a size is never a
// version (0021-MADR C6).
func parseVersion(id string) (modelVersion, bool) {
	tokens := modelTokens(id)
	for i := 1; i < len(tokens); i++ {
		n, isVersion := versionNumber(tokens[i])
		if !isVersion || len(tokens[i]) > 1 {
			continue
		}
		v := modelVersion{family: strings.Join(tokens[:i], "-"), major: n}
		if i+1 < len(tokens) {
			if m, isMinor := versionNumber(tokens[i+1]); isMinor {
				v.minor, v.digits = m, len(tokens[i+1])
			}
		}
		return v, true
	}
	return modelVersion{}, false
}

// versionKey is v's order key, in tenths of a minor: 1000·N + 10·M, or, for
// a two-digit M read as a decimal, 1000·N + M. So 4.20 read as a decimal is
// 4.2, between 4.1 and 4.3, and 3.10 read as an integer follows 3.9
// (0070-MADR D1). A one-digit minor has the same key under both readings.
func versionKey(v modelVersion, decimal bool) int {
	if decimal && v.digits == 2 {
		return 1000*v.major + v.minor
	}
	return 1000*v.major + 10*v.minor
}

// versionSlot is v's place for the gap test: its minor, or the whole tenths
// of a two-digit minor read as a decimal.
func versionSlot(v modelVersion, decimal bool) int {
	if decimal && v.digits == 2 {
		return v.minor / 10
	}
	return v.minor
}

// versionReadings maps each family that holds a two-digit minor to its
// reading: true reads the minor as a decimal, false as an integer. A family
// with no two-digit minor is not in it, and reads as an integer.
type versionReadings map[versionFamily]bool

// decimal reports whether id's family reads its minor as a decimal.
func (r versionReadings) decimal(id string) bool {
	v, ok := parseVersion(id)
	return ok && r[versionFamily{v.family, v.major}]
}

// versionMember is one id of a family, with its release date, zero when it
// has none.
type versionMember struct {
	v        modelVersion
	released time.Time
}

// chooseReadings chooses the reading of each family in ids that holds a
// two-digit minor, from released where it dates the ids (0070-MADR D1):
//
//  1. the dates: of the pairs of dated ids that the two readings order
//     differently, the reading that puts fewer against their dates;
//  2. otherwise the gaps: the reading whose versions fill more of the slots
//     from the lowest to the highest;
//  3. otherwise the integer reading (0021-MADR C6).
//
// A pair both readings order alike adds to both counts or to neither, so it
// cannot decide: a later snapshot of an older generation is not evidence.
func chooseReadings(ids []string, released map[string]time.Time) versionReadings {
	families := map[versionFamily][]versionMember{}
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		v, ok := parseVersion(id)
		if !ok {
			continue
		}
		f := versionFamily{v.family, v.major}
		families[f] = append(families[f], versionMember{v, released[id]})
	}
	out := versionReadings{}
	for f, members := range families {
		if slices.ContainsFunc(members, func(m versionMember) bool { return m.v.digits == 2 }) {
			out[f] = readsDecimal(members)
		}
	}
	return out
}

// readsDecimal is chooseReadings' choice for one family's members.
func readsDecimal(members []versionMember) bool {
	if integer, decimal := againstDates(members); integer != decimal {
		return decimal < integer
	}
	filledInt, spanInt := gapShare(members, false)
	filledDec, spanDec := gapShare(members, true)
	return filledDec*spanInt > filledInt*spanDec
}

// againstDates counts, for each reading, the pairs of dated members that the
// readings order differently and that it orders against their dates.
func againstDates(members []versionMember) (integer, decimal int) {
	for i, a := range members {
		for _, b := range members[i+1:] {
			if a.released.IsZero() || b.released.IsZero() || a.released.Equal(b.released) {
				continue
			}
			ai, bi := versionKey(a.v, false), versionKey(b.v, false)
			ad, bd := versionKey(a.v, true), versionKey(b.v, true)
			if cmp.Compare(ai, bi) == cmp.Compare(ad, bd) {
				continue
			}
			earlier := a.released.Before(b.released)
			if ai != bi && (ai < bi) != earlier {
				integer++
			}
			if ad != bd && (ad < bd) != earlier {
				decimal++
			}
		}
	}
	return integer, decimal
}

// gapShare is the number of slots a reading's versions fill, and the span of
// slots from the lowest to the highest.
func gapShare(members []versionMember, decimal bool) (filled, span int) {
	slots := map[int]bool{}
	lo, hi := 0, 0
	for i, m := range members {
		s := versionSlot(m.v, decimal)
		slots[s] = true
		if i == 0 || s < lo {
			lo = s
		}
		if i == 0 || s > hi {
			hi = s
		}
	}
	return len(slots), hi - lo + 1
}

// versionRanker is a provider's ranker that reads versions with
// generationScore, taking the reading of the id's family, or nil for a
// provider whose ranker reads none. provider is lower case.
func versionRanker(provider llmprovider.ProviderID) func(m string, decimal bool) int {
	switch provider {
	case llmprovider.ProviderGemini:
		return rankGeminiWith
	case llmprovider.ProviderClaude:
		return rankClaudeWith
	case llmprovider.ProviderGrok:
		return rankGrokWith
	}
	return nil
}

// staticReading is model's family reading over the provider's static
// catalog and model, with no dates: the reading Rank uses.
func staticReading(provider llmprovider.ProviderID, model string) bool {
	return chooseReadings(append(Static(provider), model), nil).decimal(model)
}

// rankerFor is provider's ranker, with each family's reading chosen over
// ids and the provider's static catalog, with released's dates.
func rankerFor(provider llmprovider.ProviderID, ids []string, released map[string]time.Time) func(string) int {
	p := llmprovider.ProviderID(strings.ToLower(string(provider)))
	rank := versionRanker(p)
	if rank == nil {
		return func(m string) int { return Rank(p, m) }
	}
	readings := chooseReadings(slices.Concat(ids, Static(p)), released)
	return func(m string) int { return rank(m, readings.decimal(m)) }
}

// SortByRank returns ids sorted by Rank's score, highest first, keeping
// their order among equal scores. Where Rank sees one id with the provider's
// static catalog, SortByRank reads each model family's version numbers from
// the whole list, and from released, the release dates the caller has. A
// family's two-digit minor reads as an integer, so gemini-3.10 follows
// gemini-3.9, or as a decimal, so grok-4.20 is 4.2, below grok-4.3: its
// dates choose, else the gaps between its versions, else the integer
// reading (0070-MADR D1).
//
// released may be nil; a Catalog's MenuEntry.Info.Released holds a
// listing's dates. The provider id is compared case-insensitively, as Rank
// compares it. SortByRank returns a new slice, empty for no ids, and does not
// change ids. It keeps no state, so it is safe for concurrent use.
func SortByRank(provider llmprovider.ProviderID, ids []string, released map[string]time.Time) []string {
	rank := rankerFor(provider, ids, released)
	scores := make(map[string]int, len(ids))
	for _, id := range ids {
		scores[id] = rank(id)
	}
	out := make([]string, len(ids))
	copy(out, ids)
	slices.SortStableFunc(out, func(a, b string) int { return cmp.Compare(scores[b], scores[a]) })
	return out
}
