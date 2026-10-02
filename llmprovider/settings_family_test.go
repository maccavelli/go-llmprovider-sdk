package llmprovider

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

// TestScopedOptionFor_EachListedProviderTakesIt (0015-MADR amendment "the
// OpenCode family"): an option scoped to several ids reaches each of them,
// and every other New refuses it, naming the ids.
func TestScopedOptionFor_EachListedProviderTakesIt(t *testing.T) {
	ids := []ProviderID{ProviderOpencodeZen, ProviderOpencodeGo}
	opt := ScopedOptionFor(ids, "opencode.WithRoute", "messages")
	for _, id := range ids {
		st, err := ResolveOptions(id, []Option{opt})
		if err != nil {
			t.Fatalf("%s: ResolveOptions: %v", id, err)
		}
		if got := st.Values(); !slices.Equal(got, []any{"messages"}) {
			t.Errorf("%s: Values() = %v, want [messages]", id, got)
		}
	}
	_, err := ResolveOptions(ProviderKilo, []Option{opt})
	if !errors.Is(err, ErrInvalidRequest) ||
		!strings.Contains(err.Error(), `opencode.WithRoute is for providers "opencode-zen", "opencode-go", not "kilo"`) {
		t.Fatalf("kilo: err = %v, want the option refused and its ids named", err)
	}
}

// TestScopedOptionFor_EmptyListIsRefusedEverywhere: an option scoped to no id
// is no provider's, so every New refuses it rather than treating it as common.
func TestScopedOptionFor_EmptyListIsRefusedEverywhere(t *testing.T) {
	for _, id := range []ProviderID{ProviderOpenAI, ProviderOpencodeZen} {
		if _, err := ResolveOptions(id, []Option{ScopedOptionFor(nil, "x.WithNothing", 1)}); !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("%s: err = %v, want ErrInvalidRequest", id, err)
		}
	}
}

// TestScopedOptionFor_CopiesTheList: changing the caller's slice afterwards
// does not change the option's scope.
func TestScopedOptionFor_CopiesTheList(t *testing.T) {
	ids := []ProviderID{ProviderOpencodeZen}
	opt := ScopedOptionFor(ids, "opencode.WithRoute", "messages")
	ids[0] = ProviderKilo
	if _, err := ResolveOptions(ProviderOpencodeZen, []Option{opt}); err != nil {
		t.Fatalf("opencode-zen after the caller's slice changed: %v", err)
	}
}

// TestResolveOptions_ModelMetadataURLIsCommon: the metadata URL reaches
// every provider (0015-MADR amendment "the OpenCode family").
func TestResolveOptions_ModelMetadataURLIsCommon(t *testing.T) {
	st, err := ResolveOptions(ProviderOpencodeGo, []Option{WithModelMetadataURL("http://meta.invalid/api.json")})
	if err != nil {
		t.Fatalf("ResolveOptions: %v", err)
	}
	if got := st.ModelMetadataURL(); got != "http://meta.invalid/api.json" {
		t.Errorf("ModelMetadataURL() = %q, want the option's URL", got)
	}
	if st, _ := ResolveOptions(ProviderOpenAI, nil); st.ModelMetadataURL() != "" {
		t.Errorf("default ModelMetadataURL() = %q, want empty", st.ModelMetadataURL())
	}
}
