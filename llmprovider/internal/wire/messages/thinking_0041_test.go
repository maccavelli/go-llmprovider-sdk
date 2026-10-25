package messages

import (
	"encoding/json"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0041-MADR D2, R3: the Messages thinking from a model's facts, and the
// request's shape when there are none (0062-MADR D2).

// TestAddThinkingFor_0041: D2's shape is the facts', else a Budget's, else
// adaptive (0062-MADR D2); and each shape's body, with the facts' bounds and
// output limit, whichever shape D3's ladder sends.
func TestAddThinkingFor_0041(t *testing.T) {
	adaptive, budgetShape := llmprovider.ThinkingAdaptive, llmprovider.ThinkingBudget
	budgetFacts := Thinking{Known: true, BudgetMin: 1024, BudgetMax: 32000}
	for _, tc := range []struct {
		name   string
		facts  Thinking
		budget int
		want   llmprovider.ThinkingShape
	}{
		{"no facts, no budget", Thinking{}, 0, adaptive},
		{"no facts, a budget", Thinking{}, 2000, budgetShape},
		{"facts of a budget, no budget asked", budgetFacts, 0, budgetShape},
		{"facts of efforts, a budget asked", Thinking{Known: true, Adaptive: true, EffortOnly: true}, 2000, adaptive},
	} {
		if got := PreferredShape(tc.facts, tc.budget); got != tc.want {
			t.Errorf("PreferredShape, %s: %s; want %s", tc.name, got, tc.want)
		}
	}
	encode := func(body map[string]any, maxTokens int) string {
		raw, err := json.Marshal(map[string]any{"body": body, "max_tokens": maxTokens})
		if err != nil {
			t.Fatal(err)
		}
		return string(raw)
	}
	for _, tc := range []struct {
		name              string
		effort            string
		budget, maxTokens int
		facts             Thinking
		shape             llmprovider.ThinkingShape
		want              string
	}{
		{"a budget, the default", "high", 0, 8192, budgetFacts, budgetShape,
			`{"body":{"thinking":{"budget_tokens":4096,"type":"enabled"}},"max_tokens":8192}`},
		{"a budget, the default at low", "low", 0, 8192, Thinking{}, budgetShape,
			`{"body":{"thinking":{"budget_tokens":1024,"type":"enabled"}},"max_tokens":8192}`},
		{"a budget cut to its maximum", "", 64000, 8192, budgetFacts, budgetShape,
			`{"body":{"thinking":{"budget_tokens":32000,"type":"enabled"}},"max_tokens":36096}`},
		{"a budget raised to its minimum", "", 500, 8192, budgetFacts, budgetShape,
			`{"body":{"thinking":{"budget_tokens":1024,"type":"enabled"}},"max_tokens":8192}`},
		{"efforts: adaptive, with the effort", "high", 10000, 8192, Thinking{Known: true, Adaptive: true, EffortOnly: true},
			adaptive, `{"body":{"output_config":{"effort":"high"},"thinking":{"type":"adaptive"}},"max_tokens":8192}`},
		{"a toggle alone: adaptive, no effort", "high", 0, 8192, Thinking{Known: true, Adaptive: true}, adaptive,
			`{"body":{"thinking":{"type":"adaptive"}},"max_tokens":8192}`},
		{"a budget model's refusal learned: adaptive, with the effort", "high", 0, 8192, budgetFacts, adaptive,
			`{"body":{"output_config":{"effort":"high"},"thinking":{"type":"adaptive"}},"max_tokens":8192}`},
		{"the raise stops at the output limit", "", 6000, 4096, Thinking{Known: true, MaxOutput: 8000}, budgetShape,
			`{"body":{"thinking":{"budget_tokens":6000,"type":"enabled"}},"max_tokens":8000}`},
		{"a budget at the limit stays under it", "", 9000, 4096, Thinking{Known: true, MaxOutput: 8000}, budgetShape,
			`{"body":{"thinking":{"budget_tokens":7999,"type":"enabled"}},"max_tokens":8000}`},
		{"the limit with no facts", "", 6000, 4096, Thinking{MaxOutput: 8000}, budgetShape,
			`{"body":{"thinking":{"budget_tokens":6000,"type":"enabled"}},"max_tokens":8000}`},
	} {
		body := map[string]any{}
		if got := encode(body, AddThinkingFor(body, tc.effort, tc.budget, tc.maxTokens, tc.facts, tc.shape)); got != tc.want {
			t.Errorf("%s: %s; want %s", tc.name, got, tc.want)
		}
	}
}

// TestFactsThinking_0041: the facts' three shapes, and none without a
// reasoning control.
func TestFactsThinking_0041(t *testing.T) {
	for _, tc := range []struct {
		name    string
		toggle  bool
		efforts []string
		budget  bool
		want    Thinking
	}{
		{"a budget", true, []string{"high"}, true, Thinking{Known: true, BudgetMin: 1024, BudgetMax: 32000, MaxOutput: 64000}},
		{"efforts", true, []string{"low", "high"}, false, Thinking{Known: true, Adaptive: true, EffortOnly: true, MaxOutput: 64000}},
		{"a toggle", true, nil, false, Thinking{Known: true, Adaptive: true, MaxOutput: 64000}},
		{"nothing", false, nil, false, Thinking{MaxOutput: 64000}},
	} {
		if got := FactsThinking(tc.toggle, tc.efforts, tc.budget, 1024, 32000, 64000); got != tc.want {
			t.Errorf("%s: %+v; want %+v", tc.name, got, tc.want)
		}
	}
}
