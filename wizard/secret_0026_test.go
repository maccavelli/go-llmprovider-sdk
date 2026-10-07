package wizard

import (
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"
)

// failingWriter refuses every write, as a closed terminal does.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("write: broken pipe") }

// TestTextPrompter_SecretSurfacesWriteError (0026-MADR F49): a failed write
// is returned by Secret, as Input returns one, not at the next prompt.
func TestTextPrompter_SecretSurfacesWriteError(t *testing.T) {
	p := &TextPrompter{In: strings.NewReader("sk-key-1234\n"), Out: failingWriter{}}
	if value, err := p.Secret("API key"); err == nil || value != "" {
		t.Fatalf("Secret = %q, %v; want the write error and no value", value, err)
	}
	// The first end of input still answers with what was read, as Input's.
	q := &TextPrompter{In: strings.NewReader("sk-key-1234"), Out: io.Discard}
	if value, err := q.Secret("API key"); err != nil || value != "sk-key-1234" {
		t.Fatalf("Secret at end of input = %q, %v; want the value", value, err)
	}
}

// TestTextPrompter_SecretRefusesPartialEntry (0026-MADR F50): in raw mode, a
// read error in mid-entry, such as a hang-up mid-paste, is an error with no
// value: a truncated key is never kept.
func TestTextPrompter_SecretRefusesPartialEntry(t *testing.T) {
	for name, err := range map[string]error{"hang-up": errors.New("read: input/output error"), "end of input": io.EOF} {
		p := &TextPrompter{In: io.MultiReader(strings.NewReader("sk-partial-k"), iotest.ErrReader(err)), Out: io.Discard}
		value, got := p.readMasked("API key")
		if value != "" || !errors.Is(got, err) {
			t.Errorf("%s: readMasked = %q, %v; want no value and the error", name, value, got)
		}
	}
}
