// Package providers builds the built-in providers by id (0015-MADR D10).
//
// Default returns a new Registry holding every built-in provider; there is no
// global registry and no init-time registration. Each built-in provider is in
// its own package, as 0015-PLAN S7 moved it.
//
// A configuration that names its provider by id builds it with New, and a
// menu lists Default's descriptors:
//
//	p, err := providers.New(llmprovider.ProviderID(cfg.Provider),
//		llmprovider.WithAPIKey(cfg.APIKey), llmprovider.WithModel(cfg.Model))
//
//	for _, d := range providers.Default().Descriptors() {
//		fmt.Println(d.ID, d.Label)
//	}
package providers

import (
	"fmt"
	"slices"

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
	return DefaultWith()
}

// DefaultWith is Default, with opts given to every built-in provider's New
// before the options of the call that builds it, so that a later option wins:
// catalog.WithCache, for example, so that a Registry's providers share one
// cache (0048-MADR D1). A provider the caller registers on the Registry is not
// given opts. An option a built-in refuses fails that provider's New, as it
// would given there, with an error matching llmprovider.ErrInvalidRequest.
func DefaultWith(opts ...llmprovider.Option) *llmprovider.Registry {
	r := llmprovider.NewRegistry()
	for _, b := range builtins {
		d := b.descriptor()
		if err := r.Register(d, withOptions(b.new, opts)); err != nil {
			panic(fmt.Sprintf("providers: built-in %q: %v", d.ID, err))
		}
	}
	return r
}

// withOptions is f with opts before its own; f itself when there are none.
func withOptions(f llmprovider.Factory, opts []llmprovider.Option) llmprovider.Factory {
	if len(opts) == 0 {
		return f
	}
	base := slices.Clone(opts)
	return func(more ...llmprovider.Option) (llmprovider.Provider, error) {
		return f(append(slices.Clone(base), more...)...)
	}
}

// New builds the built-in provider id with opts. An unknown id gives an error
// matching llmprovider.ErrInvalidProvider.
func New(id llmprovider.ProviderID, opts ...llmprovider.Option) (llmprovider.Provider, error) {
	return Default().New(id, opts...)
}
