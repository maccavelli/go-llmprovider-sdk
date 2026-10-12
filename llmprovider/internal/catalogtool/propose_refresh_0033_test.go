package catalogtool

import (
	"reflect"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0033-PLAN-refresh-the-curated-catalogs.md Phase 6: each list is proposed by
// its own rule (0033-MADR's amendment "the second refresh, and what the
// command reads", D4b as refined).

// refreshDoc is a models.dev document with an OpenCode Zen section and a
// Claude section: a current Zen model, a newer one the ranker did not pick,
// and a newer Claude model.
const refreshDoc = `{
 "opencode": {"models": {
  "zen-a": {"name": "A", "tool_call": true, "release_date": "2026-09-01", "cost": {"input": 1, "output": 2}},
  "zen-new": {"name": "New", "tool_call": true, "release_date": "2026-10-07", "cost": {"input": 0.1, "output": 0.5}},
  "zen-unpicked": {"name": "Unpicked", "tool_call": true, "release_date": "2026-10-08", "cost": {"input": 9, "output": 9}}
 }},
 "anthropic": {"models": {
  "claude-cur": {"name": "Cur", "tool_call": true, "release_date": "2026-01-01", "cost": {"input": 1, "output": 5}},
  "claude-new": {"name": "New", "tool_call": true, "release_date": "2026-09-28", "cost": {"input": 3, "output": 15}}
 }}
}`

// TestPropose_RankedListDiff: a ranked list's proposals are the difference
// between it and today's utility six, and nothing else.
func TestPropose_RankedListDiff(t *testing.T) {
	zen := llmprovider.ProviderOpencodeZen
	static := map[llmprovider.ProviderID][]string{zen: {"zen-a", "zen-gone"}}
	src := Sources{Metadata: metadata(t, refreshDoc),
		Listed: map[llmprovider.ProviderID][]string{zen: {"zen-a", "zen-gone", "zen-new", "zen-unpicked"}},
		Ranked: map[llmprovider.ProviderID][]string{zen: {"zen-new", "zen-a"}}}
	got := Propose(static, src)
	want := []Proposal{
		{Provider: zen, ID: "zen-gone", Kind: KindRemove, Reason: "not in today's utility six"},
		{Provider: zen, ID: "zen-new", Kind: KindAdd,
			Reason: "ranked 1 of 2 in today's utility six; status active, released 2026-10-07, input $0.10 per million tokens"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("proposals %+v,\nwant %+v", got, want)
	}
}

// TestPropose_RankedListNoBroadAdds: a newer model the ranker did not pick is
// not proposed for a ranked list, and a ranked list whose six was not read
// gets no addition.
func TestPropose_RankedListNoBroadAdds(t *testing.T) {
	zen := llmprovider.ProviderOpencodeZen
	static := map[llmprovider.ProviderID][]string{zen: {"zen-a"}}
	for _, src := range []Sources{
		{Metadata: metadata(t, refreshDoc), Ranked: map[llmprovider.ProviderID][]string{zen: {"zen-a"}}},
		{Metadata: metadata(t, refreshDoc)},
	} {
		if got := Propose(static, src); len(got) != 0 {
			t.Errorf("proposals %+v, want none", got)
		}
	}
}

// TestPropose_RetiredRemovedFromAnyList: an id the retired table names is
// proposed for removal from a ranked list too, even when the ranker picks it.
func TestPropose_RetiredRemovedFromAnyList(t *testing.T) {
	claude := llmprovider.ProviderClaude
	static := map[llmprovider.ProviderID][]string{claude: {"claude-haiku-4-5", "claude-cur"}}
	got := Propose(static, Sources{Metadata: metadata(t, refreshDoc),
		Ranked: map[llmprovider.ProviderID][]string{claude: {"claude-haiku-4-5", "claude-cur"}}})
	want := Proposal{Provider: claude, ID: "claude-haiku-4-5", Kind: KindRemove, Reason: "retired: retirement floor 2026-10-15"}
	if len(got) == 0 || !reflect.DeepEqual(got[0], want) {
		t.Errorf("proposals %+v, want first %+v", got, want)
	}
}

// TestPropose_VendorReviewLines: a vendor list's newer model is a review line,
// never an addition, and its line says to check the official page.
func TestPropose_VendorReviewLines(t *testing.T) {
	claude := llmprovider.ProviderClaude
	got := Propose(map[llmprovider.ProviderID][]string{claude: {"claude-cur"}}, Sources{Metadata: metadata(t, refreshDoc)})
	if len(got) != 1 || got[0].Kind != KindReview || got[0].ID != "claude-new" {
		t.Fatalf("proposals %+v, want one review line for claude-new", got)
	}
	if line := got[0].String(); !strings.HasPrefix(line, "claude\treview\tclaude-new\treview against the official page; ") {
		t.Errorf("String() = %q", line)
	}
}

// TestRankedLists: the ranked lists are the five gateways (Q17), and no
// vendor list.
func TestRankedLists(t *testing.T) {
	for _, p := range []llmprovider.ProviderID{llmprovider.ProviderOpencodeZen, llmprovider.ProviderOpencodeGo,
		llmprovider.ProviderKilo, llmprovider.ProviderHuggingFace, llmprovider.ProviderTogether} {
		if !Ranked(p) {
			t.Errorf("Ranked(%s) = false, want true", p)
		}
	}
	for _, p := range []llmprovider.ProviderID{llmprovider.ProviderOpenAI, llmprovider.ProviderClaude,
		llmprovider.ProviderGemini, llmprovider.ProviderGrok} {
		if Ranked(p) {
			t.Errorf("Ranked(%s) = true, want false", p)
		}
	}
}
