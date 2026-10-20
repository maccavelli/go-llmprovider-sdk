// Package cacheref carries a catalog.Cache from a provider's options to the
// code that reads it: the catalog's listings and metadata, the wire's probes
// (0048-MADR D1), and the Messages wire's learned refusals (0049-MADR D3). catalog.WithCache puts a *Value among the options; a
// provider and the catalog find it with From.
package cacheref

import (
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/messages"
)

// Value is catalog.WithCache's option value. Catalog is the catalog's state,
// which only the catalog reads; Probes is the probe store; Refusals is the
// memo of learned forced-choice refusals. A nil *Value means
// the process's caches. A Value is not changed after it is made, so it is safe
// for concurrent use.
type Value struct {
	Catalog  any
	Probes   *transport.ProbeStore
	Refusals *messages.Refusals
}

// New is a Value of catalog's state, with a probe store and a memo of learned
// refusals of its own (0049-MADR D3).
func New(catalog any) *Value {
	return &Value{Catalog: catalog, Probes: transport.NewProbeStore(), Refusals: messages.NewRefusals()}
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

// LearnedRefusals is v's memo of learned refusals: nil for a nil v, which the
// memo reads as the process's.
func (v *Value) LearnedRefusals() *messages.Refusals {
	if v == nil {
		return nil
	}
	return v.Refusals
}

// ProbeStore is v's probe store: nil for a nil v, which the store reads as
// the process's.
func (v *Value) ProbeStore() *transport.ProbeStore {
	if v == nil {
		return nil
	}
	return v.Probes
}
