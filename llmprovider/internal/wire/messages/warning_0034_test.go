package messages

import (
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestToolChoiceWarning_0034 (0034-MADR D7): a degraded forced choice warns,
// naming the model; a request sent as asked does not.
func TestToolChoiceWarning_0034(t *testing.T) {
	tools := []llmprovider.Tool{{Name: choiceTool}}
	forced := llmprovider.ForceTool(choiceTool)
	for _, c := range []struct {
		name     string
		req      *llmprovider.Request
		thinking bool
		model    string
		warns    bool
		detail   string
	}{
		{"a refusing model, forced", &llmprovider.Request{Tools: tools, ToolChoice: forced}, false, "claude-sonnet-5-5", true, choiceTool},
		{"a refusing model, required", &llmprovider.Request{Tools: tools, ToolChoice: llmprovider.ToolChoiceRequired}, false, "claude-opus-5-5", true, "one of the tools"},
		{"thinking, forced", &llmprovider.Request{Tools: tools, ToolChoice: forced}, true, "claude-sonnet-5", true, "thinking"},
		{"accepting model, forced", &llmprovider.Request{Tools: tools, ToolChoice: forced}, false, "claude-sonnet-5", false, ""},
		{"a refusing model, auto", &llmprovider.Request{Tools: tools}, false, "claude-sonnet-5-5", false, ""},
		{"a refusing model, none", &llmprovider.Request{Tools: tools, ToolChoice: llmprovider.ToolChoiceNone}, false, "claude-sonnet-5-5", false, ""},
		{"no tools", &llmprovider.Request{ToolChoice: forced}, false, "claude-sonnet-5-5", false, ""},
	} {
		w, ok := ToolChoiceWarning(c.req, c.thinking, Forcing{Provider: "claude", Model: c.model})
		if ok != c.warns {
			t.Errorf("%s: warns = %v, want %v", c.name, ok, c.warns)
			continue
		}
		if ok && (w.Kind != llmprovider.WarningForcedChoiceDegraded || w.Model != c.model || !strings.Contains(w.Detail, c.detail)) {
			t.Errorf("%s: warning = %+v, want kind %s, model %s, a detail naming %q", c.name, w, llmprovider.WarningForcedChoiceDegraded, c.model, c.detail)
		}
	}
}
