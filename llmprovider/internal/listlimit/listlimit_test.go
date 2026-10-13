package listlimit

import (
	"slices"
	"testing"
)

// TestOf: the last Value wins, other values are ignored, and none is unset.
func TestOf(t *testing.T) {
	for _, tc := range []struct {
		name    string
		values  []any
		n       int
		wantSet bool
	}{
		{"none", nil, 0, false},
		{"other values only", []any{"x", 3}, 0, false},
		{"one", []any{Value(6)}, 6, true},
		{"the last wins", []any{Value(6), "x", Value(0)}, 0, true},
		{"negative", []any{Value(-1)}, -1, true},
	} {
		if n, set := Of(tc.values); n != tc.n || set != tc.wantSet {
			t.Errorf("%s: Of = %d, %v; want %d, %v", tc.name, n, set, tc.n, tc.wantSet)
		}
	}
}

// TestCut: a positive n below the length cuts; any other n copies the list
// whole; the result never shares the input's array.
func TestCut(t *testing.T) {
	ids := []string{"a", "b", "c"}
	for _, tc := range []struct {
		n    int
		want []string
	}{{2, []string{"a", "b"}}, {3, ids}, {9, ids}, {0, ids}, {-1, ids}} {
		got := Cut(ids, tc.n)
		if !slices.Equal(got, tc.want) {
			t.Errorf("Cut(%v, %d) = %v; want %v", ids, tc.n, got, tc.want)
		}
		if len(got) > 0 && &got[0] == &ids[0] {
			t.Errorf("Cut(%v, %d) shares the input's array", ids, tc.n)
		}
	}
	if got := Cut(nil, 2); got != nil {
		t.Errorf("Cut(nil, 2) = %v; want nil", got)
	}
}
