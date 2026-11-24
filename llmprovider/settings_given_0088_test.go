package llmprovider_test

import (
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestSettings_MaxTokensGiven_0088 (0088-MADR ST35, ST36): through the
// exported API, as a provider outside the module reads it, MaxTokensGiven is
// false with no WithMaxTokens, true for WithMaxTokens(8192), the default's own
// number, and, for For(claude, WithMaxTokens(5000)), true when resolving for
// Claude and false for Kilo.
func TestSettings_MaxTokensGiven_0088(t *testing.T) {
	cases := []struct {
		name string
		id   llmprovider.ProviderID
		opts []llmprovider.Option
		want bool
		max  int
	}{
		{"none", llmprovider.ProviderClaude, nil, false, 8192},
		{"given 8192", llmprovider.ProviderClaude, []llmprovider.Option{llmprovider.WithMaxTokens(8192)}, true, 8192},
		{"For claude, resolved for claude", llmprovider.ProviderClaude,
			[]llmprovider.Option{llmprovider.For(llmprovider.ProviderClaude, llmprovider.WithMaxTokens(5000))}, true, 5000},
		{"For claude, resolved for kilo", llmprovider.ProviderKilo,
			[]llmprovider.Option{llmprovider.For(llmprovider.ProviderClaude, llmprovider.WithMaxTokens(5000))}, false, 8192},
	}
	for _, c := range cases {
		st, err := llmprovider.ResolveOptions(c.id, c.opts)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got := st.MaxTokensGiven(); got != c.want {
			t.Errorf("%s: MaxTokensGiven = %t; want %t", c.name, got, c.want)
		}
		if got := st.MaxTokens(); got != c.max {
			t.Errorf("%s: MaxTokens = %d; want %d", c.name, got, c.max)
		}
	}
}
