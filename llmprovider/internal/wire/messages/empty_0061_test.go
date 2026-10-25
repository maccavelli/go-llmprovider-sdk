package messages

import (
	"errors"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestDecode_EmptyIsErrEmptyAnswer_0061 (0061-MADR D1): each of the wire's
// no-content paths is an error matching ErrEmptyAnswer.
func TestDecode_EmptyIsErrEmptyAnswer_0061(t *testing.T) {
	for name, body := range map[string]string{
		"no content":                `{"type":"message","content":[],"stop_reason":"end_turn"}`,
		"cut while still reasoning": `{"type":"message","content":[{"type":"thinking","thinking":"hmm","signature":"s"}],"stop_reason":"max_tokens"}`,
	} {
		_, err := Decode(strings.NewReader(body))
		if !errors.Is(err, llmprovider.ErrEmptyAnswer) || !errors.Is(err, llmprovider.ErrIncomplete) {
			t.Errorf("%s: %v; want an error matching ErrEmptyAnswer and ErrIncomplete", name, err)
		}
	}
}
