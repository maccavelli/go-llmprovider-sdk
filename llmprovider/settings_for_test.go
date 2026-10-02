package llmprovider

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

// maxTokensFor resolves opts for id and returns MaxTokens, failing on error.
func maxTokensFor(t *testing.T, id ProviderID, opts ...Option) int {
	t.Helper()
	st, err := ResolveOptions(id, opts)
	if err != nil {
		t.Fatalf("%s: ResolveOptions: %v", id, err)
	}
	return st.MaxTokens()
}

// TestFor_AppliesOnlyToItsID (0015-MADR D5 step 2): For's options apply when
// building its id, and are skipped for any other.
func TestFor_AppliesOnlyToItsID(t *testing.T) {
	opts := []Option{WithMaxTokens(100), For(ProviderKilo, WithMaxTokens(5))}
	if got := maxTokensFor(t, ProviderKilo, opts...); got != 5 {
		t.Errorf("kilo MaxTokens = %d, want For's 5", got)
	}
	if got := maxTokensFor(t, ProviderOpenAI, opts...); got != 100 {
		t.Errorf("openai MaxTokens = %d, want the baseline's 100", got)
	}
}

// TestFor_WinsOverTheBaselineWhereverItSits: an overlay applies after the
// baseline, so a baseline option placed after it does not beat it.
func TestFor_WinsOverTheBaselineWhereverItSits(t *testing.T) {
	if got := maxTokensFor(t, ProviderKilo, For(ProviderKilo, WithMaxTokens(5)), WithMaxTokens(100)); got != 5 {
		t.Errorf("MaxTokens = %d, want For's 5 over the later baseline", got)
	}
	// Within one overlay, and across two for the same id, the later wins.
	got := maxTokensFor(t, ProviderKilo, For(ProviderKilo, WithMaxTokens(5), WithMaxTokens(6)), For(ProviderKilo, WithMaxTokens(7)))
	if got != 7 {
		t.Errorf("MaxTokens = %d, want the last overlay's 7", got)
	}
}

// TestFor_ScopedOptionInASharedList (D5 step 3): a provider-specific option
// inside For may sit in a list every provider is built from; given bare it is
// still an error from another provider's New.
func TestFor_ScopedOptionInASharedList(t *testing.T) {
	org := ScopedOption(ProviderKilo, "kilo.WithOrganization", "org-1")
	shared := []Option{WithMaxTokens(9), For(ProviderKilo, org)}
	for _, id := range []ProviderID{ProviderOpenAI, ProviderClaude, ProviderOpencodeGo} {
		st, err := ResolveOptions(id, shared)
		if err != nil || len(st.Values()) != 0 {
			t.Errorf("%s: err %v, values %v; want the overlay skipped", id, err, st.Values())
		}
	}
	st, err := ResolveOptions(ProviderKilo, shared)
	if err != nil || !slices.Equal(st.Values(), []any{"org-1"}) {
		t.Fatalf("kilo: err %v, values %v; want [org-1]", err, st.Values())
	}
	if _, err := ResolveOptions(ProviderOpenAI, []Option{org}); !errors.Is(err, ErrInvalidRequest) {
		t.Errorf("bare foreign option: err = %v, want it refused", err)
	}
}

// TestFor_StaysStrictForItsOwnID: an option inside For(kilo, …) that kilo
// does not take is refused when building kilo, as it would be bare.
func TestFor_StaysStrictForItsOwnID(t *testing.T) {
	route := ScopedOption(ProviderOpencodeZen, "opencode.WithRoute", "messages")
	_, err := ResolveOptions(ProviderKilo, []Option{For(ProviderKilo, route)})
	if !errors.Is(err, ErrInvalidRequest) || !strings.Contains(err.Error(), "opencode.WithRoute") {
		t.Fatalf("err = %v, want the foreign option inside For refused", err)
	}
}

// TestFor_Refusals (D5 step 4): a For nested under a different id, and a For
// with no id, are refused by every provider's New, whichever it builds.
func TestFor_Refusals(t *testing.T) {
	for _, c := range []struct {
		name string
		opt  Option
		want string
	}{
		{"nested under a different id", For(ProviderKilo, For(ProviderOpenAI, WithMaxTokens(1))), `For("openai", …) inside For("kilo", …)`},
		{"no id", For("", WithMaxTokens(1)), "For needs a provider id"},
	} {
		for _, id := range []ProviderID{ProviderKilo, ProviderOpenAI, ProviderGemini} {
			_, err := ResolveOptions(id, []Option{c.opt})
			if !errors.Is(err, ErrInvalidRequest) || !strings.Contains(err.Error(), c.want) {
				t.Errorf("%s, building %s: err = %v, want %q", c.name, id, err, c.want)
			}
		}
	}
	// The same id nested is only redundant, and applies.
	if got := maxTokensFor(t, ProviderKilo, For(ProviderKilo, For(ProviderKilo, WithMaxTokens(3)))); got != 3 {
		t.Errorf("same id nested: MaxTokens = %d, want 3", got)
	}
}
