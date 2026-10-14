package catalog

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/generatecontent"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/messages"
)

// 0041-MADR D7's offline half: on the committed snapshot, each decision from
// models.dev's facts equals its rule for every model both answer, except a
// listed set, each with its reason. R5 and R6 are tested in providers/gemini
// and providers/grok, whose rules are unexported there.

// messagesFacts is the Thinking models.dev's facts give info.
func messagesFacts(info ModelInfo) messages.Thinking {
	efforts := make([]string, len(info.Reasoning.Efforts))
	for i, e := range info.Reasoning.Efforts {
		efforts[i] = string(e)
	}
	return messages.FactsThinking(info.Reasoning.Toggle, efforts, info.Has(FieldReasoningBudget),
		info.Reasoning.BudgetMin, info.Reasoning.BudgetMax, 0)
}

// TestAgreement0041_MessagesThinking is R1 and R2: AddThinkingFor from facts
// sends AddThinking's body on every Claude reasoning model of the Messages
// sections, and on MiniMax-M3, at each effort.
func TestAgreement0041_MessagesThinking(t *testing.T) {
	doc := evidence0041Doc(t)
	meta := Metadata{doc: doc}
	except := map[string]string{}
	var differ, silent []string
	seen := 0
	for _, provider := range []llmprovider.ProviderID{llmprovider.ProviderClaude, llmprovider.ProviderOpencodeZen,
		llmprovider.ProviderOpencodeGo} {
		for _, info := range meta.Models(provider) {
			facts := messagesFacts(info)
			// Every model the Messages wire serves: Anthropic's, and
			// OpenCode's on its messages route (its npm package).
			npm, _ := meta.NPM(provider, info.ID)
			if !info.Reasoning.Supported || provider != llmprovider.ProviderClaude && npm != "@ai-sdk/anthropic" {
				continue
			}
			if !facts.Known {
				silent = append(silent, string(provider)+"/"+info.ID)
				continue
			}
			seen++
			for _, effort := range []string{"", "low", "high"} {
				rule, fromFacts := map[string]any{}, map[string]any{}
				a, _ := json.Marshal([]any{rule, messages.AddThinking(rule, info.ID, effort, 0, 8192)})
				b, _ := json.Marshal([]any{fromFacts, messages.AddThinkingFor(fromFacts, info.ID, effort, 0, 8192, facts)})
				if string(a) != string(b) {
					differ = append(differ, string(provider)+"/"+info.ID)
					t.Logf("%s/%s at %q: rule %s, facts %s", provider, info.ID, effort, a, b)
					break
				}
			}
		}
	}
	if seen == 0 {
		t.Fatal("no model of the three sections has reasoning facts")
	}
	slices.Sort(differ)
	slices.Sort(silent)
	// 0041-MADR A2: these list reasoning with no option, so the rule decides.
	wantSilent := []string{"opencode-go/minimax-m2.7", "opencode-zen/minimax-m2.1-free", "opencode-zen/minimax-m2.5-free"}
	if len(differ) != len(except) || !slices.Equal(silent, wantSilent) {
		t.Errorf("R1, R2: facts and the rule differ on %v, facts silent on %v; want only %v, and silent on %v",
			differ, silent, except, wantSilent)
	}
}

// TestAgreement0041_GeminiConfig is R4: at low effort, ThinkingConfigFor from
// facts sends ThinkingConfig's budget or level on every Google reasoning
// model whose facts answer, except three image models; the facts of nine
// are silent, §3's two among them, and the rule decides for them (0041-MADR
// A2).
func TestAgreement0041_GeminiConfig(t *testing.T) {
	doc := evidence0041Doc(t)
	// Each lists minimal and high only: the rule's low is not in its list.
	differWant := []string{"gemini-3.1-flash-image", "gemini-3.1-flash-image-preview", "gemini-3.1-flash-lite-image"}
	// Each lists reasoning with no budget and no effort.
	silentWant := map[string]string{
		"deep-research-max-preview-04-2026":       "a research agent",
		"deep-research-preview-04-2026":           "a research agent",
		"gemini-2.5-computer-use-preview-10-2025": "§3: a computer-use model",
		"gemini-2.5-flash-image":                  "§3: an image model",
		"gemini-3-pro-image-preview":              "an image model",
		"gemini-3.1-flash-tts-preview":            "a speech model",
		"gemini-omni-flash-preview":               "a multimodal preview",
		"gemma-4-26b-a4b-it":                      "an open-weights model",
		"gemma-4-31b-it":                          "an open-weights model",
	}
	var differ, silent []string
	for _, info := range (Metadata{doc: doc}).Models(llmprovider.ProviderGemini) {
		if !info.Reasoning.Supported {
			continue
		}
		efforts := make([]string, len(info.Reasoning.Efforts))
		for i, e := range info.Reasoning.Efforts {
			efforts[i] = string(e)
		}
		facts := generatecontent.FactsThinking(efforts, info.Has(FieldReasoningBudget))
		if !facts.Known {
			silent = append(silent, info.ID)
			continue
		}
		rule, _ := json.Marshal(generatecontent.ThinkingConfig(info.ID, "low", 0))
		fromFacts, _ := json.Marshal(generatecontent.ThinkingConfigFor(info.ID, "low", 0, facts))
		if string(rule) != string(fromFacts) {
			differ = append(differ, info.ID)
			t.Logf("%s: rule %s, facts %s", info.ID, rule, fromFacts)
		}
	}
	slices.Sort(differ)
	slices.Sort(silent)
	var want []string
	for id := range silentWant {
		want = append(want, id)
	}
	slices.Sort(want)
	if !slices.Equal(differ, differWant) || !slices.Equal(silent, want) {
		t.Errorf("R4: facts and the rule differ on %v, facts silent on %v; want differing %v, silent on %v",
			differ, silent, differWant, want)
	}
}
