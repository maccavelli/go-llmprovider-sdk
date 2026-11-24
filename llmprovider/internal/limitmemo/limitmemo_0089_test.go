package limitmemo

import (
	"strconv"
	"sync"
	"testing"
)

// key is a Claude key for model.
func key(model string) Key { return Key{Provider: "claude", Model: model} }

// TestLimitMemo_Bound_0089 (DL38): the memo holds at most Bound keys; the
// 257th drops the first.
func TestLimitMemo_Bound_0089(t *testing.T) {
	m := New()
	for i := range Bound + 1 {
		if !m.Keep(key("model-"+strconv.Itoa(i)), 4096) {
			t.Fatalf("Keep of model-%d reported no change", i)
		}
	}
	if _, ok := m.Cap(key("model-0")); ok {
		t.Errorf("after %d keys, the first is still kept; want it dropped", Bound+1)
	}
	for _, i := range []int{1, Bound} {
		if c, ok := m.Cap(key("model-" + strconv.Itoa(i))); !ok || c != 4096 {
			t.Errorf("model-%d: %d, %v; want 4096 kept", i, c, ok)
		}
	}
	if n := len(m.caps); n != Bound {
		t.Errorf("%d keys held; want %d", n, Bound)
	}
}

// TestLimitMemo_SelfHeal_0089 (DL39): a response for a kept cap's key that
// reports more output than the cap drops it; one at or below it does not.
func TestLimitMemo_SelfHeal_0089(t *testing.T) {
	m := New()
	k := key("claude-opus-4-5")
	m.Keep(k, 4096)
	if m.Observe(k, 4096) {
		t.Errorf("an output at the cap dropped it")
	}
	if m.Observe(Key{Provider: "claude", Model: "other"}, 10000) {
		t.Errorf("another key's output dropped a cap")
	}
	if !m.Observe(k, 10000) {
		t.Errorf("Observe(10000) over a cap of 4096 reported no drop")
	}
	if c, ok := m.Cap(k); ok {
		t.Errorf("after 10,000 output tokens the cap is %d; want it dropped", c)
	}
	if m.Keep(k, 4096); len(m.order) != 1 {
		t.Errorf("order %v after a drop and a keep; want one key", m.order)
	}
}

// TestMemo_KeyAndValue: a key is matched in lower case and trimmed, the base
// URL without its trailing slashes; a second Keep of the same cap is no
// change, of another cap a change; a cap below 1 is not kept; a nil memo is
// the process's.
func TestMemo_KeyAndValue(t *testing.T) {
	m := New()
	k := Key{Provider: "Kilo", BaseURL: "https://example.test/", Scope: " org ", Model: " Anthropic/Claude-Sonnet-4.5 "}
	if !m.Keep(k, 64000) || m.Keep(k, 64000) {
		t.Errorf("Keep twice: want a change, then none")
	}
	if c, ok := m.Cap(Key{Provider: "kilo", BaseURL: "https://example.test", Scope: "org", Model: "anthropic/claude-sonnet-4.5"}); !ok || c != 64000 {
		t.Errorf("the normalised key: %d, %v; want 64000", c, ok)
	}
	if !m.Keep(k, 32000) {
		t.Errorf("Keep of another cap reported no change")
	}
	if m.Keep(key("zero"), 0) {
		t.Errorf("a cap of 0 kept")
	}
	var none *Memo
	pk := key("process-memo-test")
	none.Keep(pk, 1000)
	if c, ok := process.Cap(pk); !ok || c != 1000 {
		t.Errorf("a nil memo's Keep: the process's memo holds %d, %v; want 1000", c, ok)
	}
	if !none.Observe(pk, 2000) {
		t.Errorf("a nil memo's Observe did not drop the process's cap")
	}
}

// TestMemo_Notes: a model's notes, the id a one-model read resolved it to
// (0089-MADR V6) and its route's fallback (DL51), are kept under the model's
// key whatever its Stream, normalised as a cap's key is; an empty id is not
// kept; a fallback below 1 clears it; the notes hold at most Bound models,
// the oldest dropped; a nil memo's are the process's.
func TestMemo_Notes(t *testing.T) {
	m := New()
	k := Key{Provider: "Claude", Model: " Claude-Sonnet-4-5 ", Stream: true}
	m.KeepResolved(k, " claude-sonnet-4-5-20250929 ")
	m.KeepResolved(k, "  ")
	plain := Key{Provider: "claude", Model: "claude-sonnet-4-5"}
	if got := m.Resolved(plain); got != "claude-sonnet-4-5-20250929" {
		t.Errorf("Resolved = %q; want the id kept, the empty one ignored", got)
	}
	m.SetFallback(plain, 64000, "models.dev (bundled 2026-10-12)")
	if n, src := m.Fallback(k); n != 64000 || src != "models.dev (bundled 2026-10-12)" {
		t.Errorf("Fallback = %d, %q; want 64000 from the extract", n, src)
	}
	if got := m.Resolved(k); got != "claude-sonnet-4-5-20250929" {
		t.Errorf("after SetFallback, Resolved = %q; want it kept", got)
	}
	m.SetFallback(plain, 0, "ignored")
	if n, src := m.Fallback(plain); n != 0 || src != "" {
		t.Errorf("a fallback of 0: %d, %q; want it cleared", n, src)
	}
	for i := range Bound + 1 {
		m.SetFallback(key("note-"+strconv.Itoa(i)), 1000, "s")
	}
	if n, _ := m.Fallback(key("note-0")); n != 0 || len(m.notes) != Bound {
		t.Errorf("after %d more models: note-0 %d, %d notes; want the oldest dropped, %d held", Bound+1, n,
			len(m.notes), Bound)
	}
	var none *Memo
	pk := key("process-note-test")
	none.KeepResolved(pk, "resolved-id")
	if got := process.Resolved(pk); got != "resolved-id" {
		t.Errorf("a nil memo's note: the process's memo holds %q", got)
	}
}

// TestLearningOff: Off among a provider's option values turns learning off;
// without it, it stays on.
func TestLearningOff(t *testing.T) {
	if !LearningOff([]any{"x", Off{}}) {
		t.Errorf("Off among the values: want learning off")
	}
	if LearningOff([]any{"x", 1}) || LearningOff(nil) {
		t.Errorf("no Off: want learning on")
	}
}

// TestMemo_ConcurrentUse: under -race, goroutines keeping, reading and
// dropping caps on one memo, the race detector reports nothing, and the
// bound holds.
func TestMemo_ConcurrentUse(t *testing.T) {
	m := New()
	var wg sync.WaitGroup
	for g := range 16 {
		wg.Go(func() {
			for i := range 64 {
				k := key("model-" + strconv.Itoa((g*64+i)%300))
				m.Keep(k, 4096+i)
				m.Cap(k)
				m.Observe(k, 4096+2*i)
			}
		})
	}
	wg.Wait()
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.caps) > Bound || len(m.order) != len(m.caps) {
		t.Errorf("%d caps, %d in order; want at most %d, the same", len(m.caps), len(m.order), Bound)
	}
}
