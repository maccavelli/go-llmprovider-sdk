package llmprovider

import "testing"

// TestWarning_0034 (0034-MADR D7): a Response carries typed warnings, and the
// two kinds have stable values.
func TestWarning_0034(t *testing.T) {
	resp := Response{Warnings: []Warning{{Kind: WarningForcedChoiceDegraded, Model: "m", Detail: "d"}}}
	if w := resp.Warnings[0]; w.Kind != "forced_tool_choice_degraded" || w.Model != "m" || w.Detail != "d" {
		t.Errorf("warning = %+v", w)
	}
	if WarningForcedChoiceLearned != "forced_tool_choice_learned" {
		t.Errorf("WarningForcedChoiceLearned = %q", WarningForcedChoiceLearned)
	}
}
