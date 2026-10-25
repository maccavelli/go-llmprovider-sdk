package responses

import (
	"errors"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestDecode_EmptyIsErrEmptyAnswer_0061 (0061-MADR D1): a completed answer
// with no output is an error matching ErrEmptyAnswer.
func TestDecode_EmptyIsErrEmptyAnswer_0061(t *testing.T) {
	_, err := Decode(strings.NewReader(`{"id":"resp_1","status":"completed","output":[]}`))
	if !errors.Is(err, llmprovider.ErrEmptyAnswer) || !errors.Is(err, llmprovider.ErrIncomplete) {
		t.Errorf("%v; want an error matching ErrEmptyAnswer and ErrIncomplete", err)
	}
}
