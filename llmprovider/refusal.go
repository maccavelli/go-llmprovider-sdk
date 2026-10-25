package llmprovider

import (
	"context"
	"fmt"
	"slices"
)

// ThinkingShape is how a wire with two forms of thinking is asked to reason:
// adaptively, with an effort, or with a fixed budget (0062-MADR D1, D6). It
// is a string, safe for concurrent use.
type ThinkingShape string

const (
	// ThinkingAdaptive is adaptive thinking: thinking.type "adaptive" on the
	// Messages wire, and a thinkingLevel on generateContent.
	ThinkingAdaptive ThinkingShape = "adaptive"
	// ThinkingBudget is a fixed budget: thinking.type "enabled" with
	// budget_tokens on the Messages wire, and a thinkingBudget on
	// generateContent.
	ThinkingBudget ThinkingShape = "budget"
)

// valid reports whether s is one of the ThinkingShape constants.
func (s ThinkingShape) valid() bool { return s == ThinkingAdaptive || s == ThinkingBudget }

// ReasoningRefusal is a thinking setting a model refused: an effort or a
// shape, never both (0062-MADR D6). A provider that learns one reports it as
// a WarningReasoningLearned's Refused; a caller that keeps it passes it back
// with WithReasoningRefusals, so that a later process never sends it. It is
// comparable, and safe for concurrent use only while no goroutine changes it.
type ReasoningRefusal struct {
	// Model is the model that refused, as the request named it.
	Model string
	// Effort is the refused effort, or empty when Shape is set.
	Effort Effort
	// Shape is the refused shape, or empty when Effort is set.
	Shape ThinkingShape
}

// WithReasoningRefusals passes back refusals that WarningReasoningLearned
// reported, so that the provider never sends a refused setting to that model:
// it chooses as if it had learned them itself (0062-MADR D2, D6). A later
// call adds to an earlier one's. The model is matched in lower case. New
// refuses, with ErrInvalidRequest, a refusal with no model, one with both or
// neither of Effort and Shape, and an unknown effort or shape.
func WithReasoningRefusals(refusals ...ReasoningRefusal) Option {
	return commonOption("WithReasoningRefusals", func(cfg *providerConfig) {
		cfg.ReasoningRefusals = slices.Concat(cfg.ReasoningRefusals, refusals)
	})
}

// probePrompt is ProbeReasoning's one-line request.
const probePrompt = "Reply with the word OK."

// ProbeReasoning finds the thinking settings model refuses on p, by one
// generation at EffortMax with a fixed one-line prompt (0062-MADR D7), and
// returns the refusals its WarningReasoningLearned warnings carry, in order,
// or nil when the model refused nothing. An empty model is p's own. A
// built-in provider resends a refused setting within the request, as it does
// for any request, so that the one call also finds a refused thinking shape,
// and remembers each refusal for the rest of the process. A caller keeps the
// refusals, and passes them back with WithReasoningRefusals on its next
// start. The accepted send is billed; whether a refused one is billed is not
// known. It sends whatever WithModelProbes says, as the caller asked for it.
// On an error, Generate's is returned with no refusals: an answer that is
// empty or cut at the output limit carries no warnings, though p still
// remembers what it learned.
func ProbeReasoning(ctx context.Context, p Provider, model string) ([]ReasoningRefusal, error) {
	resp, err := generate(ctx, p, &Request{Model: model, Input: []Item{MessageItem{Role: RoleUser, Text: probePrompt}},
		Reasoning: &Reasoning{Effort: EffortMax}})
	if err != nil {
		return nil, err
	}
	var refused []ReasoningRefusal
	for _, w := range resp.Warnings {
		if w.Kind == WarningReasoningLearned {
			refused = append(refused, w.Refused)
		}
	}
	return refused, nil
}

// ReasoningRefusals is a copy of the refusals WithReasoningRefusals passed
// back, in order, or nil. A provider reads them before each request
// (0062-MADR D2).
func (st *Settings) ReasoningRefusals() []ReasoningRefusal {
	return slices.Clone(st.s.cfg.ReasoningRefusals)
}

// checkReasoningRefusals refuses a passed-back refusal no request could use
// (R23).
func (cfg providerConfig) checkReasoningRefusals() error {
	for i, r := range cfg.ReasoningRefusals {
		set := "both"
		if r.Effort == "" {
			set = "neither"
		}
		switch {
		case r.Model == "":
			return fmt.Errorf("%w: WithReasoningRefusals: refusal %d names no model", ErrInvalidRequest, i)
		case hasControl(r.Model):
			return fmt.Errorf("%w: WithReasoningRefusals: refusal %d's model holds a control character", ErrInvalidRequest, i)
		case (r.Effort == "") == (r.Shape == ""):
			return fmt.Errorf("%w: WithReasoningRefusals: refusal %d (%s) sets %s of Effort and Shape; a refusal is of one",
				ErrInvalidRequest, i, r.Model, set)
		case r.Effort != "" && !r.Effort.valid():
			return fmt.Errorf("%w: WithReasoningRefusals: refusal %d (%s): unknown effort %q", ErrInvalidRequest, i, r.Model, r.Effort)
		case r.Shape != "" && !r.Shape.valid():
			return fmt.Errorf("%w: WithReasoningRefusals: refusal %d (%s): unknown shape %q", ErrInvalidRequest, i, r.Model, r.Shape)
		}
	}
	return nil
}
