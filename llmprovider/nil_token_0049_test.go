package llmprovider_test

import (
	"errors"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
)

// noPanic0049 runs f, failing t with what a panic in it was, so that a
// regression is a failure, not a crash of the test binary.
func noPanic0049(t *testing.T, what string, f func() error) error {
	t.Helper()
	var err error
	func() {
		defer func() {
			if v := recover(); v != nil {
				t.Fatalf("%s panicked: %v; want an error", what, v)
			}
		}()
		err = f()
	}()
	return err
}

// TestResolveOptions_NilStaticToken_0049 (0049-MADR D1): a typed-nil
// *StaticToken is refused with ErrInvalidRequest, by ResolveOptions and so by
// every New, never a panic; a nil TokenSource is still no source.
func TestResolveOptions_NilStaticToken_0049(t *testing.T) {
	var nilToken *llmprovider.StaticToken
	err := noPanic0049(t, "ResolveOptions", func() error {
		_, err := llmprovider.ResolveOptions(llmprovider.ProviderClaude, []llmprovider.Option{llmprovider.WithTokenSource(nilToken)})
		return err
	})
	if !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Errorf("ResolveOptions(a nil *StaticToken) = %v; want ErrInvalidRequest", err)
	}
	err = noPanic0049(t, "providers.New", func() error {
		_, err := providers.New(llmprovider.ProviderClaude, llmprovider.WithTokenSource(nilToken))
		return err
	})
	if !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Errorf("providers.New(claude, a nil *StaticToken) = %v; want ErrInvalidRequest", err)
	}
	st, err := llmprovider.ResolveOptions(llmprovider.ProviderClaude, []llmprovider.Option{llmprovider.WithTokenSource(nil)})
	if err != nil || st.TokenSource() != nil {
		t.Errorf("ResolveOptions(a nil TokenSource) = %v, source %v; want no error and no source", err, st.TokenSource())
	}
}
