package catalog

import (
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Moved from llmprovider's catalog_state_test.go (0015-PLAN S8, commit 2).

// TestStaticModels_ReturnsCopy: a caller cannot change the curated catalog.
func TestStaticModels_ReturnsCopy(t *testing.T) {
	first := Static(llmprovider.ProviderClaude)
	first[0] = "CHANGED"
	if Static(llmprovider.ProviderClaude)[0] == "CHANGED" {
		t.Error("a caller's change reached the static catalog")
	}
}

// TestRankModel_Dispatches (0015-PLAN S5 step 3): Rank scores with the
// provider's own ranking, both OpenCode gateways share one, and a provider
// without a ranking scores 0.
func TestRankModel_Dispatches(t *testing.T) {
	for _, tc := range []struct {
		provider llmprovider.ProviderID
		model    string
		rank     func(string) int
	}{
		{llmprovider.ProviderGemini, "gemini-3.7-flash", rankGeminiModel},
		{llmprovider.ProviderOpenAI, "gpt-5.4-mini", rankOpenAIModel},
		{llmprovider.ProviderClaude, "claude-haiku-4-5", rankClaudeModel},
		{llmprovider.ProviderGrok, "grok-4.5", rankGrokModel},
		{llmprovider.ProviderOpencodeZen, "glm-5.3", rankOpencodeModel},
		{llmprovider.ProviderOpencodeGo, "glm-5.3", rankOpencodeModel},
		{llmprovider.ProviderHuggingFace, "zai-org/GLM-5.3-Flash", rankHuggingFaceModel},
		{llmprovider.ProviderKilo, "anthropic/claude-sonnet-5", rankKiloModel},
	} {
		if got, want := Rank(tc.provider, tc.model), tc.rank(tc.model); got != want {
			t.Errorf("RankModel(%s, %s) = %d, want %d", tc.provider, tc.model, got, want)
		}
	}
	for _, p := range []llmprovider.ProviderID{llmprovider.ProviderTogether, llmprovider.ProviderOllama, "unknown"} {
		if got := Rank(p, "any-model"); got != 0 {
			t.Errorf("RankModel(%s) = %d, want 0", p, got)
		}
	}
}
