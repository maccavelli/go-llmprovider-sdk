package wire

import (
	"errors"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestEmptyAnswer_Kind_0061 (0061-MADR D1): the empty answer is of kind
// ErrEmptyAnswer, still matching ErrIncomplete, with its message and reason
// as before.
func TestEmptyAnswer_Kind_0061(t *testing.T) {
	err := EmptyAnswer("x", llmprovider.FinishContentFilter)
	apiErr, ok := errors.AsType[*llmprovider.APIError](err)
	if !ok || !errors.Is(err, llmprovider.ErrEmptyAnswer) || !errors.Is(err, llmprovider.ErrIncomplete) {
		t.Fatalf("EmptyAnswer = %v; want an *APIError matching ErrEmptyAnswer and ErrIncomplete", err)
	}
	if !strings.HasSuffix(apiErr.Message, ": the answer has no content") || apiErr.Reason != "content_filter" {
		t.Errorf("Message %q, Reason %q; want the message unchanged and reason content_filter", apiErr.Message, apiErr.Reason)
	}
}
