package generatecontent

import (
	"encoding/json"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0041-MADR D2: Gemini's thinkingConfig from a model's facts, and the
// request's shape when there are none (0062-MADR D2).

// TestThinkingConfigFor_0041: D2's shape is the facts', else a Budget's, else
// the level (0062-MADR D2); the level is thinkingLevel the effort, and the
// budget thinkingBudget, the request's, else R3's: 1024 at low, 4096 else.
func TestThinkingConfigFor_0041(t *testing.T) {
	level, budgetShape := llmprovider.ThinkingAdaptive, llmprovider.ThinkingBudget
	levels := Thinking{Known: true, Levels: []string{"low", "high"}}
	budgetOnly := Thinking{Known: true, Budget: true}
	for _, tc := range []struct {
		name   string
		facts  Thinking
		budget int
		want   llmprovider.ThinkingShape
	}{
		{"no facts, no budget", Thinking{}, 0, level},
		{"no facts, a budget", Thinking{}, 2048, budgetShape},
		{"levels listed, a budget asked: the facts first", levels, 2048, level},
		{"a budget listed", budgetOnly, 0, budgetShape},
	} {
		if got := PreferredShape(tc.facts, tc.budget); got != tc.want {
			t.Errorf("PreferredShape, %s: %s; want %s", tc.name, got, tc.want)
		}
	}
	encode := func(cfg map[string]any) string {
		raw, err := json.Marshal(cfg)
		if err != nil {
			t.Fatal(err)
		}
		return string(raw)
	}
	for _, tc := range []struct {
		name   string
		effort string
		budget int
		shape  llmprovider.ThinkingShape
		want   string
	}{
		{"a level", "high", 0, level, `{"includeThoughts":true,"thinkingLevel":"high"}`},
		{"no effort, no level", "", 0, level, `{"includeThoughts":true}`},
		{"a budget asked", "high", 2048, budgetShape, `{"includeThoughts":true,"thinkingBudget":2048}`},
		{"a budget at low effort", "low", 0, budgetShape, `{"includeThoughts":true,"thinkingBudget":1024}`},
		{"a budget at another effort", "high", 0, budgetShape, `{"includeThoughts":true,"thinkingBudget":4096}`},
	} {
		if got := encode(ThinkingConfigFor(tc.effort, tc.budget, tc.shape)); got != tc.want {
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
