package nilsource

import "testing"

type stringer0050 interface{ String() string }

// TestIs_0050 (0050-MADR D1): a nil interface is not a typed nil; a nil value
// of each nilable kind behind an interface is; a non-nil one, and a value of
// a kind that cannot be nil, is not.
func TestIs_0050(t *testing.T) {
	var (
		nilPointer *int
		nilMap     map[string]int
		nilSlice   []int
		nilFunc    func()
		nilChan    chan int
		nilIface   stringer0050
	)
	one := 1
	for _, c := range []struct {
		name string
		v    any
		want bool
	}{
		{"a nil interface", nil, false},
		{"a nil pointer", nilPointer, true},
		{"a nil map", nilMap, true},
		{"a nil slice", nilSlice, true},
		{"a nil func", nilFunc, true},
		{"a nil chan", nilChan, true},
		{"a nil interface value, passed as any", nilIface, false},
		{"a pointer", &one, false},
		{"a map", map[string]int{}, false},
		{"a slice", []int{}, false},
		{"a func", func() {}, false},
		{"a chan", make(chan int), false},
		{"an int", 0, false},
		{"a string", "", false},
		{"a struct", struct{}{}, false},
	} {
		if got := Is(c.v); got != c.want {
			t.Errorf("Is(%s) = %v, want %v", c.name, got, c.want)
		}
	}
}
