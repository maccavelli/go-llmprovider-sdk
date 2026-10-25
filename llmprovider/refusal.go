package llmprovider

import (
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
