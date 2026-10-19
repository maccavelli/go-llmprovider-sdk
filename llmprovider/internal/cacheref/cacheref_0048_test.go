package cacheref

import (
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// TestFrom_LastValueWins_0048 (0048-MADR D1): From finds the last *Value
// among other option values, so a later WithCache wins, and a nil last one is
// the process's.
func TestFrom_LastValueWins_0048(t *testing.T) {
	a, b := &Value{Probes: transport.NewProbeStore()}, &Value{Probes: transport.NewProbeStore()}
	for i, tc := range []struct {
		values []any
		want   *Value
	}{
		{nil, nil},
		{[]any{1, "x"}, nil},
		{[]any{a}, a},
		{[]any{a, 1, b, "x"}, b},
		{[]any{b, a}, a},
		{[]any{a, (*Value)(nil)}, nil},
	} {
		if got := From(tc.values); got != tc.want {
			t.Errorf("case %d: From = %p; want %p", i, got, tc.want)
		}
	}
}

// TestProbeStore_NilValue_0048: a nil *Value has no store, which the probe
// reads as the process's; a Value's is its own.
func TestProbeStore_NilValue_0048(t *testing.T) {
	var none *Value
	if got := none.ProbeStore(); got != nil {
		t.Errorf("nil Value's ProbeStore = %p; want nil", got)
	}
	v := &Value{Probes: transport.NewProbeStore()}
	if got := v.ProbeStore(); got != v.Probes {
		t.Errorf("ProbeStore = %p; want the Value's %p", got, v.Probes)
	}
}
