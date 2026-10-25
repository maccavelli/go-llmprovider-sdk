package compatid

import "testing"

// TestOfAndFor_0061 (0061-MADR, amendment "chatcompat, provider-agnostic",
// E1): Of is the last Value among values, and reports none without one; For
// compares it without case.
func TestOfAndFor_0061(t *testing.T) {
	values := []any{"other", Value("first"), 3, Value("LMStudio")}
	if got, ok := Of(values); !ok || got != "LMStudio" {
		t.Errorf("Of = %q, %t; want the last, LMStudio", got, ok)
	}
	if got, ok := Of([]any{"lmstudio"}); ok || got != "" {
		t.Errorf("Of without a Value = %q, %t; want none", got, ok)
	}
	for _, tc := range []struct {
		id   string
		want bool
	}{{"lmstudio", true}, {"LMSTUDIO", true}, {"first", false}, {"", false}} {
		if got := For(values, tc.id); got != tc.want {
			t.Errorf("For(%q) = %t; want %t", tc.id, got, tc.want)
		}
	}
	if For(nil, "") {
		t.Error("For with no values matched an empty id")
	}
}
