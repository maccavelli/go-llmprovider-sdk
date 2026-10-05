package catalog

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// countID reports how many times id appears in ids.
func countID(ids []string, id string) int {
	n := 0
	for _, v := range ids {
		if v == id {
			n++
		}
	}
	return n
}

// TestListModelCatalog_DuplicateIDs pins MADR 0013 A1: a listing that repeats
// an id yields it once in Usable and once in Recommended.
func TestListModelCatalog_DuplicateIDs(t *testing.T) {
	pinRankingNow(t, refNow)
	a := kiloRankEntry("a/pro", "A Pro", "0.000001", "0.000004", 20, true, "")
	b := kiloRankEntry("b/pro", "B Pro", "0.000002", "0.000004", 20, true, "")
	srv := serveBody(t, `{"data":[`+strings.Join([]string{a, a, b}, ",")+`]}`)
	cat := listCatalog(context.Background(), t, llmprovider.ProviderKilo, llmprovider.WithBaseURL(srv.URL))
	if n := countID(cat.Recommended, "a/pro"); n != 1 {
		t.Errorf("Recommended lists a/pro %d times: %v", n, cat.Recommended)
	}
	if n := countID(cat.Usable, "a/pro"); n != 1 {
		t.Errorf("Usable lists a/pro %d times: %v", n, cat.Usable)
	}
}

// TestRankRecommended_SkipsDuplicateCandidates pins MADR 0013 A1 at the
// ranker: two candidates with one id take one slot.
func TestRankRecommended_SkipsDuplicateCandidates(t *testing.T) {
	c := good("x/a", "x", 1, 10)
	got := rankRecommended(ProfileUtility, llmprovider.ProviderKilo, []rankCandidate{c, c, good("y/b", "y", 2, 10)}, nil)
	assertRanked(t, got, []string{"x/a", "y/b"})
}

// kiloPricedEntry is one reasoning Kilo entry created ten days before refNow;
// pricing is a JSON member (with its leading comma) or "".
func kiloPricedEntry(t *testing.T, id, pricing string) kiloCatalogEntry {
	t.Helper()
	return decodeKiloEntry(t, fmt.Sprintf(`{"id":%q,"name":%q,"created":%d,"context_length":200000,`+
		`"supported_parameters":["tools","reasoning"]%s}`, id, id, refNow.AddDate(0, 0, -10).Unix(), pricing))
}

// TestKiloCandidate_AbsentPriceIsUnknown pins MADR 0013 A2 (MADR 0009 §3: an
// absent field never excludes). An explicit "0" is still free and excluded.
func TestKiloCandidate_AbsentPriceIsUnknown(t *testing.T) {
	absent := kiloCandidate(kiloPricedEntry(t, "x/y", ""), refNow)
	if absent.costKnown || !absent.eligible(llmprovider.ProviderKilo) {
		t.Errorf("no pricing block: costKnown=%v eligible=%v, want unknown and eligible",
			absent.costKnown, absent.eligible(llmprovider.ProviderKilo))
	}
	free := kiloCandidate(kiloPricedEntry(t, "x/free", `,"pricing":{"prompt":"0","completion":"0"}`), refNow)
	if !free.costKnown || free.eligible(llmprovider.ProviderKilo) {
		t.Errorf(`pricing "0": costKnown=%v eligible=%v, want known and excluded`,
			free.costKnown, free.eligible(llmprovider.ProviderKilo))
	}
}

// TestKiloPrice_NonFiniteIsUnknown pins MADR 0013 A2–A3: blank and
// non-finite prices are unknown; finite non-negative prices parse.
func TestKiloPrice_NonFiniteIsUnknown(t *testing.T) {
	for _, s := range []string{"", "  ", "Infinity", "+Inf", "inf", "NaN", "-1", "abc"} {
		if v, ok := kiloPrice(s); ok {
			t.Errorf("kiloPrice(%q) = %v, known; want unknown", s, v)
		}
	}
	for s, want := range map[string]float64{"0": 0, "0.000001": 0.000001, " 2e-6 ": 2e-6} {
		if v, ok := kiloPrice(s); !ok || v != want {
			t.Errorf("kiloPrice(%q) = %v/%v, want %v/true", s, v, ok, want)
		}
	}
}

