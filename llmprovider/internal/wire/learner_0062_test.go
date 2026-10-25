package wire

import (
	"errors"
	"slices"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// refusing is a Matcher that reads every error as r, the sent effort filled in
// when r names none and no shape.
func refusing(r Refusal) Matcher {
	return func(_ error, sent llmprovider.Effort) (Refusal, bool) {
		if r.Effort == "" && r.Shape == "" && !r.Field {
			r.Effort = sent
		}
		return r, true
	}
}

// TestLearner_Shape_0062 (0062-MADR D3, D5, D6): D3's shape ladder in one
// Learner. The preferred shape is sent until the model refuses it; a refusal
// of the shape sent is learned, warned with Refused naming it, and resent with
// the other; a refusal of both is returned, and the preferred shape is sent;
// the caller's refusals of the model, in any case, seed the ladder, and
// another model's do not; a refusal of a shape not sent is returned.
func TestLearner_Shape_0062(t *testing.T) {
	adaptive, budget := llmprovider.ThinkingAdaptive, llmprovider.ThinkingBudget
	refused := errors.New("refused")
	memo := NewLearned()

	l := NewLearner(memo, "p", "u", "m", nil, refusing(Refusal{Shape: adaptive}))
	if got := l.Shape(adaptive); got != adaptive {
		t.Fatalf("nothing refused: %s; want adaptive", got)
	}
	if resend, err := l.Check(refused); !resend || err != nil {
		t.Fatalf("adaptive refused: %t, %v; want a resend", resend, err)
	}
	if got := l.Shape(adaptive); got != budget {
		t.Errorf("adaptive learned: %s; want budget", got)
	}
	_, warnings := l.Effort("", nil, false)
	want := llmprovider.ReasoningRefusal{Model: "m", Shape: adaptive}
	if len(warnings) != 1 || warnings[0].Kind != llmprovider.WarningReasoningLearned || warnings[0].Refused != want {
		t.Errorf("warnings %+v; want one WarningReasoningLearned of %+v", warnings, want)
	}
	if resend, err := l.Check(refused); resend || !errors.Is(err, refused) {
		t.Errorf("adaptive refused again while budget was sent: %t, %v; want the error", resend, err)
	}
	unsent := NewLearner(NewLearned(), "p", "u", "m", nil, refusing(Refusal{Shape: adaptive}))
	unsent.Shape(budget)
	if resend, err := unsent.Check(refused); resend || !errors.Is(err, refused) {
		t.Errorf("adaptive refused while budget was sent: %t, %v; want the error", resend, err)
	}
	if got := unsent.Shape(adaptive); got != adaptive {
		t.Errorf("a refusal of a shape not sent: %s; want it not learned, adaptive", got)
	}

	again := NewLearner(memo, "p", "u", "M", nil, refusing(Refusal{Shape: budget}))
	if got := again.Shape(adaptive); got != budget {
		t.Errorf("the memo's refusal, the model in another case: %s; want budget", got)
	}
	if resend, err := again.Check(refused); resend || !errors.Is(err, refused) {
		t.Errorf("both refused: %t, %v; want the error", resend, err)
	}
	if got := again.Shape(adaptive); got != adaptive {
		t.Errorf("both refused: %s; want the preferred, adaptive", got)
	}

	passed := []llmprovider.ReasoningRefusal{{Model: " M2 ", Shape: budget}, {Model: "other", Shape: adaptive}}
	fresh := NewLearner(NewLearned(), "p", "u", "m2", passed, nil)
	if got := fresh.Shape(budget); got != adaptive {
		t.Errorf("budget passed back: %s; want adaptive", got)
	}
	if got := fresh.Shape(adaptive); got != adaptive {
		t.Errorf("another model's refusal of adaptive: %s; want adaptive", got)
	}
	if resend, err := fresh.Check(refused); resend || !errors.Is(err, refused) {
		t.Errorf("no matcher: %t, %v; want the error", resend, err)
	}
}

// TestLearner_Effort_0062 (0062-MADR D3 to D6): a refusal of the effort sent
// is learned, with the efforts it lists, and the next build steps down; a
// refusal of the field sends none; a refusal of another effort, or one
// learned already, is returned; with nothing sent, nothing is read.
func TestLearner_Effort_0062(t *testing.T) {
	refused := errors.New("refused")
	high, maxE := llmprovider.EffortHigh, llmprovider.EffortMax

	l := NewLearner(NewLearned(), "p", "u", "m", nil, refusing(Refusal{
		Accepted: []llmprovider.Effort{llmprovider.EffortLow, high}, Listed: true}))
	if resend, err := l.Check(refused); resend || !errors.Is(err, refused) {
		t.Errorf("nothing sent: %t, %v; want the error", resend, err)
	}
	if got, warnings := l.Effort(maxE, nil, false); got != maxE || len(warnings) != 0 {
		t.Fatalf("max, no facts: %s, %v; want max, unwarned", got, warnings)
	}
	if resend, err := l.Check(refused); !resend || err != nil {
		t.Fatalf("max refused: %t, %v; want a resend", resend, err)
	}
	got, warnings := l.Effort(maxE, nil, false)
	if got != high || len(warnings) != 2 || warnings[0].Refused != (llmprovider.ReasoningRefusal{Model: "m", Effort: maxE}) ||
		warnings[1].Kind != llmprovider.WarningEffortDegraded {
		t.Errorf("max learned, low and high listed: %s, %+v; want high, learned and degraded", got, warnings)
	}

	other := NewLearner(NewLearned(), "p", "u", "m", nil, refusing(Refusal{Effort: "xhigh"}))
	other.Effort(maxE, nil, false)
	if resend, err := other.Check(refused); resend || !errors.Is(err, refused) {
		t.Errorf("another effort's refusal: %t, %v; want the error", resend, err)
	}

	memo := NewLearned()
	field := NewLearner(memo, "p", "u", "m", nil, refusing(Refusal{Field: true}))
	field.Effort(high, nil, false)
	if resend, err := field.Check(refused); !resend || err != nil {
		t.Fatalf("the field refused: %t, %v; want a resend", resend, err)
	}
	if got, _ := field.Effort(high, nil, false); got != "" {
		t.Errorf("the field refused: %q; want none", got)
	}
	twice := NewLearner(memo, "p", "u", "m", nil, refusing(Refusal{Field: true}))
	twice.sent = high
	if resend, err := twice.Check(refused); resend || !errors.Is(err, refused) {
		t.Errorf("a refusal learned already: %t, %v; want the error", resend, err)
	}
}

// TestEffortLists_0062 (0062-MADR, amendment "Phase 1's evidence"): a list
// of effort words is read, Effort values only, each once; a list holding
// another word, or none, is not a list of efforts.
func TestEffortLists_0062(t *testing.T) {
	type e = llmprovider.Effort
	for _, tc := range []struct {
		name string
		got  func() ([]e, bool)
		want []e
		ok   bool
	}{
		{"quoted", func() ([]e, bool) { return QuotedEfforts(`'none', 'Low', "high" or 'high'`) },
			[]e{llmprovider.EffortLow, llmprovider.EffortHigh}, true},
		{"quoted, another word", func() ([]e, bool) { return QuotedEfforts(`'auto' or 'detailed'`) }, nil, false},
		{"quoted, none", func() ([]e, bool) { return QuotedEfforts("no list") }, nil, false},
		{"plain", func() ([]e, bool) { return PlainEfforts(" minimal, medium, max.") },
			[]e{llmprovider.EffortMedium, llmprovider.EffortMax}, true},
		{"plain, another word", func() ([]e, bool) { return PlainEfforts("low, turbo") }, nil, false},
	} {
		got, ok := tc.got()
		if ok != tc.ok || !slices.Equal(got, tc.want) {
			t.Errorf("%s: %v, %t; want %v, %t", tc.name, got, ok, tc.want, tc.ok)
		}
	}
}
