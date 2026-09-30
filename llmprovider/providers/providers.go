// Package providers builds the built-in providers by id (0015-MADR D10).
//
// Default returns a new Registry holding every built-in provider; there is no
// global registry and no init-time registration. During 0015-PLAN S7 the
// built-in providers move here one by one, and Default gains each as it
// arrives.
package providers

import "github.com/maccavelli/go-llmprovider-sdk/llmprovider"

// Default returns a new Registry holding the built-in providers.
func Default() *llmprovider.Registry {
	return llmprovider.NewRegistry()
}

// New builds the built-in provider id with opts. An unknown id gives an error
// matching llmprovider.ErrInvalidProvider.
func New(id llmprovider.ProviderID, opts ...llmprovider.Option) (llmprovider.Provider, error) {
	return Default().New(id, opts...)
}
