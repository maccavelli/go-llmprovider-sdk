// Package cacheref carries a catalog.Cache from a provider's options to the
// code that reads it: the catalog's listings and metadata, the wire's probes
// (0048-MADR D1), the Messages wire's learned refusals (0049-MADR D3), and
// the thinking settings models refused (0062-MADR D5). catalog.WithCache puts
// a *Value among the options; a provider and the catalog find it with From.
package cacheref

import (
	"slices"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wire/messages"
)

// Value is catalog.WithCache's option value. Catalog is the catalog's state,
// which only the catalog reads; Probes is the probe store; Refusals is the
// memo of learned forced-choice refusals; Thinking is the memo of refused
// thinking settings. A nil *Value means the process's caches. A Value is not
// changed after it is made, so it is safe for concurrent use.
type Value struct {
	Catalog  any
	Probes   *transport.ProbeStore
	Refusals *messages.Refusals
	Thinking *wire.Learned
}

// New is a Value of catalog's state, with a probe store and memos of learned
// refusals of its own (0049-MADR D3; 0062-MADR D5).
func New(catalog any) *Value {
	return &Value{Catalog: catalog, Probes: transport.NewProbeStore(), Refusals: messages.NewRefusals(),
		Thinking: wire.NewLearned()}
}

// From returns the last *Value among a provider's option values, so that a
// later WithCache wins; nil when there is none, or the last is nil.
func From(values []any) *Value {
	for _, value := range slices.Backward(values) {
		if v, ok := value.(*Value); ok {
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

// LearnedThinking is v's memo of refused thinking settings: nil for a nil v,
// which the memo reads as the process's.
func (v *Value) LearnedThinking() *wire.Learned {
	if v == nil {
		return nil
	}
	return v.Thinking
}

// ProbeStore is v's probe store: nil for a nil v, which the store reads as
// the process's.
func (v *Value) ProbeStore() *transport.ProbeStore {
	if v == nil {
		return nil
	}
	return v.Probes
}
