package cacheref

import "testing"

// TestLearnedThinking_0062 (0062-MADR D5): New makes a memo of refused
// thinking settings of its own, so two Values share none; a nil *Value has
// none, which the memo reads as the process's; a Value's is its own.
func TestLearnedThinking_0062(t *testing.T) {
	a, b := New(nil), New(nil)
	if a.Thinking == nil || a.Thinking == b.Thinking {
		t.Fatalf("New's memos %p and %p; want one each", a.Thinking, b.Thinking)
	}
	var none *Value
	if got := none.LearnedThinking(); got != nil {
		t.Errorf("nil Value's LearnedThinking = %p; want nil", got)
	}
	if got := a.LearnedThinking(); got != a.Thinking {
		t.Errorf("LearnedThinking = %p; want the Value's %p", got, a.Thinking)
	}
}
