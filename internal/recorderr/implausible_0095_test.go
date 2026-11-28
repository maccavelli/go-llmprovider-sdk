package recorderr

import (
	"errors"
	"fmt"
	"testing"
)

// TestImplausibleShowsAndUnwrapsItsCause_0095 (0095-MADR FU10): the marker of
// a record naming a client id that is not printable text shows its cause's
// text and reaches the cause through errors.Is and errors.As, wrapped or not.
func TestImplausibleShowsAndUnwrapsItsCause_0095(t *testing.T) {
	cause := errors.New("openai.client names a client id that is not printable text")
	err := fmt.Errorf("outer: %w", &Implausible{Err: fmt.Errorf("FileTokenStore client decode: %w", cause)})
	var marked *Implausible
	if !errors.As(err, &marked) {
		t.Fatalf("errors.As did not find the marker in %v", err)
	}
	if got, want := marked.Error(), "FileTokenStore client decode: openai.client names a client id that is not printable text"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
	if !errors.Is(err, cause) {
		t.Errorf("errors.Is(err, cause) = false; the marker hides its cause")
	}
	var oversize *Oversize
	if errors.As(err, &oversize) {
		t.Errorf("errors.As found an Oversize in %v; the two markers are apart", err)
	}
}
