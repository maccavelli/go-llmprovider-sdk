package compatdialect

import (
	"encoding/json"
	"testing"
	"time"
)

// ollamaEntry is an entry with Ollama's three marks.
func ollamaEntry(id string) Entry {
	owner := "library"
	return Entry{ID: id, Created: json.RawMessage(`1783879777`), OwnedBy: &owner}
}

// TestOf_0061: a listing with every mark is Ollama's; one entry short of a
// mark is none; an empty listing is none; the score counts marks met.
func TestOf_0061(t *testing.T) {
	full := []Entry{ollamaEntry("llama3.2:latest"), ollamaEntry("qwen3:8b")}
	if d, ok := Of(full); !ok || d.Name != "ollama" || d.Label != "Ollama" {
		t.Errorf("Of(Ollama's) = %+v, %v; want ollama", d, ok)
	}
	untagged := []Entry{ollamaEntry("llama3.2:latest"), ollamaEntry("plain")}
	noOwner := []Entry{ollamaEntry("a:1"), {ID: "b:1", Created: json.RawMessage(`5`)}}
	for name, entries := range map[string][]Entry{"untagged": untagged, "no owner": noOwner, "empty": nil} {
		if d, ok := Of(entries); ok {
			t.Errorf("%s: %q, want none", name, d.Name)
		}
	}
	if got := Score(Table[0], untagged); got != 5.0/6 {
		t.Errorf("Score(untagged) = %v, want 5/6", got)
	}
	if got := Score(Dialect{}, full); got != 0 {
		t.Errorf("Score(no marks) = %v, want 0", got)
	}
}

// TestCreatedAt_0061: a positive integer is Unix seconds, in UTC; anything
// else is none.
func TestCreatedAt_0061(t *testing.T) {
	if at, ok := ollamaEntry("a:1").CreatedAt(); !ok || !at.Equal(time.Date(2026, 7, 12, 18, 9, 37, 0, time.UTC)) ||
		at.Location() != time.UTC {
		t.Errorf("CreatedAt = %v, %v; want 2026-07-12T18:09:37Z", at, ok)
	}
	for _, raw := range []string{"", `0`, `-5`, `"1783879777"`, `1.5`, `null`} {
		if at, ok := (Entry{Created: json.RawMessage(raw)}).CreatedAt(); ok {
			t.Errorf("CreatedAt(%s) = %v, want none", raw, at)
		}
	}
}
