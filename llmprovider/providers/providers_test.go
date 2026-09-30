package providers

import (
	"context"
	"errors"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// named is the part of the old API's provider interface the test reads.
type named interface{ Name() string }

// notYetMoved builds, through the old API, each provider that has not moved
// into its own package yet (0015-PLAN S7). A provider leaves this table in the
// commit that registers it in Default; the table is empty when S7 ends.
var notYetMoved = map[string]func(key, model string) (named, error){
	llmprovider.ProviderGemini: func(key, model string) (named, error) {
		return llmprovider.NewGemini(context.Background(), key, model)
	},
	llmprovider.ProviderGrok: func(key, model string) (named, error) {
		return llmprovider.NewGrok(key, model)
	},
	llmprovider.ProviderOpencodeZen: func(key, model string) (named, error) {
		return llmprovider.NewOpencode(llmprovider.ProviderOpencodeZen, key, model)
	},
	llmprovider.ProviderOpencodeGo: func(key, model string) (named, error) {
		return llmprovider.NewOpencode(llmprovider.ProviderOpencodeGo, key, model)
	},
	llmprovider.ProviderHuggingFace: func(key, model string) (named, error) {
		return llmprovider.NewHuggingFace(key, model)
	},
	llmprovider.ProviderKilo: func(key, model string) (named, error) {
		return llmprovider.NewKilo(key, model)
	},
	llmprovider.ProviderTogether: func(key, model string) (named, error) {
		return llmprovider.NewTogether(key, model)
	},
	llmprovider.ProviderOllama: func(key, model string) (named, error) {
		return llmprovider.NewOllama(key, model)
	},
}

// TestDescriptors_EveryDescriptorIsConstructible: no descriptor exists that a
// wizard can offer but nothing can build. It was the NewProvider test of
// ollama_test.go; while S7 runs, a provider is built through Default once it
// has moved, and through its old constructor until then.
func TestDescriptors_EveryDescriptorIsConstructible(t *testing.T) {
	registry := Default()
	for _, d := range llmprovider.Descriptors() {
		t.Run(d.ID, func(t *testing.T) {
			key := "test-key"
			if !d.RequiresAPIKey {
				key = ""
			}
			model := "test-model"
			if len(d.StaticModels) > 0 {
				model = d.StaticModels[0]
			}
			id := llmprovider.ProviderID(d.ID)
			legacy, notMoved := notYetMoved[d.ID]
			if _, registered := registry.Descriptor(id); registered {
				if notMoved {
					t.Fatalf("%q is in Default and still in notYetMoved; remove it from the table", d.ID)
				}
				opts := []llmprovider.Option{llmprovider.WithModel(model)}
				if key != "" {
					opts = append(opts, llmprovider.WithAPIKey(key))
				}
				p, err := New(id, opts...)
				if err != nil || p.ID() != id {
					t.Fatalf("New(%q) = %v, %v", d.ID, p, err)
				}
				return
			}
			if !notMoved {
				t.Fatalf("descriptor %q is offered to users but neither Default nor the old API builds it", d.ID)
			}
			prov, err := legacy(key, model)
			if err != nil {
				t.Fatalf("descriptor %q is offered to users but its constructor fails: %v", d.ID, err)
			}
			if prov.Name() != d.ID {
				t.Errorf("Name() = %q, want %q", prov.Name(), d.ID)
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
