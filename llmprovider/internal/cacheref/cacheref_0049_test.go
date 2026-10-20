package cacheref

import "testing"

// TestNew_AWholeValue_0049 (0049-MADR D3): New keeps the catalog's state and
// makes a probe store and a memo of its own, so two Values share neither.
func TestNew_AWholeValue_0049(t *testing.T) {
	state := new(int)
	a, b := New(state), New(nil)
	if a.Catalog != state || a.Probes == nil || a.Refusals == nil {
		t.Fatalf("New = %+v; want the catalog's state, a probe store and a memo", a)
	}
	if a.Probes == b.Probes || a.Refusals == b.Refusals {
		t.Error("two Values share a probe store or a memo")
	}
}

// TestLearnedRefusals_NilValue_0049: a nil *Value has no memo, which the memo
// reads as the process's; a Value's is its own.
func TestLearnedRefusals_NilValue_0049(t *testing.T) {
	var none *Value
	if got := none.LearnedRefusals(); got != nil {
		t.Errorf("nil Value's LearnedRefusals = %p; want nil", got)
	}
	v := New(nil)
	if got := v.LearnedRefusals(); got != v.Refusals {
		t.Errorf("LearnedRefusals = %p; want the Value's %p", got, v.Refusals)
	}
}
