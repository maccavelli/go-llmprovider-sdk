package recorderr

import (
	"errors"
	"fmt"
	"testing"
)

var errCause = errors.New("openai.client is larger than 65536 bytes")

// The marked error shows its cause's text and reaches the cause through
// errors.Is and errors.As, wrapped or not.
func TestOversizeShowsAndUnwrapsItsCause(t *testing.T) {
	err := fmt.Errorf("outer: %w", &Oversize{Err: fmt.Errorf("FileTokenStore client decode: %w", errCause)})
	var marked *Oversize
	if !errors.As(err, &marked) {
		t.Fatalf("errors.As did not find the marker in %v", err)
	}
	if got, want := marked.Error(), "FileTokenStore client decode: openai.client is larger than 65536 bytes"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
	if !errors.Is(err, errCause) {
		t.Errorf("errors.Is(err, cause) = false; the marker hides its cause")
	}
}
