package messages

import (
	"errors"
	"fmt"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// refusalMessage is Anthropic's refusal of a forced tool choice, as 0034-PLAN
// Phase 1 measured it (M2).
const refusalMessage = `tool_choice: type "tool" and "any" are not supported for this model.`

// TestIsForcedChoiceRefusal_0034 (0034-MADR D3): HTTP 400,
// invalid_request_error, and a message naming tool_choice; nothing else.
func TestIsForcedChoiceRefusal_0034(t *testing.T) {
	for _, c := range []struct {
		err  error
		want bool
	}{
		{&llmprovider.APIError{Status: 400, Code: "invalid_request_error", Message: refusalMessage}, true},
		{fmt.Errorf("wrapped: %w", &llmprovider.APIError{Status: 400, Code: "invalid_request_error", Message: refusalMessage}), true},
		{&llmprovider.APIError{Status: 400, Code: "invalid_request_error", Message: "max_tokens: 999999 > 128000"}, false},
		{&llmprovider.APIError{Status: 400, Code: "", Message: refusalMessage}, false},
		{&llmprovider.APIError{Status: 404, Code: "invalid_request_error", Message: refusalMessage}, false},
		{errors.New(refusalMessage), false},
		{nil, false},
	} {
		if got := IsForcedChoiceRefusal(c.err); got != c.want {
			t.Errorf("IsForcedChoiceRefusal(%v) = %v, want %v", c.err, got, c.want)
		}
	}
}

// TestLearnedRefusals_0034 (0034-MADR D3): a learned refusal holds for its
// provider and model only, and the memo keeps at most learnedRefusalsLimit
// entries, dropping the oldest.
func TestLearnedRefusals_0034(t *testing.T) {
	t.Cleanup(forgetRefusals)
	forgetRefusals()
	processRefusals.learn("claude", "Claude-Future-6")
	if !refusesForcedChoice(nil, "claude", "claude-future-6") {
		t.Error("a learned refusal is not remembered, in any case")
	}
	if refusesForcedChoice(nil, "opencode-zen", "claude-future-6") {
		t.Error("a refusal learned on claude holds for opencode-zen")
	}
	for i := range learnedRefusalsLimit {
		processRefusals.learn("claude", fmt.Sprintf("model-%d", i))
	}
	if refusesForcedChoice(nil, "claude", "claude-future-6") {
		t.Errorf("the oldest refusal is kept past %d entries", learnedRefusalsLimit)
	}
	if !refusesForcedChoice(nil, "claude", fmt.Sprintf("model-%d", learnedRefusalsLimit-1)) {
		t.Error("the newest refusal is dropped")
	}
}

// TestRefusalRetry_0034 (0034-MADR D3, narrowed): a refusal from a model no
// layer knows is learned; a named tool is resent, as D8; ToolChoiceRequired
// is resent, degraded, only under ForcedChoiceDegrade, and otherwise
// returns an error matching ErrUnsupported and the *APIError, with no resend.
// Each learning earns a WarningForcedChoiceLearned naming the model. Nothing
// else is resent, learned, or warned of.
func TestRefusalRetry_0034(t *testing.T) {
	t.Cleanup(forgetRefusals)
	refusal := &llmprovider.APIError{Status: 400, Code: "invalid_request_error", Message: refusalMessage}
	other := &llmprovider.APIError{Status: 400, Code: "invalid_request_error", Message: "max_tokens: too large"}
	refuse, degrade := llmprovider.ForcedChoiceRefuse, llmprovider.ForcedChoiceDegrade
	for _, c := range []struct {
		name        string
		model       string
		fallback    llmprovider.ForcedChoiceFallback
		choice      llmprovider.ToolChoice
		err         error
		resend      bool
		unsupported bool
		learned     bool
	}{
		{"named", "claude-future-6", refuse, llmprovider.ForceTool(choiceTool), refusal, true, false, true},
		{"required, refuse", "claude-future-6", refuse, llmprovider.ToolChoiceRequired, refusal, false, true, true},
		{"required, degrade", "claude-future-6", degrade, llmprovider.ToolChoiceRequired, refusal, true, false, true},
		{"another 400", "claude-future-6", refuse, llmprovider.ForceTool(choiceTool), other, false, false, false},
		{"not forced", "claude-future-6", refuse, llmprovider.ToolChoiceAuto, refusal, false, false, false},
		{"a table model", "claude-sonnet-5-5", refuse, llmprovider.ForceTool(choiceTool), refusal, false, false, false},
	} {
		forgetRefusals()
		r := NewRefusalRetry(NewPolicy(c.fallback, nil, nil).For("claude", c.model), c.choice)
		resend, err := r.Check(c.err)
		switch {
		case resend != c.resend:
			t.Errorf("%s: resend = %v, want %v", c.name, resend, c.resend)
		case resend && err != nil:
			t.Errorf("%s: resend with error %v", c.name, err)
		case !resend && c.unsupported && (!errors.Is(err, llmprovider.ErrUnsupported) || !errors.As(err, new(*llmprovider.APIError))):
			t.Errorf("%s: err = %v, want ErrUnsupported wrapping the *APIError", c.name, err)
		case !resend && !c.unsupported && !errors.Is(err, c.err):
			t.Errorf("%s: err = %v, want %v as it came", c.name, err, c.err)
		}
		if got := processRefusals.learned("claude", c.model); got != c.learned {
			t.Errorf("%s: learned = %v, want %v", c.name, got, c.learned)
		}
		w, ok := r.Warning()
		if ok != c.learned || (ok && (w.Kind != llmprovider.WarningForcedChoiceLearned || w.Model != c.model)) {
			t.Errorf("%s: warning %+v, %v; want a learned warning naming %s: %v", c.name, w, ok, c.model, c.learned)
		}
	}
}

// TestRefusalRetry_OverrideWins_0034 (0034-MADR D9): a model the caller said
// accepts a forced choice is not learned from its refusal, nor resent: the
// caller's word stands, and the refusal is returned.
func TestRefusalRetry_OverrideWins_0034(t *testing.T) {
	t.Cleanup(forgetRefusals)
	forgetRefusals()
	refusal := &llmprovider.APIError{Status: 400, Code: "invalid_request_error", Message: refusalMessage}
	f := NewPolicy(llmprovider.ForcedChoiceRefuse, map[string]llmprovider.Support{"claude-future-7": llmprovider.Supported}, nil).
		For("claude", "claude-future-7")
	r := NewRefusalRetry(f, llmprovider.ForceTool(choiceTool))
	resend, err := r.Check(refusal)
	if _, warned := r.Warning(); resend || !errors.Is(err, refusal) || processRefusals.learned("claude", "claude-future-7") || warned {
		t.Errorf("resend %v, err %v, learned %v, warned %v; want the refusal, nothing resent, learned or warned",
			resend, err, processRefusals.learned("claude", "claude-future-7"), warned)
	}
}

// forgetRefusals empties the memo between tests.
func forgetRefusals() {
	processRefusals.mu.Lock()
	defer processRefusals.mu.Unlock()
	processRefusals.order, processRefusals.set = nil, map[string]struct{}{}
}
