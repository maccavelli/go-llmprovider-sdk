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
