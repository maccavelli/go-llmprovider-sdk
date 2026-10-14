package gemini

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// 0041-MADR D2, R5: Gemini's thinking_level from the model's facts: a level
// is sent only when it is in the model's list.

// TestThinkingLevel_Facts_0041: with facts, the level is the rule's mapping
// of the effort, sent only when the list holds it; with none, the rule.
func TestThinkingLevel_Facts_0041(t *testing.T) {
	for _, tc := range []struct {
		model  string
		effort llmprovider.Effort
		levels []string
		known  bool
		want   string
	}{
		{"gemini-3.8-flash", llmprovider.EffortHigh, []string{"low", "high"}, true, "high"},
		{"gemini-3.8-flash", llmprovider.EffortMedium, []string{"low", "high"}, true, ""},
		{"gemini-3.8-flash", llmprovider.EffortXHigh, []string{"low", "high"}, true, "high"},
		{"gemini-2.5-flash-lite", llmprovider.EffortLow, []string{"low", "high"}, true, "low"},
		{"gemini-3.8-flash", llmprovider.EffortMedium, nil, false, "medium"},
		{"gemini-2.5-flash-lite", llmprovider.EffortLow, nil, false, ""},
	} {
		if got := thinkingLevelFor(tc.model, tc.effort, tc.levels, tc.known); got != tc.want {
			t.Errorf("%s %s %v %t: %q; want %q", tc.model, tc.effort, tc.levels, tc.known, got, tc.want)
		}
	}
}

// levelLayer is a caller layer that lists a model's efforts.
type levelLayer struct {
	model   string
	efforts []llmprovider.Effort
}

func (l levelLayer) Name() string { return "test" }

func (l levelLayer) Model(_ llmprovider.ProviderID, id string) (catalog.ModelInfo, bool) {
	if id != l.model {
		return catalog.ModelInfo{}, false
	}
	return catalog.ModelInfo{ID: id, Reasoning: catalog.ModelReasoning{Supported: true, Efforts: l.efforts}}.Known(
		catalog.FieldReasoning, catalog.FieldReasoningEfforts), true
}

// TestThinkingLevel_Wired_0041: the provider sends the level its facts allow:
// with low and high listed, medium sends none and high sends high; with no
// facts, the rule's medium.
func TestThinkingLevel_Wired_0041(t *testing.T) {
	for _, tc := range []struct {
		facts  bool
		effort llmprovider.Effort
		want   any
	}{
		{true, llmprovider.EffortMedium, nil},
		{true, llmprovider.EffortHigh, "high"},
		{false, llmprovider.EffortMedium, "medium"},
	} {
		srv, got := interactionServer(t, interactionText)
		opts := apiKey(srv.URL, "gemini-3.8-flash", llmprovider.WithoutModelMetadata())
		if tc.facts {
			opts = append(opts, catalog.WithModelLayer(levelLayer{"gemini-3.8-flash",
				[]llmprovider.Effort{llmprovider.EffortLow, llmprovider.EffortHigh}}))
		}
		req := text("hi")
		req.Reasoning = &llmprovider.Reasoning{Effort: tc.effort}
		if _, err := build(t, opts...).Generate(context.Background(), req); err != nil {
			t.Fatal(err)
		}
		_, body := got.last()
		if level := generationConfig(body)["thinking_level"]; level != tc.want {
			t.Errorf("facts %t, effort %s: thinking_level %v; want %v", tc.facts, tc.effort, level, tc.want)
		}
	}
}

// TestAgreement0041_GeminiLevel is D7's offline half for R5: on the committed
// snapshot, the level from models.dev's facts equals thinkingLevel's for
// every Google model with listed efforts, at every effort, except the listed
// set.
func TestAgreement0041_GeminiLevel(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "catalog", "testdata", "ranking-2026-10-12", "metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	meta, err := catalog.ParseMetadata(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	// 0041-MADR A2: each lists minimal and high only, so a low or medium
	// level the rule sends is not in its list.
	except := map[string]string{
		"gemini-3-pro-image":             "lists minimal and high",
		"gemini-3.1-flash-image":         "lists minimal and high",
		"gemini-3.1-flash-image-preview": "lists minimal and high",
		"gemini-3.1-flash-lite-image":    "lists minimal and high",
	}
	var differ []string
	for _, info := range meta.Models(llmprovider.ProviderGemini) {
		if !info.Reasoning.Supported || !info.Has(catalog.FieldReasoningEfforts) || len(info.Reasoning.Efforts) == 0 {
			continue
		}
		levels := make([]string, len(info.Reasoning.Efforts))
		for i, e := range info.Reasoning.Efforts {
			levels[i] = string(e)
		}
		for _, effort := range []llmprovider.Effort{llmprovider.EffortLow, llmprovider.EffortMedium, llmprovider.EffortHigh,
			llmprovider.EffortXHigh} {
			if thinkingLevelFor(info.ID, effort, levels, true) != thinkingLevel(info.ID, effort) {
				differ = append(differ, info.ID)
				break
			}
		}
	}
	slices.Sort(differ)
	var want []string
	for id := range except {
		want = append(want, id)
	}
	slices.Sort(want)
	if !slices.Equal(differ, want) {
		t.Errorf("R5: facts and thinkingLevel differ on %v; want only %v (%v)", differ, want, except)
	}
}
