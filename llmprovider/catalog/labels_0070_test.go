package catalog

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// labelGolden is every curated Label string, whole (0070-MADR D2). The twelve
// whose note did not change equal b5ce51d's; the eight others differ from
// b5ce51d's only inside the brackets (0070-PLAN Phase 2, step 5).
var labelGolden = map[llmprovider.ProviderID]map[string]string{
	llmprovider.ProviderGemini: {
		"gemini-3.5-flash-lite": "Gemini 3.5 Flash-Lite  [★ Recommended: ultra-low latency]",
		"gemini-3.8-flash":      "Gemini 3.8 Flash       [frontier coding]",
		"gemini-3.6-flash":      "Gemini 3.6 Flash       [high efficiency, reduced token overhead]",
	},
	llmprovider.ProviderOpenAI: {
		"gpt-6-luna":    "GPT-6 Luna             [★ Recommended: fast & cost-effective]",
		"gpt-5.6-luna":  "GPT-5.6 Luna           [efficient tier]",
		"gpt-4.1-mini":  "GPT-4.1 Mini           [fast, no reasoning]",
		"gpt-5.6-terra": "GPT-5.6 Terra          [balanced reasoning]",
		"gpt-6.1-sol":   "GPT-6.1 Sol            [near-flagship, lower cost]",
		"gpt-6-astra":   "GPT-6 Astra            [maximum capability]",
	},
	llmprovider.ProviderClaude: {
		"claude-haiku-5-5":  "Claude Haiku 5.5       [★ Recommended: high speed, low latency]",
		"claude-sonnet-5-5": "Claude Sonnet 5.5      [high precision]",
		"claude-sonnet-5":   "Claude Sonnet 5        [balanced speed & precision]",
		"claude-opus-5-5":   "Claude Opus 5.5        [maximum capability]",
		"claude-opus-4-8":   "Claude Opus 4.8        [maximum capability of Opus 4]",
	},
	llmprovider.ProviderGrok: {
		"grok-4.6":       "Grok 4.6               [★ Recommended: the Grok CLI's built-in default]",
		"grok-4.5":       "Grok 4.5               [second in the Grok CLI's built-in list]",
		"grok-4.7":       "Grok 4.7               [frontier model]",
		"grok-4.3":       "Grok 4.3               [low cost, long context]",
		"grok-build-0.1": "Grok Build 0.1         [coding, early access]",
	},
	llmprovider.ProviderTogether: {
		"openai/gpt-oss-120b": "GPT-OSS 120B           [highest throughput]",
	},
}

// composeLabel is Label's rule written out from its parts: the name padded to
// 23 columns, then the note in brackets, starred on the provider's first
// static id; the name alone when there is no note.
func composeLabel(p llmprovider.ProviderID, id string) string {
	name, note := LabelName(p, id), LabelNote(p, id)
	if note == "" {
		return name
	}
	if static := Static(p); len(static) > 0 && static[0] == id {
		note = "★ Recommended: " + note
	}
	return fmt.Sprintf("%-23s[%s]", name, note)
}

// TestLabel_Parts_0070 (0070-MADR D2): every curated Label equals its golden
// string and the composition of LabelName and LabelNote, and the golden
// holds every entry of modelLabels.
func TestLabel_Parts_0070(t *testing.T) {
	n := 0
	for _, p := range slices.Sorted(maps.Keys(labelGolden)) {
		for _, id := range slices.Sorted(maps.Keys(labelGolden[p])) {
			n++
			want := labelGolden[p][id]
			if got := Label(p, id); got != want {
				t.Errorf("Label(%s, %q) = %q, want %q", p, id, got, want)
			}
			if got, composed := Label(p, id), composeLabel(p, id); got != composed {
				t.Errorf("Label(%s, %q) = %q, want the composition of its parts, %q", p, id, got, composed)
			}
		}
	}
	if n != 20 {
		t.Errorf("golden holds %d labels, want 20", n)
	}
	for p, labels := range modelLabels {
		for id := range labels {
			if _, ok := labelGolden[p][id]; !ok {
				t.Errorf("%s %q is labelled but not in the golden", p, id)
			}
		}
	}
}

// TestLabelName_Fallback_0070 (0070-MADR D2; 0033-MADR D4): an unlabelled id,
// and an id labelled for another provider only, give the id and no note. The
// provider is compared case-insensitively, as Label compares it.
func TestLabelName_Fallback_0070(t *testing.T) {
	for _, c := range []struct {
		p  llmprovider.ProviderID
		id string
	}{
		{llmprovider.ProviderGrok, "grok-4.20-0309-reasoning"},
		{llmprovider.ProviderOpencodeZen, "gemini-3.8-flash"},
		{llmprovider.ProviderOpencodeGo, "gpt-6-luna"},
		{llmprovider.ProviderGrok, ""},
	} {
		if got := LabelName(c.p, c.id); got != c.id {
			t.Errorf("LabelName(%s, %q) = %q, want the id", c.p, c.id, got)
		}
		if got := LabelNote(c.p, c.id); got != "" {
			t.Errorf("LabelNote(%s, %q) = %q, want none", c.p, c.id, got)
		}
	}
	if got := LabelName("GROK", "grok-4.7"); got != "Grok 4.7" {
		t.Errorf(`LabelName("GROK", "grok-4.7") = %q, want "Grok 4.7"`, got)
	}
	if got := LabelNote("GROK", "grok-4.7"); got != "frontier model" {
		t.Errorf(`LabelNote("GROK", "grok-4.7") = %q, want "frontier model"`, got)
	}
}

// TestLabel_StarIsFirstStatic_0070 (0070-MADR D2, D3): the star is on exactly
// the first Static id of each provider with an entry, when that id has one.
// Together's first static id is unlabelled (0070-MADR Context §3), so none of
// its labels is starred; every other provider with an entry has exactly one.
func TestLabel_StarIsFirstStatic_0070(t *testing.T) {
	for _, p := range slices.Sorted(maps.Keys(modelLabels)) {
		first := Static(p)[0]
		_, firstLabelled := modelLabels[p][first]
		var starred []string
		for _, id := range slices.Sorted(maps.Keys(modelLabels[p])) {
			if strings.Contains(Label(p, id), "★") {
				starred = append(starred, id)
			}
		}
		want := []string{}
		if firstLabelled {
			want = []string{first}
		}
		if !slices.Equal(starred, want) {
			t.Errorf("%s: starred %q, want %q, the first static id's label", p, starred, want)
		}
	}
	for _, p := range []llmprovider.ProviderID{
		llmprovider.ProviderGemini, llmprovider.ProviderOpenAI, llmprovider.ProviderClaude, llmprovider.ProviderGrok,
	} {
		if _, ok := modelLabels[p][Static(p)[0]]; !ok {
			t.Errorf("%s: the first static id %q has no label to star", p, Static(p)[0])
		}
	}
}
