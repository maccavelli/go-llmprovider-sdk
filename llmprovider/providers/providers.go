// Package providers builds the built-in providers by id (0015-MADR D10).
//
// Default returns a new Registry holding every built-in provider; there is no
// global registry and no init-time registration. Each built-in provider is in
// its own package, as 0015-PLAN S7 moved it.
package providers

import (
	"fmt"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/claude"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/gemini"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/grok"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/huggingface"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/kilo"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/ollama"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/openai"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/opencode"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/together"
)

// builtins are the built-in providers, each with its package's descriptor, in
// menu order: remote providers first, local last, as llmprovider's old
// Descriptors() listed them for the wizard (0015-PLAN S8, commit 1).
var builtins = []struct {
	descriptor func() llmprovider.Descriptor
	new        llmprovider.Factory
}{
	{gemini.Descriptor, gemini.New},
	{openai.Descriptor, openai.New},
	{claude.Descriptor, claude.New},
	{grok.Descriptor, grok.New},
	{opencode.DescriptorZen, opencode.NewZen},
	{opencode.DescriptorGo, opencode.NewGo},
	{huggingface.Descriptor, huggingface.New},
	{kilo.Descriptor, kilo.New},
	{together.Descriptor, together.New},
	{ollama.Descriptor, ollama.New},
}

// Default returns a new Registry holding the built-in providers.
func Default() *llmprovider.Registry {
	r := llmprovider.NewRegistry()
	for _, b := range builtins {
		d := b.descriptor()
		if err := r.Register(d, b.new); err != nil {
			panic(fmt.Sprintf("providers: built-in %q: %v", d.ID, err))
		}
	}
	return r
}

// New builds the built-in provider id with opts. An unknown id gives an error
// matching llmprovider.ErrInvalidProvider.
func New(id llmprovider.ProviderID, opts ...llmprovider.Option) (llmprovider.Provider, error) {
	return Default().New(id, opts...)
}
