package wire

import (
	"fmt"
	"slices"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestLadder_Rungs_0062 (0062-MADR D3): the efforts a request may be sent at,
// in order: the effort, each lower one, highest first, each higher one,
// lowest first; less what the facts do not list, when known, and what was
// refused.
func TestLadder_Rungs_0062(t *testing.T) {
	type e = llmprovider.Effort
	low, medium, high, xhigh, maxE := llmprovider.EffortLow, llmprovider.EffortMedium, llmprovider.EffortHigh,
		llmprovider.EffortXHigh, llmprovider.EffortMax
	for _, tc := range []struct {
		name    string
		want    e
		listed  []e
		known   bool
		refused []e
		rungs   []e
	}{
		{"max, no facts", maxE, nil, false, nil, []e{maxE, xhigh, high, medium, low}},
		{"high, no facts: lower first, then higher", high, nil, false, nil, []e{high, medium, low, xhigh, maxE}},
		{"the facts list below max", maxE, []e{low, medium, high}, true, nil, []e{high, medium, low}},
		{"nothing listed below: the higher rungs", low, []e{medium, high}, true, nil, []e{medium, high}},
		{"max refused", maxE, nil, false, []e{maxE}, []e{xhigh, high, medium, low}},
		{"listed, and one refused", medium, []e{low, high}, true, []e{low}, []e{high}},
		{"everything listed refused: none", xhigh, []e{low}, true, []e{low}, nil},
		{"known to list none: none", high, nil, true, nil, nil},
		{"no effort", "", nil, false, nil, nil},
		{"an unknown effort", "bogus", nil, false, nil, nil},
	} {
		if got := Ladder(tc.want, tc.listed, tc.known, tc.refused); !slices.Equal(got, tc.rungs) {
			t.Errorf("%s: %v; want %v", tc.name, got, tc.rungs)
		}
	}
}

// TestMemo_KeyAndBound_0062 (0062-MADR D5): the memo is keyed by provider,
// base URL and model, the model in lower case and the base URL without a
// trailing slash; it holds at most 256 models, the oldest dropped; a nil memo
// is the process's.
func TestMemo_KeyAndBound_0062(t *testing.T) {
	key := LearnedKey("p", "https://h/v1", "Model-A")
	for _, tc := range []struct {
		name string
		key  string
		same bool
	}{
		{"case and a trailing slash", LearnedKey("p", "https://h/v1/", " model-a "), true},
		{"another base URL", LearnedKey("p", "https://other/v1", "model-a"), false},
		{"another provider", LearnedKey("q", "https://h/v1", "model-a"), false},
		{"another model", LearnedKey("p", "https://h/v1", "model-b"), false},
	} {
		if (tc.key == key) != tc.same {
			t.Errorf("%s: key equal %t; want %t", tc.name, tc.key == key, tc.same)
		}
	}

	memo := NewLearned()
	for i := range learnedLimit + 1 {
		memo.learnEffort(LearnedKey("p", "u", fmt.Sprintf("m%d", i)), llmprovider.EffortMax, nil, false)
	}
	for _, tc := range []struct {
		model string
		kept  bool
	}{{"m0", false}, {"m1", true}, {fmt.Sprintf("m%d", learnedLimit), true}} {
		refused, _, _ := memo.efforts(LearnedKey("p", "u", tc.model))
		if kept := slices.Contains(refused, llmprovider.EffortMax); kept != tc.kept {
			t.Errorf("past %d models, %s remembered %t; want %t", learnedLimit, tc.model, kept, tc.kept)
		}
	}

	process := LearnedKey("p", "u", t.Name())
	(*Learned)(nil).learnEffort(process, llmprovider.EffortHigh, nil, false)
	if refused, _, _ := processLearned.efforts(process); !slices.Equal(refused, []llmprovider.Effort{llmprovider.EffortHigh}) {
		t.Errorf("a nil memo learned %v in the process's; want high", refused)
	}
}