// TestRankRecommended_InfinitePriceNotCheapest pins MADR 0013 A3: before the
// fix an "Infinity" price made the blend NaN, which cmp.Compare orders first.
func TestRankRecommended_InfinitePriceNotCheapest(t *testing.T) {
	mk := func(id, price string) rankCandidate {
		return kiloCandidate(kiloPricedEntry(t, id,
			fmt.Sprintf(`,"pricing":{"prompt":%q,"completion":%q}`, price, price)), refNow)
	}
	got := rankRecommended(ProfileUtility, llmprovider.ProviderKilo,
		[]rankCandidate{mk("a/cheap", "0.0000001"), mk("b/mid", "0.000001"), mk("z/infinite", "Infinity")}, nil)
	assertRanked(t, got, []string{"a/cheap", "b/mid", "z/infinite"})
}

// guardCandidates are metadata candidates of each kind C9 guards: an unusable
// cost, a future release date, and ordinary ones.
func guardCandidates() map[string]rankCandidate {
	reasoning := true
	entry := func(input, output float64, date string) modelMetadata {
		m := modelMetadata{Reasoning: &reasoning, Cost: &modelCost{Input: input, Output: output}, ReleaseDate: date}
		m.Limit.Context = 131072
		return m
	}
	future := refNow.AddDate(0, 2, 0).Format(time.DateOnly)
	cands := map[string]rankCandidate{}
	for id, m := range map[string]modelMetadata{
		"negative": entry(-1, 0.5, "2026-08-01"),
		"nan":      entry(math.NaN(), 1, "2026-08-01"),
		"inf":      entry(math.Inf(1), 1, "2026-08-01"),
		"overflow": entry(1e308, 1e308, "2026-08-01"),
		"future":   entry(0.2, 0.6, future),
		"cheap":    entry(0.1, 0.2, "2026-08-01"),
		"dear":     entry(3, 15, "2026-06-01"),
		"old":      entry(0.1, 0.2, "2025-01-01"),
	} {
		cands[id] = metadataCandidate(id, m, true, refNow)
	}
	return cands
}

// TestMetadataCandidate_GuardsCostAndAge (0021-MADR C9): a cost that is
// negative, NaN, infinite, or overflows when summed is unknown, and so is the
// age of a model released after now.
func TestMetadataCandidate_GuardsCostAndAge(t *testing.T) {
	cands := guardCandidates()
	for _, id := range []string{"negative", "nan", "inf", "overflow"} {
		if c := cands[id]; c.costKnown {
			t.Errorf("%s: cost %v is known, want unknown", id, c.cost)
		}
	}
	if c := cands["future"]; c.ageKnown {
		t.Errorf("a future release date gives age %d days, want unknown", c.ageDays)
	}
	if c := cands["cheap"]; !c.costKnown || !c.ageKnown {
		t.Errorf("an ordinary entry: cost known %v, age known %v; want both", c.costKnown, c.ageKnown)
	}
}

// TestKiloPriceRank_NaNIsUnknown (0021-MADR C9): a NaN price ranks last, as
// an unknown one does.
func TestKiloPriceRank_NaNIsUnknown(t *testing.T) {
	for _, s := range []string{"NaN", "Infinity", "-1", "x"} {
		if got := kiloPriceRank(s); got != math.MaxFloat64 {
			t.Errorf("kiloPriceRank(%q) = %v, want math.MaxFloat64", s, got)
		}
	}
	if got := kiloPriceRank("0.000002"); got != 0.000002 {
		t.Errorf("kiloPriceRank of a price = %v, want it", got)
	}
}

// TestRankOrders_StrictWeakOrder (0021-MADR C9): both comparators are strict
// weak orders over candidates of every kind, checked pairwise and over
// triples.
func TestRankOrders_StrictWeakOrder(t *testing.T) {
	var cands []rankCandidate
	for _, c := range guardCandidates() {
		cands = append(cands, c)
	}
	norm := newRankNorm(cands)
	for name, order := range map[string]func(a, b rankCandidate) int{
		"utility": norm.utilityOrder, "capable": norm.capableOrder,
	} {
		for _, a := range cands {
			if order(a, a) != 0 {
				t.Errorf("%s: %s is not equal to itself", name, a.id)
			}
			for _, b := range cands {
				if sign(order(a, b)) != -sign(order(b, a)) {
					t.Errorf("%s: %s and %s are not antisymmetric", name, a.id, b.id)
				}
				for _, c := range cands {
					if order(a, b) < 0 && order(b, c) < 0 && order(a, c) >= 0 {
						t.Errorf("%s: %s < %s < %s, but not %s < %s", name, a.id, b.id, c.id, a.id, c.id)
					}
				}
			}
		}
	}
}

func sign(n int) int { return cmp.Compare(n, 0) }
