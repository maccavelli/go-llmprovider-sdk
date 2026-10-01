package grok

import "testing"

// Ported from llmprovider's grok_effort_menu_test.go and
// grok_reasoning_test.go (0015-PLAN S7).

type effortCase struct{ model, effort, want string }

func checkEfforts(t *testing.T, cases []effortCase) {
	t.Helper()
	for _, tc := range cases {
		if got := clampReasoningEffort(tc.model, tc.effort); got != tc.want {
			t.Errorf("clampReasoningEffort(%q, %q) = %q, want %q", tc.model, tc.effort, got, tc.want)
		}
	}
}

// TestGrokEffortMenus_ClampToCLIMenu: the Grok CLI offers grok-4.5 only
// high/medium/low and grok-4.6-build and grok-3-mini only low/high
// (grok-build f0e3be11: xai-grok-models/default_models.json;
// xai-grok-shell/src/agent/config_tests.rs:7958).
func TestGrokEffortMenus_ClampToCLIMenu(t *testing.T) {
	checkEfforts(t, []effortCase{
		{"grok-4.5", "xhigh", "high"},
		{"grok-4.6-build", "xhigh", "high"},
		{"grok-4.6-build", "medium", "low"},
		{"grok-3-mini", "medium", "low"},
		{"grok-3-mini-fast", "medium", "low"},
	})
}

// TestGrokEffortMenus_KeepMenuValues: an effort on the menu is sent as is,
// no effort sends high, and a model without a menu sends none.
func TestGrokEffortMenus_KeepMenuValues(t *testing.T) {
	checkEfforts(t, []effortCase{
		{"grok-4.6", "xhigh", "xhigh"},
		{"grok-4.6", "medium", "medium"},
		{"grok-4.5", "low", "low"},
		{"grok-4.5", "", "high"},
		{"grok-4.6", "", "high"},
		{"grok-3-mini", "high", "high"},
		{"grok-4", "high", ""},
		{"grok-4.7", "high", ""},
	})
}

// TestGrokClampReasoningEffort pins models without a CLI menu: they send no
// reasoning_effort (MADR 0012 §6).
func TestGrokClampReasoningEffort(t *testing.T) {
	checkEfforts(t, []effortCase{
		{"grok-3", "high", ""},
		{"grok-4", "medium", ""},
		{"grok-4-fast-reasoning", "low", ""},
		{"grok-code-fast-1", "high", ""},
		{"unknown-model", "high", ""},
	})
}
