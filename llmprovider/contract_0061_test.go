package llmprovider

import (
	"errors"
	"testing"
)

// TestEffortMax_Valid_0061 (0061-MADR D3): EffortMax is "max", a request and
// WithReasoning take it, an effort no constant names is still refused, and
// WarningEffortDegraded is "effort_degraded".
func TestEffortMax_Valid_0061(t *testing.T) {
	if EffortMax != "max" {
		t.Errorf("EffortMax = %q; want %q", EffortMax, "max")
	}
	req := &Request{Input: []Item{MessageItem{Role: RoleUser, Text: "hi"}}, Reasoning: &Reasoning{Effort: EffortMax}}
	if err := req.validate(); err != nil {
		t.Errorf("a request at EffortMax: %v; want valid", err)
	}
	if err := (&Reasoning{Effort: EffortMax}).check(); err != nil {
		t.Errorf("WithReasoning's check of EffortMax: %v; want valid", err)
	}
	if err := (&Reasoning{Effort: "maximum"}).check(); !errors.Is(err, ErrInvalidRequest) {
		t.Errorf(`an effort of "maximum": %v; want ErrInvalidRequest`, err)
	}
	if WarningEffortDegraded != "effort_degraded" {
		t.Errorf("WarningEffortDegraded = %q; want %q", WarningEffortDegraded, "effort_degraded")
	}
}
