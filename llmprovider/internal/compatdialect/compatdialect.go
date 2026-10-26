// Package compatdialect tells which server a Chat Completions server's GET
// {base}/models reply is from, by the marks each entry carries (0061-MADR
// E2; its amendment "E2 and E4's menu, made exact before Phase 4c",
// decision 3). catalog reads a dialect's details; chatcompat labels a
// discovered server by it (the amendment "E5's ports and a keyed server,
// confirmed before Phase 4e"). A dialect is in Table only with a recorded
// reply of its server; a listing that scores none is a plain listing.
package compatdialect

import (
	"encoding/json"
	"strings"
	"time"
)

// Threshold is the share of a dialect's marks a listing must meet: every
// mark, on every entry.
const Threshold = 1.0

// Entry is one entry of a GET {base}/models reply: what a listing reads, and
// what the marks and a default rank read.
type Entry struct {
	ID      string          `json:"id"`
	Created json.RawMessage `json:"created"`
	OwnedBy *string         `json:"owned_by"`
}

// CreatedAt is the entry's created time, and whether it is a positive
// integer of Unix seconds.
func (e Entry) CreatedAt() (time.Time, bool) {
	var unix int64
	if len(e.Created) == 0 || json.Unmarshal(e.Created, &unix) != nil || unix <= 0 {
		return time.Time{}, false
	}
	return time.Unix(unix, 0).UTC(), true
}

// Dialect is one server's name, its label for a person, and its marks.
type Dialect struct {
	Name, Label string
	Marks       []func(Entry) bool
}

// Table is the recorded dialects, in the order Of tries them.
var Table = []Dialect{
	{
		// Ollama's /v1/models: every id tagged (llama3.2:latest), an integer
		// created, an owned_by (testdata/compat/ollama/models.json).
		Name: "ollama", Label: "Ollama",
		Marks: []func(Entry) bool{
			func(e Entry) bool { return strings.Contains(e.ID, ":") },
			func(e Entry) bool { _, ok := e.CreatedAt(); return ok },
			func(e Entry) bool { return e.OwnedBy != nil },
		},
	},
}

// Score is the share of d's marks entries meet, 0 for none.
func Score(d Dialect, entries []Entry) float64 {
	if len(entries) == 0 || len(d.Marks) == 0 {
		return 0
	}
	met := 0
	for _, e := range entries {
		for _, mark := range d.Marks {
			if mark(e) {
				met++
			}
		}
	}
	return float64(met) / float64(len(entries)*len(d.Marks))
}

// Of is the first dialect of Table entries score at Threshold, and whether
// one does.
func Of(entries []Entry) (Dialect, bool) {
	for _, d := range Table {
		if Score(d, entries) >= Threshold {
			return d, true
		}
	}
	return Dialect{}, false
}
