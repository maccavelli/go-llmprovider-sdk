package chatcompletions

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
		"no choice":                 `{"choices":[]}`,
		"cut while still reasoning": `{"choices":[{"finish_reason":"length","message":{"role":"assistant","content":"","reasoning_content":"hmm"}}]}`,
		"nothing in the message":    `{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":""}}]}`,
	} {
		_, err := Decode(strings.NewReader(body))
		if !errors.Is(err, llmprovider.ErrEmptyAnswer) || !errors.Is(err, llmprovider.ErrIncomplete) {
			t.Errorf("%s: %v; want an error matching ErrEmptyAnswer and ErrIncomplete", name, err)
		}
	}
}
