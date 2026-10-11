package grok

import "testing"

// TestGrokEffortMenus_0033 (0033-MADR D4): grok-4.7 and grok-4.3 take low,
// medium, high and xhigh, as xAI's listing reports and each answered on
// 2026-10-11 (0033-PLAN-refresh-the-curated-catalogs.md Phase 1, step 4).
// No effort sends high, as on every menu.
func TestGrokEffortMenus_0033(t *testing.T) {
	checkEfforts(t, []effortCase{
		{"grok-4.7", "low", "low"},
		{"grok-4.7", "medium", "medium"},
		{"grok-4.7", "high", "high"},
		{"grok-4.7", "xhigh", "xhigh"},
		{"grok-4.7", "", "high"},
		{"grok-4.3", "low", "low"},
		{"grok-4.3", "medium", "medium"},
		{"grok-4.3", "high", "high"},
		{"grok-4.3", "xhigh", "xhigh"},
		{"grok-4.3", "", "high"},
	})
}
