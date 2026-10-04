package wire

import (
	"errors"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestDecodeError (0020-MADR F9, Q2 a): an unreadable answer is ErrIncomplete
// and names its provider; an error with a kind keeps it; a network failure
// stays a failure to reach the service, which WithRetry retries.
func TestDecodeError(t *testing.T) {
	if DecodeError("together", nil) != nil {
		t.Error("DecodeError(nil) is not nil")
	}
	syntax := DecodeError("together", io.ErrUnexpectedEOF)
	if !errors.Is(syntax, llmprovider.ErrIncomplete) || !strings.Contains(syntax.Error(), "together") {
		t.Errorf("an unreadable answer = %v, want ErrIncomplete naming together", syntax)
	}
	quota := &llmprovider.APIError{Kind: llmprovider.ErrQuotaExhausted}
	if got := DecodeError("together", quota); !errors.Is(got, llmprovider.ErrQuotaExhausted) || errors.Is(got, llmprovider.ErrIncomplete) {
		t.Errorf("a kinded error = %v, want its own kind kept", got)
	}
	network := &net.OpError{Op: "read", Err: errors.New("connection reset by peer")}
	if got := DecodeError("together", network); errors.Is(got, llmprovider.ErrIncomplete) || !errors.As(got, new(net.Error)) {
		t.Errorf("a network read failure = %v, want it kindless and still a net.Error", got)
	}
}
