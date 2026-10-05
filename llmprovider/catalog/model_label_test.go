package catalog

import (
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestModelLabel was in descriptor_test.go, whose descriptors moved to the
// provider packages (0015-PLAN S8, commit 1).
func TestModelLabel(t *testing.T) {
	if got := Label(llmprovider.ProviderClaude, "claude-haiku-4-5"); !strings.Contains(got, "Haiku") {
		t.Errorf("ModelLabel = %q, want a human label", got)
	}
	// Unknown models degrade to the bare id rather than disappearing.
	if got := Label(llmprovider.ProviderKilo, "some/unlisted-model"); got != "some/unlisted-model" {
		t.Errorf("unknown model label = %q, want the bare id", got)
	}
	if got := Label(llmprovider.ProviderGemini, ""); got != "" {
		t.Errorf("empty model label = %q, want empty", got)
	}
}

// fallbackSix is why a static id has no label: it is one of a
// metadata-ranked catalog's fallback six, which a live listing replaces, and
// a label's annotation would claim what the listing's metadata decides.
const fallbackSix = "a metadata-ranked catalog's fallback six"

// unlabelledStaticIDs are the static ids that have no curated label, each
// with the reason (0021-MADR C12).
var unlabelledStaticIDs = map[string]string{
	// Kilo
	"deepseek/deepseek-v4.1-flash": fallbackSix, "z-ai/glm-5.3-flash": fallbackSix,
	"google/gemini-3.8-flash": fallbackSix, "google/gemini-3.6-flash": fallbackSix,
	"meta/muse-spark-1.2": fallbackSix, "thinkingmachines/inkling": fallbackSix,
	// OpenCode Zen and Go
	"deepseek-v4.1-flash": fallbackSix, "qwen3.8-flash": fallbackSix, "glm-5.3-flash": fallbackSix,
	"deepseek-v4-flash": fallbackSix, "gemini-3.5-flash-lite": fallbackSix, "gemini-3.8-flash": fallbackSix,
	"mimo-v2.6-flash": fallbackSix, "gpt-6-luna": fallbackSix, "mimo-v2.6-pro": fallbackSix, "hy3": fallbackSix,
	// Hugging Face
	"deepseek-ai/DeepSeek-V4-Flash-0731": fallbackSix, "zai-org/GLM-5.3-Flash": fallbackSix,
	"deepseek-ai/DeepSeek-V4.1-Flash": fallbackSix, "thinkingmachines/Inkling-Small": fallbackSix,
	"stepfun-ai/Step-3.7-Flash": fallbackSix, "stepfun-ai/Step-3.5-Flash": fallbackSix,
	// Together; its deepseek-ai/DeepSeek-V4.1-Flash is listed under Hugging
	// Face.
	"zai-org/GLM-5.3": fallbackSix, "moonshotai/Kimi-K3": fallbackSix,
	"MiniMaxAI/MiniMax-M3": fallbackSix, "Qwen/Qwen3.6-Plus": fallbackSix,
}

// staticProviders are the providers with a static catalog.
var staticProviders = []llmprovider.ProviderID{
	llmprovider.ProviderGemini, llmprovider.ProviderOpenAI, llmprovider.ProviderClaude, llmprovider.ProviderGrok,
	llmprovider.ProviderOpencodeZen, llmprovider.ProviderOpencodeGo, llmprovider.ProviderHuggingFace,
	llmprovider.ProviderKilo, llmprovider.ProviderTogether,
}

// TestStaticIDs_Labelled (0021-MADR C12): every static id has a curated
// label, or is listed with the reason it has none.
func TestStaticIDs_Labelled(t *testing.T) {
	for _, p := range staticProviders {
		for _, id := range Static(p) {
			_, labelled := modelLabels[id]
			if _, excused := unlabelledStaticIDs[id]; !labelled && !excused {
				t.Errorf("%s: static id %q has no label and no reason in unlabelledStaticIDs", p, id)
			}
		}
	}
}

// TestLabels_NoOrphans (0021-MADR C12): every curated label belongs to a
// static id.
func TestLabels_NoOrphans(t *testing.T) {
	for id := range modelLabels {
		if !slices.ContainsFunc(staticProviders, func(p llmprovider.ProviderID) bool { return slices.Contains(Static(p), id) }) {
			t.Errorf("label for %q, which no static catalog lists", id)
		}
	}
}
