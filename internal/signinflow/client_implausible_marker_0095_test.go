package signinflow

import (
	"errors"
	"fmt"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/internal/recorderr"
)

// TestUndecodable_Implausible_0095 (0095-MADR FU10): Undecodable takes the
// module's marker for a record naming a client id that is not printable
// text, found through any wrapping; text saying so is not taken.
func TestUndecodable_Implausible_0095(t *testing.T) {
	marked := &recorderr.Implausible{Err: errors.New("names a client id that is not printable text")}
	for _, c := range []struct {
		name string
		err  error
		want bool
	}{
		{"the marker", marked, true},
		{"the marker, wrapped twice", fmt.Errorf("a: %w", fmt.Errorf("b: %w", marked)), true},
		{"text alone", errors.New("names a client id that is not printable text"), false},
	} {
		if got := Undecodable(c.err); got != c.want {
			t.Errorf("%s: Undecodable = %v, want %v", c.name, got, c.want)
		}
	}
}
