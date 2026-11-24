package cacheref

import "testing"

// TestLimitMemo_0089 (0089-MADR D5): New makes a memo of output caps of its
// own, so two Values share none; a nil *Value has none, which the memo
// reads as the process's; a Value's is its own.
func TestLimitMemo_0089(t *testing.T) {
	a, b := New(nil), New(nil)
	if a.Limits == nil || a.Limits == b.Limits {
		t.Fatalf("New's memos %p and %p; want one each", a.Limits, b.Limits)
	}
	var none *Value
	if got := none.LimitMemo(); got != nil {
		t.Errorf("nil Value's LimitMemo = %p; want nil", got)
	}
	if got := a.LimitMemo(); got != a.Limits {
		t.Errorf("LimitMemo = %p; want the Value's %p", got, a.Limits)
	}
}
