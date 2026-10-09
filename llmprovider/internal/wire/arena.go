package wire

// Arena keeps strings for an encoder that points to them: one allocation
// for every string a request's items carry, where each pointer taken on its
// own would be one apiece (0028-PLAN D16).
type Arena struct{ strings []string }

// NewArena is an Arena with room for n strings. Past n it grows, and the
// pointers it gave keep pointing to the strings they were given.
func NewArena(n int) *Arena { return &Arena{strings: make([]string, 0, n)} }

// Keep returns a pointer to a copy of s.
func (a *Arena) Keep(s string) *string {
	a.strings = append(a.strings, s)
	return &a.strings[len(a.strings)-1]
}
