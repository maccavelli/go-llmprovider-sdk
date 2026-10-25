package llmprovider

import (
	"errors"
	"slices"
	"testing"
)

// TestReasoningRefusals_Option_0062 (0062-MADR D6): WithReasoningRefusals's
// refusals come back from Settings in order, a later call's after an
// earlier's, as a copy; and each refusal no request could use is refused
// with ErrInvalidRequest.
func TestReasoningRefusals_Option_0062(t *testing.T) {
	first := ReasoningRefusal{Model: "m-1", Effort: EffortMax}
	second := ReasoningRefusal{Model: "M-2", Shape: ThinkingBudget}
	third := ReasoningRefusal{Model: "m-3", Effort: EffortXHigh}
	st, err := ResolveOptions(ProviderClaude, []Option{WithAPIKey("k"), WithReasoningRefusals(first, second),
		WithReasoningRefusals(third)})
	if err != nil {
		t.Fatal(err)
	}
	got := st.ReasoningRefusals()
	if want := []ReasoningRefusal{first, second, third}; !slices.Equal(got, want) {
		t.Errorf("ReasoningRefusals = %v; want %v", got, want)
	}
	got[0].Model = "changed"
	if st.ReasoningRefusals()[0].Model != "m-1" {
		t.Error("ReasoningRefusals returned the settings' own slice, not a copy")
	}
	if none, err := ResolveOptions(ProviderClaude, []Option{WithAPIKey("k")}); err != nil || none.ReasoningRefusals() != nil {
		t.Errorf("without the option: %v, %v; want nil", none.ReasoningRefusals(), err)
	}

	for _, tc := range []struct {
		name string
		r    ReasoningRefusal
	}{
		{"no model", ReasoningRefusal{Effort: EffortMax}},
		{"a control character", ReasoningRefusal{Model: "m\n", Effort: EffortMax}},
		{"both", ReasoningRefusal{Model: "m", Effort: EffortMax, Shape: ThinkingAdaptive}},
		{"neither", ReasoningRefusal{Model: "m"}},
		{"an unknown effort", ReasoningRefusal{Model: "m", Effort: "bogus"}},
		{"an unknown shape", ReasoningRefusal{Model: "m", Shape: "bogus"}},
	} {
		if _, err := ResolveOptions(ProviderClaude, []Option{WithAPIKey("k"), WithReasoningRefusals(tc.r)}); !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("%s: %v; want ErrInvalidRequest", tc.name, err)
		}
	}
}
