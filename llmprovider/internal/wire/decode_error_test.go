package wire

import (
	"errors"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// TestDecodeError (0020-MADR F9, Q2 a; 0021-MADR D1, W6): an unreadable answer
// is ErrIncomplete and names its provider; an error with a kind keeps it; a
// failure to read the body is ErrProviderUnavailable marked after the reply;
// a reply over the limit is ErrIncomplete, never a cut connection.
func TestDecodeError(t *testing.T) {
	if DecodeError("together", nil, nil) != nil {
		t.Error("DecodeError(nil) is not nil")
	}
	syntax := DecodeError("together", io.ErrUnexpectedEOF, nil)
	if !errors.Is(syntax, llmprovider.ErrIncomplete) || !strings.Contains(syntax.Error(), "together") {
		t.Errorf("an unreadable answer = %v, want ErrIncomplete naming together", syntax)
	}
	quota := &llmprovider.APIError{Kind: llmprovider.ErrQuotaExhausted}
	if got := DecodeError("together", quota, nil); !errors.Is(got, llmprovider.ErrQuotaExhausted) || errors.Is(got, llmprovider.ErrIncomplete) {
		t.Errorf("a kinded error = %v, want its own kind kept", got)
	}
	network := &net.OpError{Op: "read", Err: errors.New("connection reset by peer")}
	got := DecodeError("together", network, network)
	if !errors.Is(got, llmprovider.ErrProviderUnavailable) || !transport.IsAfterReply(got) || !errors.As(got, new(net.Error)) {
		t.Errorf("a read failure = %v, want ErrProviderUnavailable, after the reply, still a net.Error", got)
	}
	cut := DecodeError("together", io.ErrUnexpectedEOF, io.ErrUnexpectedEOF)
	if !errors.Is(cut, llmprovider.ErrProviderUnavailable) || !transport.IsAfterReply(cut) {
		t.Errorf("a body cut short = %v, want ErrProviderUnavailable after the reply", cut)
	}
	big := DecodeError("together", transport.ErrReplyTooLarge, nil)
	if !errors.Is(big, llmprovider.ErrIncomplete) || transport.IsAfterReply(big) || !strings.Contains(big.Error(), "over 16 MiB") {
		t.Errorf("a reply over the limit = %v, want ErrIncomplete, over 16 MiB, not after the reply", big)
	}
}
