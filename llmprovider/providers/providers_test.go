package providers

import (
	"errors"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestDescriptors_EveryDescriptorIsConstructible: no descriptor exists that a
// wizard can offer but nothing can build. It was the NewProvider test of
// ollama_test.go. While 0015-PLAN S7 ran, a provider not yet moved was built
// through its old constructor; since S7's last move, every descriptor must be
// in Default.
func TestDescriptors_EveryDescriptorIsConstructible(t *testing.T) {
	registry := Default()
	for _, d := range llmprovider.Descriptors() {
		t.Run(d.ID, func(t *testing.T) {
			id := llmprovider.ProviderID(d.ID)
			if _, registered := registry.Descriptor(id); !registered {
				t.Fatalf("descriptor %q is offered to users but Default does not build it", d.ID)
			}
			model := "test-model"
			if len(d.StaticModels) > 0 {
				model = d.StaticModels[0]
			}
			opts := []llmprovider.Option{llmprovider.WithModel(model)}
			if d.RequiresAPIKey {
				opts = append(opts, llmprovider.WithAPIKey("test-key"))
			}
			p, err := New(id, opts...)
			if err != nil || p.ID() != id {
				t.Fatalf("New(%q) = %v, %v", d.ID, p, err)
			}
		})
	}
}

// TestNew_RefusesAnUnknownProvider was the unknown-name half of TestNewProvider.
func TestNew_RefusesAnUnknownProvider(t *testing.T) {
	if _, err := New("unsupported-p"); !errors.Is(err, llmprovider.ErrInvalidProvider) {
		t.Fatalf("New(unsupported-p) err = %v, want ErrInvalidProvider", err)
	}
}

func TestDefault_ReturnsANewRegistryEachCall(t *testing.T) {
	a, b := Default(), Default()
	if a == b {
		t.Fatal("Default returned the same Registry twice; there is no global registry (0015-MADR D10)")
	}
	if err := a.Register(llmprovider.Descriptor{ID: "extra"}, func(...llmprovider.Option) (llmprovider.Provider, error) {
		return nil, errors.New("unused")
	}); err != nil {
		t.Fatal(err)
	}
	if _, ok := b.Descriptor("extra"); ok {
		t.Fatal("a registration reached another Default registry")
	}
}
