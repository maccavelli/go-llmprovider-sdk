package llmprovider

import (
	"errors"
	"slices"
	"testing"
)

// TestSharedOption_0061 (0061-MADR, amendment "Phase 4's options"): a shared
// option is taken by every id, a built-in's or another's, its value in
// Values in order with the scoped options'; a scoped option is still refused
// by another id; inside For(id, …) a shared option applies to id alone.
func TestSharedOption_0061(t *testing.T) {
	for _, id := range []ProviderID{ProviderClaude, ProviderOllama, "third-party"} {
		st, err := ResolveOptions(id, []Option{SharedOption("test.Shared", 1), ScopedOptionFor([]ProviderID{id}, "test.Scoped", 2),
			SharedOption("test.Shared", 3)})
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		if got := st.Values(); !slices.Equal(got, []any{1, 2, 3}) {
			t.Errorf("%s: Values = %v; want [1 2 3]", id, got)
		}
	}
	if _, err := ResolveOptions("third-party", []Option{ScopedOption(ProviderClaude, "test.Scoped", 1)}); !errors.Is(err,
		ErrInvalidRequest) {
		t.Errorf("a scoped option for another id: %v; want ErrInvalidRequest", err)
	}
	for _, tc := range []struct {
		id   ProviderID
		want []any
	}{{"a", []any{"for a"}}, {"b", nil}} {
		st, err := ResolveOptions(tc.id, []Option{For("a", SharedOption("test.Shared", "for a"))})
		if err != nil || !slices.Equal(st.Values(), tc.want) {
			t.Errorf("For(a, shared) built as %s: %v, %v; want %v", tc.id, st.Values(), err, tc.want)
		}
	}
}
