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
	learnRefusal("claude", "Claude-Future-6")
	if !refusesForcedChoice("claude", "claude-future-6") {
		t.Error("a learned refusal is not remembered, in any case")
	}
	if refusesForcedChoice("opencode-zen", "claude-future-6") {
		t.Error("a refusal learned on claude holds for opencode-zen")
	}
	for i := range learnedRefusalsLimit {
		learnRefusal("claude", fmt.Sprintf("model-%d", i))
	}
	if refusesForcedChoice("claude", "claude-future-6") {
		t.Errorf("the oldest refusal is kept past %d entries", learnedRefusalsLimit)
	}
	if !refusesForcedChoice("claude", fmt.Sprintf("model-%d", learnedRefusalsLimit-1)) {
		t.Error("the newest refusal is dropped")
	}
}

// TestSendLearning_0034 (0034-MADR D3): a forced request refused for forcing
// is learned and sent once more; any other result is returned as it is.
func TestSendLearning_0034(t *testing.T) {
	t.Cleanup(forgetRefusals)
	refusal := &llmprovider.APIError{Status: 400, Code: "invalid_request_error", Message: refusalMessage}
	other := &llmprovider.APIError{Status: 400, Code: "invalid_request_error", Message: "max_tokens: too large"}
	forced := llmprovider.ForceTool("get_weather")
	for _, c := range []struct {
		name    string
		choice  llmprovider.ToolChoice
		replies []error
		sends   int
		learned bool
		err     error
	}{
		{"refused, then answered", forced, []error{refusal, nil}, 2, true, nil},
		{"refused twice", forced, []error{refusal, refusal}, 2, true, refusal},
		{"another 400", forced, []error{other}, 1, false, other},
		{"not forced", llmprovider.ToolChoiceAuto, []error{refusal}, 1, false, refusal},
		{"required, refused, then answered", llmprovider.ToolChoiceRequired, []error{refusal, nil}, 2, true, nil},
	} {
		forgetRefusals()
		sends := 0
		_, err := SendLearning("claude", "claude-future-6", c.choice, func() (int, error) {
			sends++
			if sends > len(c.replies) {
				return sends, errors.New("a send past the replies the case expects")
			}
			return sends, c.replies[sends-1]
		})
		if sends != c.sends || !errors.Is(err, c.err) {
			t.Errorf("%s: %d sends, err %v; want %d, %v", c.name, sends, err, c.sends, c.err)
		}
		if got := refusesForcedChoice("claude", "claude-future-6"); got != c.learned {
			t.Errorf("%s: learned = %v, want %v", c.name, got, c.learned)
		}
	}
	// A model already known to refuse is sent degraded; a refusal of that is
	// returned, with no resend.
	forgetRefusals()
	sends := 0
	_, err := SendLearning("claude", "claude-sonnet-5-5", forced, func() (int, error) { sends++; return 0, refusal })
	if sends != 1 || !errors.Is(err, refusal) {
		t.Errorf("a table model: %d sends, err %v; want 1 and the refusal", sends, err)
	}
}

// forgetRefusals empties the memo between tests.
func forgetRefusals() {
	learned.mu.Lock()
	defer learned.mu.Unlock()
	learned.order, learned.set = nil, map[string]struct{}{}
}
