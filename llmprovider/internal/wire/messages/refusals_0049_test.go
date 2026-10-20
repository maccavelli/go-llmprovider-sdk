package messages

import (
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestRefusals_KeptApart_0049 (0049-MADR D3): a policy learns a refusal into
// its own memo, which another memo and the process's do not see; a policy
// with no memo learns into the process's.
func TestRefusals_KeptApart_0049(t *testing.T) {
	t.Cleanup(forgetRefusals)
	forgetRefusals()
	refusal := &llmprovider.APIError{Status: 400, Code: "invalid_request_error", Message: refusalMessage}
	refuse := llmprovider.ForcedChoiceRefuse
	a, b := NewRefusals(), NewRefusals()
	const model = "claude-future-0049"
	if resend, err := NewRefusalRetry(NewPolicy(refuse, nil, a).For("claude", model), llmprovider.ForceTool(choiceTool)).Check(refusal); !resend || err != nil {
		t.Fatalf("Check = %v, %v; want the refusal learned and the request resent", resend, err)
	}
	for _, c := range []struct {
		name string
		memo *Refusals
		want bool
	}{{"a", a, true}, {"b", b, false}, {"the process's", nil, false}} {
		if got := NewPolicy(refuse, nil, c.memo).For("claude", model).refuses(); got != c.want {
			t.Errorf("%s: refuses = %v, want %v", c.name, got, c.want)
		}
	}
	const other = "claude-future-0049-process"
	_, _ = NewRefusalRetry(NewPolicy(refuse, nil, nil).For("claude", other), llmprovider.ForceTool(choiceTool)).Check(refusal)
	if !refusesForcedChoice(nil, "claude", other) || a.learned("claude", other) {
		t.Error("a policy with no memo did not learn into the process's alone")
	}
}
