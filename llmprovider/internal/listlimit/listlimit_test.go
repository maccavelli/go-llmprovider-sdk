package listlimit

import (
	"slices"
	"testing"
)

// TestOf: the last Value wins, other values are ignored, and none is zero,
// no limit.
func TestOf(t *testing.T) {
	for _, tc := range []struct {
		name   string
		values []any
		n      int
	}{
		{"none", nil, 0},
		{"other values only", []any{"x", 3}, 0},
		{"one", []any{Value(6)}, 6},
		{"the last wins", []any{Value(6), "x", Value(0)}, 0},
		{"negative", []any{Value(-1)}, -1},
	} {
		if n := Of(tc.values); n != tc.n {
			t.Errorf("%s: Of = %d; want %d", tc.name, n, tc.n)
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
