package generatecontent

import (
	"encoding/json"
	"testing"
)

// 0041-MADR D2, R4–R5: Gemini's thinkingConfig from a model's facts, and the
// rule when there are none.

// TestThinkingConfigFor_0041: with no facts it is ThinkingConfig; a budget
// without efforts is thinkingBudget; efforts are thinkingLevel, sent only
// when the level is in the model's list.
func TestThinkingConfigFor_0041(t *testing.T) {
	encode := func(cfg map[string]any) string {
		raw, err := json.Marshal(cfg)
		if err != nil {
			t.Fatal(err)
		}
		return string(raw)
	}
	for _, model := range []string{"gemini-2.5-flash", "gemini-3.8-flash", "gemini-x"} {
		for _, effort := range []string{"", "low", "medium", "high"} {
			for _, budget := range []int{0, 2048} {
				if got, want := encode(ThinkingConfigFor(model, effort, budget, Thinking{})), encode(ThinkingConfig(model, effort, budget)); got != want {
					t.Errorf("%s %q %d with no facts: %s; want ThinkingConfig's %s", model, effort, budget, got, want)
				}
			}
		}
	}
	levels := Thinking{Known: true, Levels: []string{"low", "high"}}
	budgetOnly := Thinking{Known: true, Budget: true}
	for _, tc := range []struct {
		name   string
		model  string
		effort string
		budget int
		facts  Thinking
		want   string
	}{
		{"a configured budget wins", "gemini-3.8-flash", "high", 2048, levels, `{"includeThoughts":true,"thinkingBudget":2048}`},
		{"a level in the list", "gemini-2.5-flash", "high", 0, levels, `{"includeThoughts":true,"thinkingLevel":"high"}`},
		{"a level not in the list", "gemini-3.8-flash", "medium", 0, levels, `{"includeThoughts":true}`},
		{"no effort, no level", "gemini-3.8-flash", "", 0, levels, `{"includeThoughts":true}`},
		{"a budget at low effort", "gemini-3.8-flash", "low", 0, budgetOnly, `{"includeThoughts":true,"thinkingBudget":1024}`},
		{"a budget at another effort", "gemini-3.8-flash", "high", 0, budgetOnly, `{"includeThoughts":true,"thinkingBudget":-1}`},
	} {
		if got := encode(ThinkingConfigFor(tc.model, tc.effort, tc.budget, tc.facts)); got != tc.want {
			t.Errorf("%s: %s; want %s", tc.name, got, tc.want)
		}
	}
}

// TestFactsThinking_0041: efforts are levels; a budget without efforts is a
// budget; neither is no facts.
func TestFactsThinking_0041(t *testing.T) {
	if got := FactsThinking([]string{"low", "high"}, true); !got.Known || got.Budget || len(got.Levels) != 2 {
		t.Errorf("efforts and a budget: %+v; want the levels", got)
	}
	if got := FactsThinking(nil, true); !got.Known || !got.Budget {
		t.Errorf("a budget: %+v; want Budget", got)
	}
	if got := FactsThinking(nil, false); got.Known {
		t.Errorf("neither: %+v; want no facts", got)
	}
}
