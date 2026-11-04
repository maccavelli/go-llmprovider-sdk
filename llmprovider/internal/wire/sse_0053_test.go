package wire

import (
	"errors"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// TestReadEvents_DispatchErrorMarked_0053 (0053-MADR D3): a dispatch error,
// such as a failure event the service sent, comes back marked after the
// reply, at an event, with its text and kind unchanged; an event over
// EventLimit stays unmarked; a clean stop is nil. At the stream's end nothing
// is dispatched: the error is the early end's, not the failure (0069-MADR D2,
// as amended).
func TestReadEvents_DispatchErrorMarked_0053(t *testing.T) {
	failure := &llmprovider.APIError{Kind: llmprovider.ErrProviderUnavailable, Message: "server_error"}
	dispatch := func([]byte) (bool, error) { return false, failure }
	err := ReadEvents("p", strings.NewReader("data: x\n\n"), "done", dispatch)
	var apiErr *llmprovider.APIError
	if !transport.IsAfterReply(err) || !errors.Is(err, llmprovider.ErrProviderUnavailable) || !errors.As(err, &apiErr) ||
		err.Error() != failure.Error() {
		t.Errorf("at an event: %v (after the reply: %v); want the failure, marked, its text unchanged", err, transport.IsAfterReply(err))
	}
	dispatched := false
	err = ReadEvents("p", strings.NewReader("data: x"), "done", func([]byte) (bool, error) {
		dispatched = true
		return false, failure
	})
	if dispatched || !transport.IsAfterReply(err) || !errors.Is(err, llmprovider.ErrProviderUnavailable) || errors.As(err, &apiErr) {
		t.Errorf("at the stream end: dispatched %v, %v; want no dispatch, and the early end marked after the reply", dispatched, err)
	}
	big := "data: " + strings.Repeat("a", EventLimit+1) + "\n\n"
	if err := ReadEvents("p", strings.NewReader(big), "done", dispatch); !errors.Is(err, llmprovider.ErrIncomplete) || transport.IsAfterReply(err) {
		t.Errorf("an event over the limit: %v; want ErrIncomplete, unmarked", err)
	}
	stop := func([]byte) (bool, error) { return true, nil }
	if err := ReadEvents("p", strings.NewReader("data: x\n\n"), "done", stop); err != nil {
		t.Errorf("a clean stop: %v; want nil", err)
	}
}

// TestStreamIndex_Helpers_0053 (0053-MADR D2): the bound, for an integer and
// for a JSON number.
func TestStreamIndex_Helpers_0053(t *testing.T) {
	for _, i := range []int{0, 1, MaxStreamIndex - 1} {
		if err := StreamIndex("p", "f", i); err != nil {
			t.Errorf("StreamIndex(%d) = %v; want nil", i, err)
		}
	}
	for _, i := range []int{-1, MaxStreamIndex, 1 << 30} {
		if err := StreamIndex("p", "f", i); !errors.Is(err, llmprovider.ErrIncomplete) {
			t.Errorf("StreamIndex(%d) = %v; want ErrIncomplete", i, err)
		}
	}
	for _, f := range []float64{0, 7, MaxStreamIndex - 1} {
		if i, err := StreamIndexNumber("p", "f", f); err != nil || float64(i) != f {
			t.Errorf("StreamIndexNumber(%v) = %d, %v; want %v, nil", f, i, err, f)
		}
	}
	nan, inf := 0.0, 1.0
	for _, f := range []float64{-1, 1.5, MaxStreamIndex, 1e300, nan / nan, inf / 0} {
		if _, err := StreamIndexNumber("p", "f", f); !errors.Is(err, llmprovider.ErrIncomplete) {
			t.Errorf("StreamIndexNumber(%v) = %v; want ErrIncomplete", f, err)
		}
	}
}
