package llmprovider

import "testing"

// TestProviderEnvVars_ReturnsCopy (0015-MADR D9, 0015-PLAN S5 step 2): the
// map is a copy; changing it changes nothing the package reads.
func TestProviderEnvVars_ReturnsCopy(t *testing.T) {
	first := ProviderEnvVars()
	if first[ProviderTogether] != "TOGETHER_API_KEY" || first[ProviderOpenAI] != "OPENAI_API_KEY" {
		t.Fatalf("ProviderEnvVars() = %v", first)
	}
	first[ProviderOpenAI] = "CHANGED"
	delete(first, ProviderTogether)
	if again := ProviderEnvVars(); again[ProviderOpenAI] != "OPENAI_API_KEY" || again[ProviderTogether] == "" {
		t.Errorf("a caller's change reached the package: %v", again)
	}
}

// TestStaticModels_ReturnsCopy: a caller cannot change the curated catalog.
func TestStaticModels_ReturnsCopy(t *testing.T) {
	first := StaticModels(ProviderClaude)
	first[0] = "CHANGED"
	if StaticModels(ProviderClaude)[0] == "CHANGED" {
		t.Error("a caller's change reached the static catalog")
	}
}

// TestRankModel_Dispatches (0015-PLAN S5 step 3): RankModel scores with the
// provider's own ranking, both OpenCode gateways share one, and a provider
// without a ranking scores 0.
func TestRankModel_Dispatches(t *testing.T) {
	for _, tc := range []struct {
		provider, model string
		rank            func(string) int
	}{
		{ProviderGemini, "gemini-3.7-flash", rankGeminiModel},
		{ProviderOpenAI, "gpt-5.4-mini", rankOpenAIModel},
		{ProviderClaude, "claude-haiku-4-5", rankClaudeModel},
		{ProviderGrok, "grok-4.5", rankGrokModel},
		{ProviderOpencodeZen, "glm-5.3", rankOpencodeModel},
		{ProviderOpencodeGo, "glm-5.3", rankOpencodeModel},
		{ProviderHuggingFace, "zai-org/GLM-5.3-Flash", rankHuggingFaceModel},
		{ProviderKilo, "anthropic/claude-sonnet-5", rankKiloModel},
	} {
		if got, want := RankModel(tc.provider, tc.model), tc.rank(tc.model); got != want {
			t.Errorf("RankModel(%s, %s) = %d, want %d", tc.provider, tc.model, got, want)
		}
	}
	for _, p := range []string{ProviderTogether, ProviderOllama, "unknown"} {
		if got := RankModel(p, "any-model"); got != 0 {
			t.Errorf("RankModel(%s) = %d, want 0", p, got)
		}
	}
}
