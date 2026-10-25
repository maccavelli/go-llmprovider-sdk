package gemini

import (
	"context"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// 0041-MADR D2: Gemini's thinking_level from the model's facts, by
// 0062-MADR D3's ladder: a level is sent only when the model lists it, the
// nearest lower first. R5's gemini-2.5-flash-lite exception and the mapping
// of xhigh to high are removed (0062-MADR D8, amendment "Phase 1's
// evidence"): a level the model refuses is learned instead.

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

// sentLevel is the thinking_level a request for effort on model sends, with
// levels listed by a caller layer when not nil, and the response's warnings.
func sentLevel(t *testing.T, model string, effort llmprovider.Effort, levels []llmprovider.Effort) (any,
	[]llmprovider.Warning) {
	t.Helper()
	srv, got := interactionServer(t, interactionText)
	opts := apiKey(srv.URL, model, llmprovider.WithoutModelMetadata())
	if levels != nil {
		opts = append(opts, catalog.WithModelLayer(levelLayer{model, levels}))
	}
	req := text("hi")
	req.Reasoning = &llmprovider.Reasoning{Effort: effort}
	resp, err := build(t, opts...).Generate(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	_, body := got.last()
	return generationConfig(body)["thinking_level"], resp.Warnings
}

// TestThinkingLevel_Facts_0041: with low and high listed, high sends high,
// medium sends low and xhigh high, each lower one warned, and flash-lite's
// low sends low; with no facts, the effort asked, xhigh and flash-lite's low
// included.
func TestThinkingLevel_Facts_0041(t *testing.T) {
	lowHigh := []llmprovider.Effort{llmprovider.EffortLow, llmprovider.EffortHigh}
	for _, tc := range []struct {
		model  string
		effort llmprovider.Effort
		levels []llmprovider.Effort
		want   any
		warned bool
	}{
		{"gemini-3.8-flash", llmprovider.EffortHigh, lowHigh, "high", false},
		{"gemini-3.8-flash", llmprovider.EffortMedium, lowHigh, "low", true},
		{"gemini-3.8-flash", llmprovider.EffortXHigh, lowHigh, "high", true},
		{"gemini-2.5-flash-lite", llmprovider.EffortLow, lowHigh, "low", false},
		{"gemini-3.8-flash", llmprovider.EffortMedium, nil, "medium", false},
		{"gemini-3.8-flash", llmprovider.EffortXHigh, nil, "xhigh", false},
		{"gemini-2.5-flash-lite", llmprovider.EffortLow, nil, "low", false},
	} {
		level, warnings := sentLevel(t, tc.model, tc.effort, tc.levels)
		warned := false
		for _, w := range warnings {
			warned = warned || w.Kind == llmprovider.WarningEffortDegraded
		}
		if level != tc.want || warned != tc.warned {
			t.Errorf("%s %s %v: thinking_level %v, warned %t; want %v, %t", tc.model, tc.effort, tc.levels, level, warned,
				tc.want, tc.warned)
		}
	}
}

// TestThinkingLevel_Wired_0041: the provider sends the level its facts allow:
// with low and high listed, medium sends low, the nearest lower, and high
// sends high; with no facts, medium as asked.
func TestThinkingLevel_Wired_0041(t *testing.T) {
	for _, tc := range []struct {
		facts  bool
		effort llmprovider.Effort
		want   any
	}{
		{true, llmprovider.EffortMedium, "low"},
		{true, llmprovider.EffortHigh, "high"},
		{false, llmprovider.EffortMedium, "medium"},
	} {
		var levels []llmprovider.Effort
		if tc.facts {
			levels = []llmprovider.Effort{llmprovider.EffortLow, llmprovider.EffortHigh}
		}
		if level, _ := sentLevel(t, "gemini-3.8-flash", tc.effort, levels); level != tc.want {
			t.Errorf("facts %t, effort %s: thinking_level %v; want %v", tc.facts, tc.effort, level, tc.want)
		}
	}
}
