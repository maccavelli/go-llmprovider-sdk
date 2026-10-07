package wire

import (
	"regexp"
	"strconv"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// CallIDRule is a wire's rule for a call id: the characters it refuses, each
// sent as "_", and its longest length in bytes.
type CallIDRule struct {
	Refused *regexp.Regexp
	MaxLen  int
}

// AnthropicCallIDs is the Messages wire's rule: ^[a-zA-Z0-9_-]+$, at most 64
// long (0026-MADR F8).
var AnthropicCallIDs = CallIDRule{Refused: regexp.MustCompile(`[^a-zA-Z0-9_-]`), MaxLen: 64}

// RemapCallIDs returns items with each call id that breaks rule, or repeats
// an earlier call's id, replaced by a valid id unique in the request: refused
// characters become "_", the id is cut to MaxLen, and "_2", "_3"… tell
// collisions apart. Each result takes the id of the latest call before it
// that had its original id. items itself is never changed; when no id needs
// replacing it is returned as it is (0026-MADR F8, Q5 A).
//
// A conversation can move between wires, through OpenCode's per-request
// Model or a caller's fallback, so ids another wire made, such as Gemini's
// "name#index" (0021-MADR W10), reach a wire whose rule they break.
func RemapCallIDs(items []llmprovider.Item, rule CallIDRule) []llmprovider.Item {
	used := map[string]bool{}
	latest := map[string]string{} // a call's original id → the id it is sent with
	var out []llmprovider.Item
	set := func(i int, item llmprovider.Item) {
		if out == nil {
			out = make([]llmprovider.Item, len(items))
			copy(out, items)
		}
		out[i] = item
	}
	for i, item := range items {
		switch v := item.(type) {
		case llmprovider.FunctionCallItem:
			id := rule.unique(rule.valid(v.CallID), used)
			used[id], latest[v.CallID] = true, id
			if id != v.CallID {
				v.CallID = id
				set(i, v)
			}
		case llmprovider.FunctionCallOutputItem:
			id, ok := latest[v.CallID]
			if !ok {
				id = rule.valid(v.CallID)
			}
			if id != v.CallID {
				v.CallID = id
				set(i, v)
			}
		}
	}
	if out == nil {
		return items
	}
	return out
}

// valid is id with refused characters replaced and cut to MaxLen; an empty
// id is "call".
func (r CallIDRule) valid(id string) string {
	id = r.Refused.ReplaceAllString(id, "_")
	if id == "" {
		id = "call"
	}
	return r.cut(id, r.MaxLen)
}

// unique is id, or id with the first "_n" suffix no earlier call used, cut
// so that it stays within MaxLen.
func (r CallIDRule) unique(id string, used map[string]bool) string {
	for n := 2; used[id]; n++ {
		suffix := "_" + strconv.Itoa(n)
		candidate := r.cut(id, r.MaxLen-len(suffix)) + suffix
		if !used[candidate] {
			return candidate
		}
	}
	return id
}

// cut trims id to n bytes. A valid id is ASCII, so a byte cut is a rune cut.
func (r CallIDRule) cut(id string, n int) string {
	if len(id) > n {
		return id[:n]
	}
	return id
}
