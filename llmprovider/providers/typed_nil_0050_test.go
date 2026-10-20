package providers_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
)

// tokenFunc0050 is a caller's own source, of a func type.
type tokenFunc0050 func(context.Context) (llmprovider.Token, error)

func (f tokenFunc0050) Token(ctx context.Context) (llmprovider.Token, error) { return f(ctx) }

// TestNew_TypedNilSources_0050 (0050-MADR D3): every built-in provider's New
// refuses each typed-nil source with ErrInvalidRequest, and none panics; the
// MADR's probe saw a panic in New for OpenAI, Grok and Kilo, and in the first
// Generate of every provider for a *CommandToken.
func TestNew_TypedNilSources_0050(t *testing.T) {
	sources := []llmprovider.TokenSource{
		(*llmprovider.StaticToken)(nil),
		(*llmprovider.CommandToken)(nil),
		(*auth.OAuthSession)(nil),
		(*auth.VendorCLISession)(nil),
		tokenFunc0050(nil),
	}
	for _, d := range providers.Default().Descriptors() {
		for _, src := range sources {
			name := fmt.Sprintf("%s, a nil %T", d.ID, src)
			var err error
			func() {
				defer func() {
					if v := recover(); v != nil {
						t.Errorf("%s: New panicked: %v; want an error", name, v)
					}
				}()
				_, err = providers.New(d.ID, llmprovider.WithTokenSource(src), llmprovider.WithModel("m"))
			}()
			if !errors.Is(err, llmprovider.ErrInvalidRequest) {
				t.Errorf("%s: New = %v; want ErrInvalidRequest", name, err)
			}
		}
	}
}
