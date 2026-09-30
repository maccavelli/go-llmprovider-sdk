// Package providers builds the built-in providers by id (0015-MADR D10).
//
// Default returns a new Registry holding every built-in provider; there is no
// global registry and no init-time registration. During 0015-PLAN S7 the
// built-in providers move here one by one, and Default gains each as it
// arrives.
package providers

import (
	"fmt"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers/openai"
)

// builtins are the providers that have moved to their own packages, in menu
// order.
var builtins = []struct {
	id  string
	new llmprovider.Factory
}{
	{llmprovider.ProviderOpenAI, openai.New},
}

// Default returns a new Registry holding the built-in providers.
func Default() *llmprovider.Registry {
	r := llmprovider.NewRegistry()
	for _, b := range builtins {
		if err := r.Register(descriptor(b.id), b.new); err != nil {
			panic(fmt.Sprintf("providers: built-in %q: %v", b.id, err))
		}
	}
	return r
}

// New builds the built-in provider id with opts. An unknown id gives an error
// matching llmprovider.ErrInvalidProvider.
func New(id llmprovider.ProviderID, opts ...llmprovider.Option) (llmprovider.Provider, error) {
	return Default().New(id, opts...)
}

// descriptor is the built-in id's descriptor, from llmprovider's until
// 0015-PLAN S8 moves each into its provider's package.
func descriptor(id string) llmprovider.Descriptor {
	d, ok := llmprovider.DescriptorFor(id)
	if !ok {
		return llmprovider.Descriptor{ID: llmprovider.ProviderID(id)}
	}
	return llmprovider.Descriptor{
		ID:              llmprovider.ProviderID(d.ID),
		Label:           d.Label,
		EnvVar:          d.EnvVar,
		DefaultBaseURL:  d.DefaultBaseURL,
		SupportsBaseURL: d.SupportsBaseURL,
		IsLocal:         d.IsLocal,
		RequiresAPIKey:  d.RequiresAPIKey,
		AuthMethods:     d.AuthMethods,
		StaticModels:    d.StaticModels,
		Notes:           d.Notes,
	}
}
