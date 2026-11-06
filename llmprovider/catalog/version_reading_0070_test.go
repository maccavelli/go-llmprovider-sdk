package catalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// day parses a YYYY-MM-DD release date for a test.
func day(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.Parse(time.DateOnly, s)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// datesOf turns id: YYYY-MM-DD pairs into released dates.
func datesOf(t *testing.T, pairs map[string]string) map[string]time.Time {
	t.Helper()
	if pairs == nil {
		return nil
	}
	out := make(map[string]time.Time, len(pairs))
	for id, s := range pairs {
		out[id] = day(t, s)
	}
	return out
}

// TestVersionReading_0070 (0070-MADR D1, D3; its amendment "the date test's
// pair rule is an identity"): the reading chooseReadings gives each family,
// from its dates, else its gaps, else the integer rule. The undated leap from
// N.5 to N.12, which no rule reading strings can read, is not a case.
func TestVersionReading_0070(t *testing.T) {
	const integer, decimal, absent = "integer", "decimal", "absent"
	for _, c := range []struct {
		name   string
		ids    []string
		dates  map[string]string
		family versionFamily
		want   string
	}{
		{"Gemini 3.5 to 3.10", []string{"gemini-3.5", "gemini-3.6", "gemini-3.7", "gemini-3.8", "gemini-3.9", "gemini-3.10"}, nil,
			versionFamily{"gemini", 3}, integer},
		{"Gemini's static ids with 3.10", []string{"gemini-3.5-flash-lite", "gemini-3.8-flash", "gemini-3.6-flash", "gemini-3.10-flash"}, nil,
			versionFamily{"gemini", 3}, integer},
		{"3.5 to 3.8 with 3.10", []string{"gemini-3.5", "gemini-3.6", "gemini-3.7", "gemini-3.8", "gemini-3.10"}, nil,
			versionFamily{"gemini", 3}, integer},
		{"3.9 with 3.10", []string{"gemini-3.9", "gemini-3.10"}, nil, versionFamily{"gemini", 3}, integer},
		{"Python-style 3.8 to 3.14", []string{"py-3.8", "py-3.9", "py-3.10", "py-3.11", "py-3.12", "py-3.13", "py-3.14"}, nil,
			versionFamily{"py", 3}, integer},
		{"Grok's static ids with 4.20", []string{"grok-4.6", "grok-4.5", "grok-4.7", "grok-4.3", "grok-build-0.1", "grok-4.20-0309-reasoning"}, nil,
			versionFamily{"grok", 4}, decimal},
		{"4.1 with 4.20", []string{"grok-4.1", "grok-4.20"}, nil, versionFamily{"grok", 4}, decimal},
		{"Grok with xAI's dates", []string{"grok-4.20", "grok-4.3", "grok-4.7"},
			map[string]string{"grok-4.20": "2026-03-09", "grok-4.3": "2026-04-17", "grok-4.7": "2026-09-02"},
			versionFamily{"grok", 4}, decimal},
		{"Llama 3.05", []string{"llama-3.05", "llama-3.1", "llama-3.2", "llama-3.3"}, nil, versionFamily{"llama", 3}, decimal},
		{"the dated leap", []string{"x-3.5", "x-3.12"}, map[string]string{"x-3.5": "2026-01-10", "x-3.12": "2026-06-10"},
			versionFamily{"x", 3}, integer},
		{"a lone 4.10", []string{"grok-4.10"}, nil, versionFamily{"grok", 4}, integer},
		{"a lone 4.20", []string{"grok-4.20"}, nil, versionFamily{"grok", 4}, integer},
		{"no two-digit minor", []string{"gemini-3.5", "gemini-3.6"}, nil, versionFamily{"gemini", 3}, absent},
		{"a late 4.1 does not decide", []string{"grok-4.20", "grok-4.3", "grok-4.1"},
			map[string]string{"grok-4.20": "2026-03-09", "grok-4.3": "2026-04-17", "grok-4.1": "2026-12-01"},
			versionFamily{"grok", 4}, decimal},
	} {
		readings := chooseReadings(c.ids, datesOf(t, c.dates))
		reading, in := readings[c.family]
		got := absent
		switch {
		case in && reading:
			got = decimal
		case in:
			got = integer
		}
		if got != c.want {
			t.Errorf("%s: %s reads as %s, want %s", c.name, c.family.name, got, c.want)
		}
	}
}

// guardFamily is one dated family the guard checks: its source, the module's
// provider for it (empty for a provider the module does not rank), and its
// ids with their dates.
type guardFamily struct {
	source   string
	provider llmprovider.ProviderID
	ids      []string
	released map[string]time.Time
}

// guardFamilies are the dated families of the three recorded sources that
// hold a two-digit minor (0070-MADR D3): the fixture of the live models.dev
// document, the recorded models.dev document, and xAI's and Anthropic's
// recorded listings. A fixture id is read by its name after the last "/", as
// the fixture grouped it (0070-PLAN Phase 1).
func guardFamilies(t *testing.T) []guardFamily {
	t.Helper()
	var out []guardFamily
	var fixture struct {
		Families []struct {
			Provider string `json:"provider"`
			Models   []struct {
				ID          string `json:"id"`
				ReleaseDate string `json:"release_date"`
			} `json:"models"`
		} `json:"families"`
	}
	readJSON(t, filepath.Join("testdata", "versions-2026-11-06", "families.json"), &fixture)
	if len(fixture.Families) != 22 {
		t.Fatalf("the fixture holds %d families, want 22 (0070-MADR Context §2)", len(fixture.Families))
	}
	for _, f := range fixture.Families {
		g := guardFamily{source: "fixture/" + f.Provider, provider: moduleProvider(f.Provider), released: map[string]time.Time{}}
		for _, m := range f.Models {
			name := m.ID[strings.LastIndex(m.ID, "/")+1:]
			g.ids = append(g.ids, name)
			g.released[name] = day(t, m.ReleaseDate)
		}
		out = append(out, g)
	}
	var meta map[string]struct {
		Models map[string]struct {
			ReleaseDate string `json:"release_date"`
		} `json:"models"`
	}
	readJSON(t, filepath.Join("testdata", "ranking-2026-10-12", "metadata.json"), &meta)
	for _, key := range []string{"google", "anthropic", "xai"} {
		dated := map[string]time.Time{}
		for id, m := range meta[key].Models {
			if d, err := time.Parse(time.DateOnly, m.ReleaseDate); err == nil {
				dated[id] = d
			}
		}
		out = append(out, splitFamilies("models.dev/"+key, moduleProvider(key), dated)...)
	}
	for _, src := range []struct {
		file     string
		provider llmprovider.ProviderID
	}{{"grok.json", llmprovider.ProviderGrok}, {"claude.json", llmprovider.ProviderClaude}} {
		var page struct {
			Data []struct {
				ID        string `json:"id"`
				Created   int64  `json:"created"`
				CreatedAt string `json:"created_at"`
			} `json:"data"`
		}
		if err := json.Unmarshal([]byte(firstPartyPage(t, src.file)), &page); err != nil {
			t.Fatal(err)
		}
		dated := map[string]time.Time{}
		for _, m := range page.Data {
			if m.Created > 0 {
				dated[m.ID] = time.Unix(m.Created, 0).UTC()
			} else if d, err := time.Parse(time.RFC3339, m.CreatedAt); err == nil {
				dated[m.ID] = d
			}
		}
		out = append(out, splitFamilies("listing/"+src.file, src.provider, dated)...)
	}
	return out
}

// readJSON decodes a recorded file.
func readJSON(t *testing.T, path string, v any) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, v); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
}

