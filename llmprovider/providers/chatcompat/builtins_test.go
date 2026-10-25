package chatcompat_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/chatcompat"
)

// TestNew_BuiltinIDs_0061 (0061-MADR D4): New refuses every id
// providers.Default() holds, in upper case too, so that a built-in added
// there is refused here; and builds one that is not.
func TestNew_BuiltinIDs_0061(t *testing.T) {
	base := llmprovider.WithBaseURL("http://127.0.0.1:1")
	for _, d := range providers.Default().Descriptors() {
		for _, id := range []llmprovider.ProviderID{d.ID, llmprovider.ProviderID(strings.ToUpper(string(d.ID)))} {
			if _, err := chatcompat.New(chatcompat.WithID(id), base); !errors.Is(err, llmprovider.ErrInvalidProvider) {
				t.Errorf("WithID(%q): %v; want ErrInvalidProvider", id, err)
			}
		}
	}
	if _, err := chatcompat.New(chatcompat.WithID("lmstudio"), base); err != nil {
		t.Errorf("WithID(lmstudio): %v", err)
	}
}

// TestRegistry_0061 (0061-MADR D4; R38): a caller registers a chatcompat
// provider by its Descriptor, with a factory that adds WithID, and builds it
// by its id.
func TestRegistry_0061(t *testing.T) {
	reg := llmprovider.NewRegistry()
	if err := reg.Register(chatcompat.Descriptor("lmstudio", "LM Studio"),
		func(opts ...llmprovider.Option) (llmprovider.Provider, error) {
			return chatcompat.New(append(opts, chatcompat.WithID("lmstudio"))...)
		}); err != nil {
		t.Fatal(err)
	}
	p, err := reg.New("lmstudio", llmprovider.For("lmstudio", llmprovider.WithBaseURL("http://127.0.0.1:1234/v1")))
	if err != nil || p.ID() != "lmstudio" {
		t.Errorf("Registry.New: %v, %v; want the lmstudio provider", p, err)
	}
}
