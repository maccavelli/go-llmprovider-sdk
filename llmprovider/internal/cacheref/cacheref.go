// Package cacheref carries a catalog.Cache from a provider's options to the
// code that reads it: the catalog's listings and metadata, and the wire's
// probes (0048-MADR D1). catalog.WithCache puts a *Value among the options; a
// provider and the catalog find it with From.
package cacheref

import "github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"

// Value is catalog.WithCache's option value. Catalog is the catalog's state,
// which only the catalog reads; Probes is the probe store. A nil *Value means
// the process's caches. A Value is not changed after it is made, so it is safe
// for concurrent use.
type Value struct {
	Catalog any
	Probes  *transport.ProbeStore
}

// From returns the last *Value among a provider's option values, so that a
// later WithCache wins; nil when there is none, or the last is nil.
func From(values []any) *Value {
	for i := len(values) - 1; i >= 0; i-- {
		if v, ok := values[i].(*Value); ok {
			return v
		}
	}
	return nil
}

// ProbeStore is v's probe store: nil for a nil v, which the store reads as
// the process's.
func (v *Value) ProbeStore() *transport.ProbeStore {
	if v == nil {
		return nil
	}
	return v.Probes
}
