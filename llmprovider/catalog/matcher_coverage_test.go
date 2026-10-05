package catalog

import (
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestSearchModels_GlobQuestionMark covers model_matcher.go's '?' glob rune:
// it matches exactly one character.
func TestSearchModels_GlobQuestionMark(t *testing.T) {
	got := Search(llmprovider.ProviderClaude, []string{"claude-sonnet-5", "claude-sonnet-45", "claude-opus-5"}, "claude-sonnet-?")
	if len(got) != 1 || got[0].ID != "claude-sonnet-5" {
		t.Errorf("claude-sonnet-? = %v, want only claude-sonnet-5", got)
	}
}

// TestSearchModels_LabelSubstringTier covers model_matcher.go's label tier: a
// query found only in a curated label's display name scores
// scoreLabelSubstring, and its bracketed annotation is not searched
// (0021-MADR C11). Every curated display name restates its id, so the test
// labels a model of its own.
func TestSearchModels_LabelSubstringTier(t *testing.T) {
	withLabel(t, "acme/m1", "Roadrunner Large [balanced speed]")
	got := Search(llmprovider.ProviderClaude, []string{"acme/m1", "claude-opus-4-8"}, "roadrunner")
	if len(got) != 1 || got[0].ID != "acme/m1" || got[0].Score != scoreLabelSubstring {
		t.Errorf("roadrunner = %+v, want acme/m1 at scoreLabelSubstring", got)
	}
	if got := Search(llmprovider.ProviderClaude, []string{"acme/m1"}, "balanced speed"); len(got) != 0 {
		t.Errorf("balanced speed = %+v, want no match: it is in the annotation", got)
	}
}
