// Package limitmemo is the memo of validated output caps (0089-MADR D5): the
// cap a service stated when it refused an output limit, kept for the rest of
// the process so that a later request is held to it. A catalog.Cache holds
// one, through internal/cacheref; a nil *Memo is the process's. It is never
// written to disk, so a wrong cap lasts at most one process.
//
// It also carries, per model, what the catalog finds and a provider reads
// with no export between them: the id a one-model read resolved the model
// to, for a later refusal's model match (0089-MADR V6). The route's fallback
// is not here: it is a call's, carried in catalog.Limits (0093-MADR D7). And
// it holds Off, the value of llmprovider.WithoutLimitLearning (DL73).
package limitmemo

import (
	"strings"
	"sync"
	"time"
)

// Bound is the most keys a memo holds: 0062-MADR's memory bound (0089-MADR
// D5, its numbers table). The oldest key is dropped first. The notes are
// bounded apart, by the same number.
const Bound = 256

// Key is what a cap is kept under (0089-MADR D5, DL38): the provider id; its
// base URL as Settings.BaseURL gives it, empty for the provider's default;
// the Kilo organization or Claude workspace, empty for none; the model as
// the request named it; and whether the request streamed, since Gemini
// states different caps for a streamed and a plain call (0088-MADR ST95).
type Key struct {
	Provider string
	BaseURL  string
	Scope    string
	Model    string
	Stream   bool
}

// norm is k as the memo compares it: the provider and model in lower case,
// each field trimmed, the base URL without trailing slashes.
func (k Key) norm() Key {
	return Key{
		Provider: strings.ToLower(strings.TrimSpace(k.Provider)),
		BaseURL:  strings.TrimRight(strings.TrimSpace(k.BaseURL), "/"),
		Scope:    strings.TrimSpace(k.Scope),
		Model:    strings.ToLower(strings.TrimSpace(k.Model)),
		Stream:   k.Stream,
	}
}

// entry is one kept cap and when it was kept.
type entry struct {
	tokens int
	at     time.Time
}

// note is what the catalog carries to a provider for one model: the id it
// resolved to.
type note struct {
	resolved string
}

// Memo holds a cap per Key, oldest key first, at most Bound keys, and the
// notes. It is safe for concurrent use: every method takes its lock, as a
// test under -race measures.
type Memo struct {
	mu    sync.Mutex
	order []Key
	caps  map[Key]entry
	// notes are each model's, under its key with Stream false, oldest first
	// in notesOrder.
	notes      map[Key]note
	notesOrder []Key
}

// New returns an empty memo.
func New() *Memo { return &Memo{caps: map[Key]entry{}, notes: map[Key]note{}} }

// process is the process's memo: every provider's without catalog.WithCache.
var process = New()

// now is the clock a kept cap's time is read from; a test may replace it.
var now = time.Now

// or is m, or the process's memo when m is nil.
func (m *Memo) or() *Memo {
	if m == nil {
		return process
	}
	return m
}

// Keep keeps tokens as k's cap, dropping the oldest key past Bound, and
// reports whether that changed the memo: k had no cap, or another. A cap
// below 1 is not kept.
func (m *Memo) Keep(k Key, tokens int) bool {
	if tokens < 1 {
		return false
	}
	m = m.or()
	k = k.norm()
	m.mu.Lock()
	defer m.mu.Unlock()
	old, ok := m.caps[k]
	if ok {
		m.caps[k] = entry{tokens, now()}
		return old.tokens != tokens
	}
	if len(m.order) >= Bound {
		delete(m.caps, m.order[0])
		m.order = m.order[1:]
	}
	m.caps[k] = entry{tokens, now()}
	m.order = append(m.order, k)
	return true
}

// Cap is k's kept cap, and whether there is one.
func (m *Memo) Cap(k Key) (int, bool) {
	tokens, _, ok := m.Entry(k)
	return tokens, ok
}

// Entry is k's kept cap, when it was kept, and whether there is one.
func (m *Memo) Entry(k Key) (int, time.Time, bool) {
	m = m.or()
	k = k.norm()
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.caps[k]
	return e.tokens, e.at, ok
}

// Observe drops k's cap when a response for k reported more output than it,
// which shows the cap wrong (0089-MADR D5, DL39), and reports whether it
// did.
func (m *Memo) Observe(k Key, output int) bool {
	m = m.or()
	k = k.norm()
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.caps[k]
	if !ok || output <= e.tokens {
		return false
	}
	delete(m.caps, k)
	m.order = deleteKey(m.order, k)
	return true
}

// KeepResolved keeps id as the id a one-model read resolved k's model to
// (0089-MADR V6); k's Stream is ignored. An empty id is not kept.
func (m *Memo) KeepResolved(k Key, id string) {
	id = strings.TrimSpace(id)
	if id == "" {
		return
	}
	m.or().setNote(k, func(n *note) { n.resolved = id })
}

// Resolved is the id k's model was resolved to, or empty; k's Stream is
// ignored.
func (m *Memo) Resolved(k Key) string { return m.or().getNote(k).resolved }

// setNote changes k's note by set, dropping the oldest note past Bound.
func (m *Memo) setNote(k Key, set func(*note)) {
	k = k.norm()
	k.Stream = false
	m.mu.Lock()
	defer m.mu.Unlock()
	n, ok := m.notes[k]
	if !ok {
		if len(m.notesOrder) >= Bound {
			delete(m.notes, m.notesOrder[0])
			m.notesOrder = m.notesOrder[1:]
		}
		m.notesOrder = append(m.notesOrder, k)
	}
	set(&n)
	m.notes[k] = n
}

// getNote is k's note.
func (m *Memo) getNote(k Key) note {
	k = k.norm()
	k.Stream = false
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.notes[k]
}

// deleteKey is order without k.
func deleteKey(order []Key, k Key) []Key {
	for i, o := range order {
		if o == k {
			return append(order[:i], order[i+1:]...)
		}
	}
	return order
}

// Off is llmprovider.WithoutLimitLearning's shared value: a provider given it
// keeps no cap and reports none, and still resends at a validated cap
// (0089-MADR DL49, DL73).
type Off struct{}

// LearningOff reports whether values, a provider's option values, hold Off.
func LearningOff(values []any) bool {
	for _, v := range values {
		if _, ok := v.(Off); ok {
			return true
		}
	}
	return false
}
