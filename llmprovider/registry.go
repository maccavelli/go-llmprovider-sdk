package llmprovider

import (
	"fmt"
	"sync"
)

// Factory builds a provider from options (0015-MADR D5, D10).
type Factory func(opts ...Option) (Provider, error)

// Descriptor is what a configuration UI needs to know about a provider
// (0015-MADR D10).
type Descriptor struct {
	// ID is the provider's canonical identifier.
	ID ProviderID
	// Label is the human-readable name for a menu.
	Label string
	// EnvVar is the conventional environment variable for the credential.
	// Empty when RequiresAPIKey is false.
	EnvVar string
	// DefaultBaseURL is the endpoint used when none is configured, or empty.
	DefaultBaseURL string
	// SupportsBaseURL reports whether a caller may override the endpoint.
	SupportsBaseURL bool
	// IsLocal reports whether the provider runs on the user's machine.
	IsLocal bool
	// RequiresAPIKey reports whether a credential must be collected.
	RequiresAPIKey bool
	// AuthMethods lists the authentication paths in menu order.
	AuthMethods []AuthMethod
	// StaticModels is the curated fallback catalog, or empty.
	StaticModels []string
	// Notes is a short qualifier for a menu, or empty.
	Notes string
}

// clone copies d's slices.
func (d Descriptor) clone() Descriptor {
	d.AuthMethods = append([]AuthMethod(nil), d.AuthMethods...)
	d.StaticModels = append([]string(nil), d.StaticModels...)
	return d
}

// Registry holds the providers a caller can build by id (0015-MADR D10).
// There is no global registry. The zero value is empty and ready to use, and
// a Registry is safe for concurrent use.
type Registry struct {
	mu      sync.RWMutex
	order   []ProviderID
	entries map[ProviderID]registryEntry
}

type registryEntry struct {
	desc    Descriptor
	factory Factory
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry { return &Registry{} }

// Register adds a provider. It refuses an empty id, a nil factory, and an id
// already registered, with an error matching ErrInvalidProvider.
func (r *Registry) Register(d Descriptor, f Factory) error {
	switch {
	case d.ID == "":
		return fmt.Errorf("llmprovider: a descriptor needs an id: %w", ErrInvalidProvider)
	case f == nil:
		return fmt.Errorf("llmprovider: provider %q has no factory: %w", d.ID, ErrInvalidProvider)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, dup := r.entries[d.ID]; dup {
		return fmt.Errorf("llmprovider: provider %q is already registered: %w", d.ID, ErrInvalidProvider)
	}
	if r.entries == nil {
		r.entries = make(map[ProviderID]registryEntry)
	}
	r.entries[d.ID] = registryEntry{desc: d.clone(), factory: f}
	r.order = append(r.order, d.ID)
	return nil
}

// Descriptor returns a copy of the descriptor registered for id.
func (r *Registry) Descriptor(id ProviderID) (Descriptor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	entry, ok := r.entries[id]
	if !ok {
		return Descriptor{}, false
	}
	return entry.desc.clone(), true
}

// Descriptors returns copies of every descriptor, in registration order.
func (r *Registry) Descriptors() []Descriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Descriptor, 0, len(r.order))
	for _, id := range r.order {
		out = append(out, r.entries[id].desc.clone())
	}
	return out
}

// New builds the provider registered for id. An unknown id, or a factory
// that builds a provider with another id, gives an error matching
// ErrInvalidProvider.
func (r *Registry) New(id ProviderID, opts ...Option) (Provider, error) {
	r.mu.RLock()
	entry, ok := r.entries[id]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("llmprovider: no provider %q in the registry: %w", id, ErrInvalidProvider)
	}
	p, err := entry.factory(opts...)
	if err != nil {
		return nil, err
	}
	if p == nil || p.ID() != id {
		return nil, fmt.Errorf("llmprovider: the factory for %q built another provider: %w", id, ErrInvalidProvider)
	}
	return p, nil
}
