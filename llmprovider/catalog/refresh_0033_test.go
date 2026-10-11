package catalog

import (
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// retired0033 are the ids each provider's deprecations page, or its listing,
// on 2026-10-11 marked retired, scheduled for shutdown, routed to another
// model, or open only to past users, each with why (0033-MADR D4;
// 0033-PLAN-refresh-the-curated-catalogs.md Phase 1).
var retired0033 = map[llmprovider.ProviderID]map[string]string{
	llmprovider.ProviderOpenAI: {
		"gpt-4.1-nano": "shuts down 2026-10-23",
		"o4-mini":      "shuts down 2026-10-23",
	},
	llmprovider.ProviderGemini: {
		"gemini-3.7-flash":      "routed to gemini-3.8-flash",
		"gemini-3.5-flash":      "routed to gemini-3.6-flash",
		"gemini-2.5-flash":      "open only to past users",
		"gemini-2.5-flash-lite": "open only to past users",
		"gemini-3.1-flash-lite": "shuts down 2027-05-07",
	},
	llmprovider.ProviderClaude: {
		"claude-haiku-4-5":           "retirement floor 2026-10-15",
		"claude-sonnet-4-5-20250929": "retires 2026-11-30",
	},
	llmprovider.ProviderGrok: {
		"grok-4-fast-reasoning": "retired 2026-05-15",
		"grok-4":                "not listed; grok-4-0709 retired 2026-05-15",
		"grok-3-mini":           "not listed",
		"grok-3-mini-fast":      "not listed",
		"grok-3":                "retired 2026-05-15",
	},
}

// waiting0033 are current models a static catalog leaves out. 0034-MADR,
// the record 0033-MADR D6 named, put claude-sonnet-5-5 and claude-opus-5-5
// in the menu; claude-fable-5-1 stays out of it (0034-MADR D5).
var waiting0033 = map[llmprovider.ProviderID][]string{
	llmprovider.ProviderClaude: {"claude-fable-5-1"},
}

// TestStatic_NoRetiredIDs_0033 (0033-MADR D4): no static catalog names an id
// its provider has retired, scheduled, rerouted or restricted, nor one that
// waits for D6's record.
func TestStatic_NoRetiredIDs_0033(t *testing.T) {
	for p, ids := range retired0033 {
		for _, id := range Static(p) {
			if why, ok := ids[id]; ok {
				t.Errorf("%s: static catalog lists %q, which is %s", p, id, why)
			}
		}
	}
	for p, ids := range waiting0033 {
		for _, id := range Static(p) {
			for _, w := range ids {
				if id == w {
					t.Errorf("%s: static catalog lists %q, which 0034-MADR D5 leaves out of the menu", p, id)
				}
			}
		}
	}
}

// TestRank_PrefersCurrentModels_0033 (0033-MADR D4): in a live listing, a
// current model ranks above the retired one it replaces.
func TestRank_PrefersCurrentModels_0033(t *testing.T) {
	for _, c := range []struct {
		provider         llmprovider.ProviderID
		current, retired string
	}{
		{llmprovider.ProviderOpenAI, "gpt-6-luna", "gpt-4.1-nano"},
		{llmprovider.ProviderOpenAI, "gpt-5.6-terra", "o4-mini"},
		{llmprovider.ProviderGemini, "gemini-3.8-flash", "gemini-3.7-flash"},
		{llmprovider.ProviderClaude, "claude-haiku-5-5", "claude-haiku-4-5"},
		{llmprovider.ProviderGrok, "grok-4.7", "grok-3-mini-fast"},
		{llmprovider.ProviderGrok, "grok-4.3", "grok-4-fast-reasoning"},
	} {
		if cur, ret := Rank(c.provider, c.current), Rank(c.provider, c.retired); cur <= ret {
			t.Errorf("%s: Rank(%q) = %d, not above Rank(%q) = %d", c.provider, c.current, cur, c.retired, ret)
		}
	}
}

// TestLabel_ByProvider_0033 (0033-MADR D4): Label's provider argument counts.
// An id two providers offer takes each one's label, and a gateway's fallback
// six keep the bare id.
func TestLabel_ByProvider_0033(t *testing.T) {
	for _, c := range []struct {
		provider llmprovider.ProviderID
		id, want string
	}{
		{llmprovider.ProviderGemini, "gemini-3.8-flash", "Gemini 3.8 Flash"},
		{llmprovider.ProviderOpenAI, "gpt-6-luna", "GPT-6 Luna"},
		{llmprovider.ProviderClaude, "claude-haiku-5-5", "Claude Haiku 5.5"},
		{llmprovider.ProviderGrok, "grok-4.7", "Grok 4.7"},
	} {
		if got := Label(c.provider, c.id); !strings.HasPrefix(got, c.want) {
			t.Errorf("Label(%s, %q) = %q, want a label starting %q", c.provider, c.id, got, c.want)
		}
	}
	for _, c := range []struct {
		provider llmprovider.ProviderID
		id       string
	}{
		{llmprovider.ProviderOpencodeZen, "gemini-3.8-flash"},
		{llmprovider.ProviderOpencodeZen, "gemini-3.5-flash-lite"},
		{llmprovider.ProviderOpencodeGo, "gpt-6-luna"},
	} {
		if got := Label(c.provider, c.id); got != c.id {
			t.Errorf("Label(%s, %q) = %q, want the bare id: a gateway's fallback six carry no label", c.provider, c.id, got)
		}
	}
}
