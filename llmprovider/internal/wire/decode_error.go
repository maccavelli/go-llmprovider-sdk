package wire

import (
	"errors"
	"fmt"
	"net"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// ReplyLimit bounds one reply's body, as the stream readers' limit does. A
// reply above it cannot be read whole, and is ErrIncomplete (0020-MADR F9).
const ReplyLimit = 16 << 20

// kinds are the error kinds every error unwraps to (R25).
var kinds = []error{
	llmprovider.ErrRateLimited, llmprovider.ErrAuthFailure, llmprovider.ErrNotPermitted,
	llmprovider.ErrInvalidRequest, llmprovider.ErrProviderUnavailable, llmprovider.ErrUnsupported,
	llmprovider.ErrInvalidProvider,
}

// DecodeError gives a failure to read a reply its kind and its provider
// (R25, R27; 0020-MADR F9, Q2 a). An error that has a kind keeps it. A
// network read failure stays a failure to reach the service, which WithRetry
// retries. Any other, such as a body that is not the expected JSON, empty, or
// cut at ReplyLimit, is ErrIncomplete: the service answered and billed, and
// asking again would buy the same answer.
func DecodeError(provider string, err error) error {
	if err == nil {
		return nil
	}
	for _, kind := range kinds {
		if errors.Is(err, kind) {
			return err
		}
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return fmt.Errorf("%s: read reply: %w", provider, err)
	}
	return fmt.Errorf("%w: %s: %w", llmprovider.ErrIncomplete, provider, err)
}
