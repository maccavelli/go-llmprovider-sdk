package redact

import (
	"bytes"
	"slices"
	"testing"
)

// 0028-PLAN D9: each distinct anchor is looked up once per Redact call, by
// its rarest byte. The lookup must answer exactly as bytes.Contains does.

// FuzzAnchorIn_MatchesContains: anchorIn agrees with bytes.Contains for
// every anchor of passes, on any lower-cased text.
func FuzzAnchorIn_MatchesContains(f *testing.F) {
	for _, s := range []string{
		"", "token", "xtoke", "toke", "kkkkk", "aq.ab", "q.ab", "aq.a", "1//0", "1//", "the bearer token_x",
		"sk-sk-", "ollamaollama", "-----begi", "-----begin", "api_org", "ya29", "ya29.", "github_pat",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		lower := asciiLower(nil, []byte(s))
		for j, a := range anchors {
			if got, want := anchorIn(lower, a, anchorRare[j]), bytes.Contains(lower, []byte(a)); got != want {
				t.Fatalf("anchorIn(%q, %q) = %v; bytes.Contains = %v", lower, a, got, want)
			}
		}
	})
}

// TestRarestOffsets: the offset is of the byte latest in byteCommonness, or
// of the first byte it does not list.
func TestRarestOffsets(t *testing.T) {
	got := rarestOffsets([]string{"token", "aq.ab", "e@t~", "1//0", "e"})
	if want := []int{2, 1, 1, 0, 0}; !slices.Equal(got, want) {
		t.Errorf("rarestOffsets = %v; want %v", got, want)
	}
}

// TestIndexAnchors_SharedOnce: an anchor two passes name is numbered once,
// and each pass keeps every one of its anchors.
func TestIndexAnchors_SharedOnce(t *testing.T) {
	if len(passAnchors) != len(passes) {
		t.Fatalf("%d passes indexed; want %d", len(passAnchors), len(passes))
	}
	if n := len(slices.Compact(slices.Sorted(slices.Values(anchors)))); n != len(anchors) {
		t.Errorf("%d anchors, %d distinct: %v", len(anchors), n, anchors)
	}
	for i, p := range passes {
		var names []string
		for _, j := range passAnchors[i] {
			names = append(names, anchors[j])
		}
		if !slices.Equal(names, p.anchors) {
			t.Errorf("pass %d: anchors %v; want %v", i, names, p.anchors)
		}
	}
}

// TestIndexAnchors_Bound: more distinct anchors than anchorsSeen holds is a
// programming error, caught at init.
func TestIndexAnchors_Bound(t *testing.T) {
	var many []string
	for i := range maxAnchors + 1 {
		many = append(many, string(rune('a'+i/26))+string(rune('a'+i%26)))
	}
	defer func() {
		if recover() == nil {
			t.Error("indexAnchors took more than maxAnchors anchors; want a panic")
		}
	}()
	indexAnchors([]pass{{anchors: many}})
}
