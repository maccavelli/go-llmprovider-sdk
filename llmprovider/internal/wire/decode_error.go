package wire

import (
	"errors"
	"fmt"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/transport"
)

// ReplyLimit bounds one reply's body. A reply above it cannot be read whole,
// and is ErrIncomplete (0020-MADR F9; 0021-MADR W6).
const ReplyLimit = 16 << 20

// kinds are the error kinds every error unwraps to (R25).
var kinds = []error{
	llmprovider.ErrRateLimited, llmprovider.ErrAuthFailure, llmprovider.ErrNotPermitted,
	llmprovider.ErrInvalidRequest, llmprovider.ErrProviderUnavailable, llmprovider.ErrUnsupported,
	llmprovider.ErrInvalidProvider,
}

// DecodeError gives a failure to read a reply its kind and its provider
// (R25, R27; 0020-MADR F9; 0021-MADR D1). readErr is the body's first read
// failure (transport.ReplyReader.ReadErr), or nil.
//
//   - A reply over ReplyLimit is ErrIncomplete: it was sent whole, and asking
//     again would buy the same answer.
//   - A read failure, such as a cut connection or the idle limit, is
//     ErrProviderUnavailable marked transport.AfterReply, which WithRetry
//     retries once.
//   - An error that has a kind keeps it.
//   - Any other, such as a body that is not the expected JSON, or empty, is
//     ErrIncomplete: the service answered and billed.
func DecodeError(provider string, err, readErr error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, transport.ErrReplyTooLarge):
		return fmt.Errorf("%w: %s: reply over %d MiB", llmprovider.ErrIncomplete, provider, ReplyLimit>>20)
	case readErr != nil:
		return fmt.Errorf("%w: %s: read reply: %w", llmprovider.ErrProviderUnavailable, provider, transport.AfterReply(readErr))
	}
	for _, kind := range kinds {
		if errors.Is(err, kind) {
			return err
		}
	}
	return fmt.Errorf("%w: %s: %w", llmprovider.ErrIncomplete, provider, err)
}
