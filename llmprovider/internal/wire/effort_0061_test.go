package wire

import (
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestResolveMax_Rules_0061 (0061-MADR D3, Q9 (a)): EffortMax is sent as max
// where the facts list it or list nothing, and otherwise as the highest
// listed effort below it, degraded.
func TestResolveMax_Rules_0061(t *testing.T) {
	efforts := func(names ...string) []llmprovider.Effort {
		out := make([]llmprovider.Effort, len(names))
		for i, n := range names {
			out[i] = llmprovider.Effort(n)
		}
		return out
	}
	for _, tc := range []struct {
		name     string
		efforts  []llmprovider.Effort
		known    bool
		want     llmprovider.Effort
		degraded bool
	}{
		{"facts with max (rule 1)", efforts("low", "medium", "high", "xhigh", "max"), true, "max", false},
		{"facts to xhigh (rule 2)", efforts("low", "medium", "high", "xhigh"), true, "xhigh", true},
		{"facts to high (rule 2, Q9)", efforts("low", "medium", "high"), true, "high", true},
		{"facts low, high and max (rule 1)", efforts("low", "high", "max"), true, "max", false},
		{"facts low only (rule 2)", efforts("low"), true, "low", true},
		{"facts in another order (rule 2)", efforts("high", "low", "medium"), true, "high", true},
		{"no facts (rule 3)", nil, false, "max", false},
		{"facts known, listing none (rule 3)", nil, true, "max", false},
		{"facts listing none of the four (rule 3)", efforts("none", "minimal"), true, "max", false},
	} {
		got, degraded := ResolveMax(tc.efforts, tc.known)
		if got != tc.want || degraded != tc.degraded {
			t.Errorf("%s: ResolveMax(%v, %t) = %q, %t; want %q, %t", tc.name, tc.efforts, tc.known, got, degraded,
				tc.want, tc.degraded)
		}
	}
}

// TestMaxEffort_Warning_0061 (0061-MADR D3): an effort other than max passes
// unchanged and unwarned; max sent lower carries one WarningEffortDegraded
// naming the model and what was sent.
func TestMaxEffort_Warning_0061(t *testing.T) {
	low := []llmprovider.Effort{llmprovider.EffortLow}
	if got, warns := MaxEffort("m", llmprovider.EffortXHigh, low, true); got != llmprovider.EffortXHigh || warns != nil {
		t.Errorf("xhigh on a model listing low: %q, %v; want xhigh unchanged, no warning", got, warns)
	}
	if got, warns := MaxEffort("m", llmprovider.EffortMax, nil, false); got != llmprovider.EffortMax || warns != nil {
		t.Errorf("max with no facts: %q, %v; want max, no warning", got, warns)
	}
	got, warns := MaxEffort("m", llmprovider.EffortMax, low, true)
	want := llmprovider.Warning{Kind: llmprovider.WarningEffortDegraded, Model: "m",
		Detail: "reasoning effort max is not offered for m; sent low"}
	if got != llmprovider.EffortLow || len(warns) != 1 || warns[0] != want {
		t.Errorf("max on a model listing low: %q, %v; want low, [%v]", got, warns, want)
	}
}
