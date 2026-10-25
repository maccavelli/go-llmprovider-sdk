package llmprovider

import (
	"context"
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

// TestProbeReasoning_0062 (0062-MADR D7): ProbeReasoning sends one request,
// to the model named or the provider's own, at EffortMax with a one-line
// prompt, and returns the refusals of its WarningReasoningLearned warnings, in
// order, and no other warning's; none when the model refused nothing; and
// Generate's error, with no refusals, when the request fails.
func TestProbeReasoning_0062(t *testing.T) {
	max1 := ReasoningRefusal{Model: "m", Effort: EffortMax}
	shape := ReasoningRefusal{Model: "m", Shape: ThinkingAdaptive}
	failed := errors.New("failed")
	for _, tc := range []struct {
		name  string
		model string
		resp  *Response
		err   error
		want  []ReasoningRefusal
		fails error
	}{
		{"refused max and a shape", "m", &Response{Warnings: []Warning{
			{Kind: WarningReasoningLearned, Model: "m", Refused: max1},
			{Kind: WarningEffortDegraded, Model: "m"},
			{Kind: WarningReasoningLearned, Model: "m", Refused: shape}}}, nil, []ReasoningRefusal{max1, shape}, nil},
		{"refused nothing", "", &Response{Warnings: []Warning{{Kind: WarningEffortDegraded}}}, nil, nil, nil},
		{"failed", "m", nil, failed, nil, failed},
		{"no response", "m", nil, nil, nil, ErrIncomplete},
	} {
		var sent []*Request
		p := &stubProvider{id: "stub", gen: func(_ context.Context, req *Request) (*Response, error) {
			sent = append(sent, req)
			return tc.resp, tc.err
		}}
		got, err := ProbeReasoning(t.Context(), p, tc.model)
		if !slices.Equal(got, tc.want) || (tc.fails == nil) != (err == nil) || (tc.fails != nil && !errors.Is(err, tc.fails)) {
			t.Errorf("%s: %v, %v; want %v, %v", tc.name, got, err, tc.want, tc.fails)
		}
		if len(sent) != 1 {
			t.Fatalf("%s: %d request(s); want one", tc.name, len(sent))
		}
		req := sent[0]
		msg, _ := req.Input[0].(MessageItem)
		if req.Model != tc.model || req.Reasoning == nil || *req.Reasoning != (Reasoning{Effort: EffortMax}) ||
			len(req.Input) != 1 || msg.Role != RoleUser || msg.Text != probePrompt || len(req.Tools) != 0 {
			t.Errorf("%s: sent %+v; want one user message to %q at EffortMax, no tools", tc.name, req, tc.model)
		}
	}
}
