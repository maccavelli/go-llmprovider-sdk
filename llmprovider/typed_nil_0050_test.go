package llmprovider_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// tokenFunc0050 is a caller's own source, of a func type: a nil one is a
// typed nil.
type tokenFunc0050 func(context.Context) (llmprovider.Token, error)

func (f tokenFunc0050) Token(ctx context.Context) (llmprovider.Token, error) { return f(ctx) }

// TestResolveOptions_TypedNilSources_0050 (0050-MADR D3): a typed-nil source
// of any type is refused with ErrInvalidRequest, naming its type, and no
// panic; a nil interface is still no source.
func TestResolveOptions_TypedNilSources_0050(t *testing.T) {
	for _, c := range []struct {
		src  llmprovider.TokenSource
		name string
	}{
		{(*llmprovider.StaticToken)(nil), "*llmprovider.StaticToken"},
		{(*llmprovider.CommandToken)(nil), "*llmprovider.CommandToken"},
		{tokenFunc0050(nil), "llmprovider_test.tokenFunc0050"},
	} {
		err := noPanic0049(t, c.name, func() error {
			_, err := llmprovider.ResolveOptions(llmprovider.ProviderClaude, []llmprovider.Option{llmprovider.WithTokenSource(c.src)})
			return err
		})
		if !errors.Is(err, llmprovider.ErrInvalidRequest) || !strings.Contains(err.Error(), "a nil "+c.name) {
			t.Errorf("a nil %s: ResolveOptions = %v; want ErrInvalidRequest naming its type", c.name, err)
		}
	}
	st, err := llmprovider.ResolveOptions(llmprovider.ProviderClaude, []llmprovider.Option{llmprovider.WithTokenSource(nil)})
	if err != nil || st.TokenSource() != nil {
		t.Errorf("a nil interface: ResolveOptions = %v, source %v; want no error and no source", err, st.TokenSource())
	}
}