// moduleProvider is the module's provider for a models.dev provider key, or
// empty for one the module does not rank by version.
func moduleProvider(key string) llmprovider.ProviderID {
	switch key {
	case "xai":
		return llmprovider.ProviderGrok
	case "google":
		return llmprovider.ProviderGemini
	case "anthropic":
		return llmprovider.ProviderClaude
	}
	return ""
}

// splitFamilies groups dated ids into their families that hold a two-digit
// minor.
func splitFamilies(source string, provider llmprovider.ProviderID, dated map[string]time.Time) []guardFamily {
	byFamily := map[versionFamily]*guardFamily{}
	ids := slices.Sorted(func(yield func(string) bool) {
		for id := range dated {
			if !yield(id) {
				return
			}
		}
	})
	for _, id := range ids {
		v, ok := parseVersion(id)
		if !ok {
			continue
		}
		f := versionFamily{v.family, v.major}
		if byFamily[f] == nil {
			byFamily[f] = &guardFamily{source: source, provider: provider, released: map[string]time.Time{}}
		}
		byFamily[f].ids = append(byFamily[f].ids, id)
		byFamily[f].released[id] = dated[id]
	}
	var out []guardFamily
	for _, g := range byFamily {
		if slices.ContainsFunc(g.ids, func(id string) bool { v, _ := parseVersion(id); return v.digits == 2 }) {
			out = append(out, *g)
		}
	}
	return out
}

