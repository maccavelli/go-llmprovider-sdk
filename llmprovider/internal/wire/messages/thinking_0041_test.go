package messages

import (
	"encoding/json"
	"testing"
)

// 0041-MADR D2, R1–R3: the Messages thinking from a model's facts, and the
// rule when there are none.

// TestAddThinkingFor_0041: with no facts the body and max_tokens are
// AddThinking's, byte for byte; with facts, each of the three shapes.
func TestAddThinkingFor_0041(t *testing.T) {
	encode := func(body map[string]any, maxTokens int) string {
		raw, err := json.Marshal(map[string]any{"body": body, "max_tokens": maxTokens})
		if err != nil {
			t.Fatal(err)
		}
		return string(raw)
	}
	for _, model := range []string{"claude-sonnet-5", "claude-haiku-4-5", "minimax-m3", "glm-5", "claude-x"} {
		for _, effort := range []string{"", "low", "high"} {
			for _, budget := range []int{0, 10000} {
				for _, maxTokens := range []int{2000, 8192} {
					rule, facts := map[string]any{}, map[string]any{}
					want := encode(rule, AddThinking(rule, model, effort, budget, maxTokens))
					if got := encode(facts, AddThinkingFor(facts, model, effort, budget, maxTokens, Thinking{})); got != want {
						t.Errorf("%s %q %d %d with no facts: %s; want AddThinking's %s", model, effort, budget, maxTokens, got, want)
					}
				}
			}
		}
	}
	for _, tc := range []struct {
		name              string
		model, effort     string
		budget, maxTokens int
		facts             Thinking
		want              string
	}{
		{"a budget, the default", "claude-sonnet-5", "high", 0, 8192, Thinking{Known: true, BudgetMin: 1024, BudgetMax: 32000},
			`{"body":{"thinking":{"budget_tokens":4096,"type":"enabled"}},"max_tokens":8192}`},
		{"a budget cut to its maximum", "claude-sonnet-5", "", 64000, 8192, Thinking{Known: true, BudgetMin: 1024, BudgetMax: 32000},
			`{"body":{"thinking":{"budget_tokens":32000,"type":"enabled"}},"max_tokens":36096}`},
		{"a budget raised to its minimum", "claude-sonnet-5", "", 500, 8192, Thinking{Known: true, BudgetMin: 1024, BudgetMax: 32000},
			`{"body":{"thinking":{"budget_tokens":1024,"type":"enabled"}},"max_tokens":8192}`},
		{"efforts: adaptive, with the effort", "claude-haiku-4-5", "high", 10000, 8192, Thinking{Known: true, Adaptive: true, EffortOnly: true},
			`{"body":{"output_config":{"effort":"high"},"thinking":{"type":"adaptive"}},"max_tokens":8192}`},
		{"a toggle alone: adaptive, no effort", "claude-sonnet-5", "high", 0, 8192, Thinking{Known: true, Adaptive: true},
			`{"body":{"thinking":{"type":"adaptive"}},"max_tokens":8192}`},
		{"the raise stops at the output limit", "claude-haiku-4-5", "", 6000, 4096, Thinking{Known: true, MaxOutput: 8000},
			`{"body":{"thinking":{"budget_tokens":6000,"type":"enabled"}},"max_tokens":8000}`},
		{"a budget at the limit stays under it", "claude-haiku-4-5", "", 9000, 4096, Thinking{Known: true, MaxOutput: 8000},
			`{"body":{"thinking":{"budget_tokens":7999,"type":"enabled"}},"max_tokens":8000}`},
		{"the limit with the rule's shape", "claude-haiku-4-5", "", 6000, 4096, Thinking{MaxOutput: 8000},
			`{"body":{"thinking":{"budget_tokens":6000,"type":"enabled"}},"max_tokens":8000}`},
	} {
		body := map[string]any{}
		if got := encode(body, AddThinkingFor(body, tc.model, tc.effort, tc.budget, tc.maxTokens, tc.facts)); got != tc.want {
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
