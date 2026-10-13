package providers

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// 0038-MADR D2, A3: catalog.WithLimit is scoped to every built-in provider,
// and a provider's ListModels cuts its own static fallback to it.

// failingSource is a credential that cannot be read, so catalog.List fails
// and ListModels falls back to the static catalog.
type failingSource struct{}

func (failingSource) Token(context.Context) (llmprovider.Token, error) {
	return llmprovider.Token{}, errors.New("no credential")
}

// TestWithLimit_Scope: every built-in New takes catalog.WithLimit, and a
// provider outside its scope refuses it.
func TestWithLimit_Scope(t *testing.T) {
	for _, d := range Default().Descriptors() {
		if _, err := New(d.ID, llmprovider.WithAPIKey("k"), llmprovider.WithModel(firstModel(d)), catalog.WithLimit(2)); err != nil {
			t.Errorf("%s: New refused catalog.WithLimit: %v", d.ID, err)
		}
	}
	if _, err := llmprovider.ResolveOptions("third-party", []llmprovider.Option{catalog.WithLimit(2)}); !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Errorf("a third party resolved catalog.WithLimit with %v; want ErrInvalidRequest", err)
	}
}

// TestListModels_WithLimit: a provider whose listing fails returns its static
// catalog, cut to the limit, and whole without one.
func TestListModels_WithLimit(t *testing.T) {
	for _, d := range Default().Descriptors() {
		if len(d.StaticModels) < 3 {
			continue // Ollama has no static catalog; a short one cannot show a cut.
		}
		t.Run(string(d.ID), func(t *testing.T) {
			for _, tc := range []struct {
				opts []llmprovider.Option
				want []string
			}{
				{[]llmprovider.Option{catalog.WithLimit(2)}, d.StaticModels[:2]},
				{nil, d.StaticModels},
				{[]llmprovider.Option{catalog.WithLimit(0)}, d.StaticModels},
			} {
				p, err := New(d.ID, append([]llmprovider.Option{llmprovider.WithTokenSource(failingSource{}),
					llmprovider.WithModel(firstModel(d)), llmprovider.WithModelProbes(false), llmprovider.WithoutModelMetadata()},
					tc.opts...)...)
				if err != nil {
					t.Fatal(err)
				}
				lister, ok := p.(llmprovider.ModelLister)
				if !ok {
					t.Fatal("not a ModelLister")
				}
				if got, _ := lister.ListModels(context.Background()); !slices.Equal(got, tc.want) {
					t.Errorf("%d options: ListModels = %v; want %v", len(tc.opts), got, tc.want)
				}
			}
		})
	}
}