// TestVersionReading_Guard_0070 (0070-MADR D3): on every recorded dated
// family that holds a two-digit minor and whose dates decide, the reading the
// gap test gives with no dates is the one the dates choose, and Rank orders
// each pair the readings order differently as the dates do. A refresh of the
// recorded data that holds a naming the rule cannot read fails here.
func TestVersionReading_Guard_0070(t *testing.T) {
	decided, fixtureDecided := 0, 0
	for _, g := range guardFamilies(t) {
		v, _ := parseVersion(g.ids[0])
		family := versionFamily{v.family, v.major}
		members := make([]versionMember, 0, len(g.ids))
		for _, id := range g.ids {
			mv, _ := parseVersion(id)
			members = append(members, versionMember{mv, g.released[id]})
		}
		integer, decimal := againstDates(members)
		if integer == decimal {
			continue
		}
		decided++
		if strings.HasPrefix(g.source, "fixture/") {
			fixtureDecided++
		}
		byDates := decimal < integer
		offline := g.ids
		if g.provider != "" {
			offline = slices.Concat(g.ids, Static(g.provider))
		}
		if byGaps := chooseReadings(offline, nil)[family]; byGaps != byDates {
			t.Errorf("%s %s %d: the gap test reads decimal=%t, the dates decimal=%t", g.source, family.name, family.major, byGaps, byDates)
		}
		if g.provider == "" {
			continue
		}
		for _, a := range g.ids {
			for _, b := range g.ids {
				va, _ := parseVersion(a)
				vb, _ := parseVersion(b)
				alike := versionKey(va, false) < versionKey(vb, false) == (versionKey(va, true) < versionKey(vb, true))
				if alike || !g.released[a].Before(g.released[b]) {
					continue
				}
				if ra, rb := Rank(g.provider, a), Rank(g.provider, b); ra > rb {
					t.Errorf("%s: %s, released first, ranks %d, above %s at %d", g.source, a, ra, b, rb)
				}
			}
		}
	}
	// 20 fixture families, the recorded models.dev document's xAI Grok, and
	// xAI's recorded listing (0070-PLAN Phase 1).
	if fixtureDecided != 20 || decided != 22 {
		t.Errorf("the dates decide %d families, %d of them the fixture's; want 22 and 20", decided, fixtureDecided)
	}
}

// magicGrokIDs are the 14 ids a keyed Grok CLI 1.0.70 session listed, in its
// order (0070-MADR C46).
var magicGrokIDs = []string{
	"grok-4.20-0309-non-reasoning", "grok-4.20-0309-reasoning", "grok-4.20-multi-agent-0309", "grok-4.3", "grok-4.5",
	"grok-4.6", "grok-4.7", "grok-build-0.1", "grok-imagine-image", "grok-imagine-image-2.0", "grok-imagine-image-quality",
	"grok-imagine-video", "grok-imagine-video-1.5", "grok-imagine-video-1.5-lite",
}

// TestSortByRank_0070 (0070-MADR D1, D3): SortByRank orders a whole list,
// with each family read from the list and the caller's dates.
func TestSortByRank_0070(t *testing.T) {
	ids := slices.Clone(magicGrokIDs)
	got := SortByRank(llmprovider.ProviderGrok, ids, nil)
	want := []string{
		"grok-4.7", "grok-4.6", "grok-4.5", "grok-4.3",
		"grok-4.20-0309-non-reasoning", "grok-4.20-0309-reasoning", "grok-4.20-multi-agent-0309",
		"grok-build-0.1", "grok-imagine-image", "grok-imagine-image-2.0", "grok-imagine-image-quality",
		"grok-imagine-video", "grok-imagine-video-1.5", "grok-imagine-video-1.5-lite",
	}
	if !slices.Equal(got, want) {
		t.Errorf("SortByRank(grok, Magic's ids) = %v, want %v", got, want)
	}
	if !slices.Equal(ids, magicGrokIDs) {
		t.Errorf("SortByRank changed its argument: %v", ids)
	}
	// A family the static catalog lacks: its dates decide that 4.12 follows
	// 4.5 (generations 142 and 135 as integers, 131 and 135 as decimals).
	leap := []string{"gemini-4.5-flash", "gemini-4.12-flash"}
	released := datesOf(t, map[string]string{"gemini-4.5-flash": "2026-01-10", "gemini-4.12-flash": "2026-06-10"})
	if got := SortByRank(llmprovider.ProviderGemini, leap, released); !slices.Equal(got, []string{"gemini-4.12-flash", "gemini-4.5-flash"}) {
		t.Errorf("SortByRank(gemini, the dated leap) = %v, want gemini-4.12-flash first", got)
	}
	if got := SortByRank(llmprovider.ProviderGrok, nil, nil); got == nil || len(got) != 0 {
		t.Errorf("SortByRank(grok, nil) = %#v, want an empty slice", got)
	}
}
