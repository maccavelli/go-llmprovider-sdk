package messages

import (
	"errors"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestNormalizeModel_0034 (0034-MADR D9): a gateway prefix, Bedrock's prefix
// and version suffix, and a snapshot date are removed, then the id matches
// exactly.
func TestNormalizeModel_0034(t *testing.T) {
	for id, want := range map[string]string{
		"anthropic/claude-sonnet-5-5":    "claude-sonnet-5-5",
		"anthropic.claude-opus-5-5-v1:0": "claude-opus-5-5",
		"claude-fable-5-1-20260901":      "claude-fable-5-1",
		"claude-sonnet-5-5@20260928":     "claude-sonnet-5-5",
		"Claude-Sonnet-5":                "claude-sonnet-5",
		"claude-sonnet-5-55":             "claude-sonnet-5-55",
		"claude-haiku-4-5-20251001":      "claude-haiku-4-5",
		"anthropic.claude-mythos-5-1-v2": "claude-mythos-5-1",
	} {
		if got := normalizeModel(id); got != want {
			t.Errorf("normalizeModel(%q) = %q, want %q", id, got, want)
		}
	}
	for _, id := range []string{"anthropic/claude-sonnet-5-5", "anthropic.claude-opus-5-5-v1:0", "claude-fable-5-1-20260901", "claude-sonnet-5-5@20260928"} {
		if !refusesForcedChoice(nil, "claude", id) {
			t.Errorf("refusesForcedChoice(%q) = false: an alias of a table id", id)
		}
	}
	for _, id := range []string{"claude-sonnet-5-55", "claude-sonnet-5"} {
		if refusesForcedChoice(nil, "claude", id) {
			t.Errorf("refusesForcedChoice(%q) = true", id)
		}
	}
}

// TestForcing_Layers_0034 (0034-MADR D9): the caller's override, then the
// table, then a learned refusal, then unknown, which accepts.
func TestForcing_Layers_0034(t *testing.T) {
	t.Cleanup(forgetRefusals)
	forgetRefusals()
	processRefusals.learn("claude", "claude-learned-6")
	policy := NewPolicy(llmprovider.ForcedChoiceRefuse, map[string]llmprovider.Support{
		"claude-sonnet-5-5": llmprovider.Supported, "anthropic/claude-future-6": llmprovider.Unsupported,
		"claude-learned-6": llmprovider.Supported,
	}, nil)
	for model, want := range map[string]bool{
		"claude-sonnet-5-5": false, // the override beats the table
		"claude-future-6":   true,  // the override marks an unknown model, its key normalised
		"claude-learned-6":  false, // the override beats a learned refusal
		"claude-opus-5-5":   true,  // the table
		"claude-unknown-7":  false, // unknown accepts
	} {
		if got := policy.For("claude", model).refuses(); got != want {
			t.Errorf("%s: refuses = %v, want %v", model, got, want)
		}
	}
	if !(Forcing{Provider: "claude", Model: "claude-learned-6"}).refuses() {
		t.Error("a learned refusal, with no override, does not refuse")
	}
}

// TestCheckForcing_0034 (0034-MADR D10; R11): by default, ToolChoiceRequired
// on a model that refuses forcing fails before the network, with an error
// matching ErrUnsupported; a degrading policy, a named tool, an accepting
// model and a request without tools pass.
func TestCheckForcing_0034(t *testing.T) {
	tools := []llmprovider.Tool{{Name: choiceTool}}
	refuse, degrade := NewPolicy(llmprovider.ForcedChoiceRefuse, nil, nil), NewPolicy(llmprovider.ForcedChoiceDegrade, nil, nil)
	err := CheckForcing(&llmprovider.Request{Tools: tools, ToolChoice: llmprovider.ToolChoiceRequired}, refuse.For("claude", "claude-sonnet-5-5"))
	if !errors.Is(err, llmprovider.ErrUnsupported) || !strings.Contains(err.Error(), "claude-sonnet-5-5") ||
		!strings.Contains(err.Error(), "ForcedChoiceDegrade") {
		t.Errorf("err = %v, want ErrUnsupported naming the model and the opt-in", err)
	}
	for name, c := range map[string]struct {
		req *llmprovider.Request
		f   Forcing
	}{
		"degrade":            {&llmprovider.Request{Tools: tools, ToolChoice: llmprovider.ToolChoiceRequired}, degrade.For("claude", "claude-sonnet-5-5")},
		"a named tool":       {&llmprovider.Request{Tools: tools, ToolChoice: llmprovider.ForceTool(choiceTool)}, refuse.For("claude", "claude-sonnet-5-5")},
		"an accepting model": {&llmprovider.Request{Tools: tools, ToolChoice: llmprovider.ToolChoiceRequired}, refuse.For("claude", "claude-sonnet-5")},
		"no tools":           {&llmprovider.Request{ToolChoice: llmprovider.ToolChoiceRequired}, refuse.For("claude", "claude-sonnet-5-5")},
	} {
		if err := CheckForcing(c.req, c.f); err != nil {
			t.Errorf("%s: err = %v, want nil", name, err)
		}
	}
}
