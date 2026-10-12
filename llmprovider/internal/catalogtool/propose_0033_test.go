package catalogtool

import (
	"reflect"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// 0033-PLAN-public-model-metadata.md Phase 4: the maintainer tool's proposals
// (0033-MADR D4).

// proposeDoc is a models.dev document for Claude: one current model the menu
// lists, one newer current model, one newer deprecated one, and one older.
const proposeDoc = `{"anthropic": {"models": {
 "claude-cur": {"name": "Cur", "tool_call": true, "release_date": "2026-01-01", "cost": {"input": 1, "output": 5}},
 "claude-new": {"name": "New", "tool_call": true, "release_date": "2026-09-28", "cost": {"input": 3, "output": 15}},
 "claude-dep": {"name": "Dep", "tool_call": true, "status": "deprecated", "release_date": "2026-10-01", "cost": {"input": 2, "output": 8}},
 "claude-old": {"name": "Old", "tool_call": true, "release_date": "2025-01-01", "cost": {"input": 1, "output": 5}}
}}}`

func metadata(t *testing.T, doc string) catalog.Metadata {
	t.Helper()
	m, err := catalog.ParseMetadata(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// TestPropose_RemovesARetiredID_0033: a retired id in a static list is
// proposed for removal, with the reason the table gives.
func TestPropose_RemovesARetiredID_0033(t *testing.T) {
	static := map[llmprovider.ProviderID][]string{llmprovider.ProviderClaude: {"claude-haiku-4-5", "claude-cur"}}
	got := Propose(static, Sources{Metadata: metadata(t, proposeDoc)})
	want := Proposal{Provider: llmprovider.ProviderClaude, ID: "claude-haiku-4-5", Add: false, Reason: "retired: retirement floor 2026-10-15"}
	if len(got) == 0 || !reflect.DeepEqual(got[0], want) {
		t.Errorf("proposals %+v, want first %+v", got, want)
	}
}

// TestPropose_AddsACurrentModel_0033: a newer current model is proposed for
// addition with its status, release date and input price; a deprecated one,
// or one older than the list's newest, is not.
func TestPropose_AddsACurrentModel_0033(t *testing.T) {
	static := map[llmprovider.ProviderID][]string{llmprovider.ProviderClaude: {"claude-cur"}}
	got := Propose(static, Sources{Metadata: metadata(t, proposeDoc)})
	want := []Proposal{{Provider: llmprovider.ProviderClaude, ID: "claude-new", Add: true,
		Reason: "status active, released 2026-09-28, input $3.00 per million tokens"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("proposals %+v, want %+v", got, want)
	}
}

// TestPropose_NothingForACurrentList_0033: a list already current gets no
// proposal.
func TestPropose_NothingForACurrentList_0033(t *testing.T) {
	static := map[llmprovider.ProviderID][]string{llmprovider.ProviderClaude: {"claude-cur", "claude-new"}}
	if got := Propose(static, Sources{Metadata: metadata(t, proposeDoc)}); len(got) != 0 {
		t.Errorf("proposals %+v, want none", got)
	}
}

// TestPropose_NeverAddsADeprecatedModel_0033: a deprecated model is never
// proposed for addition, even from a listing that offers it; one the
// listing lacks, in a static list, is proposed for removal.
func TestPropose_NeverAddsADeprecatedModel_0033(t *testing.T) {
	static := map[llmprovider.ProviderID][]string{llmprovider.ProviderClaude: {"claude-cur", "claude-gone"}}
	listed := map[llmprovider.ProviderID][]string{llmprovider.ProviderClaude: {"claude-cur", "claude-dep"}}
	got := Propose(static, Sources{Metadata: metadata(t, proposeDoc), Listed: listed})
	want := []Proposal{{Provider: llmprovider.ProviderClaude, ID: "claude-gone", Add: false, Reason: "not in the listing"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("proposals %+v, want %+v", got, want)
	}
}

// TestPropose_Deterministic_0033: the same input gives the same output,
// sorted by provider, then id.
func TestPropose_Deterministic_0033(t *testing.T) {
	static := map[llmprovider.ProviderID][]string{
		llmprovider.ProviderClaude: {"claude-haiku-4-5", "claude-cur"},
		llmprovider.ProviderOpenAI: {"o4-mini", "gpt-4.1-nano"},
	}
	first := Propose(static, Sources{Metadata: metadata(t, proposeDoc)})
	for range 5 {
		if again := Propose(static, Sources{Metadata: metadata(t, proposeDoc)}); !reflect.DeepEqual(first, again) {
			t.Fatalf("proposals differ between runs: %+v, then %+v", first, again)
		}
	}
	var keys []string
	for _, p := range first {
		keys = append(keys, string(p.Provider)+"/"+p.ID)
	}
	if want := []string{"claude/claude-haiku-4-5", "claude/claude-new", "openai/gpt-4.1-nano", "openai/o4-mini"}; !reflect.DeepEqual(keys, want) {
		t.Errorf("order %v, want %v", keys, want)
	}
	if line := first[0].String(); line != "claude\tremove\tclaude-haiku-4-5\tretired: retirement floor 2026-10-15" {
		t.Errorf("String() = %q", line)
	}
}
