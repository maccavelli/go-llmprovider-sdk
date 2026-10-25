package wire

import (
	"context"
	"io"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// MaxSends bounds how many times one request is built and sent while its
// retry check says to resend: at most 7, one per effort rung and one for the
// other thinking shape, a forced-choice refusal counted against the same
// bound (0062-MADR D5, Q5 (a)). A send renewed by Reauth for a refused
// credential is part of the send it renews.
const MaxSends = 7

// Retry decides, when a send has failed after Reauth, whether its request is
// built again and sent once more. It returns true to resend, or false and the
// error to return in place of err. It is never consulted on success, nor
// after a stream's first event, nor once MaxSends sends have been made
// (0034-MADR D3; 0062-MADR D5).
type Retry func(err error) (resend bool, replaced error)

// Send builds a request with build, posts it through Reauth, and decodes the
// reply with decode. While it fails and retry says to, the request is built
// again, so that it reflects what retry learned, and sent once more, through
// Reauth again, at most MaxSends times in all; the last failure is returned.
// A nil retry sends once. provider labels a failure to fetch a token.
func Send[T any](ctx context.Context, provider string, src llmprovider.TokenSource, build func(llmprovider.Token) Call,
	decode func(io.Reader) (T, error), retry Retry) (T, error) {
	send := func(token llmprovider.Token) (T, error) { return Post(ctx, build(token), token, decode) }
	out, err := Reauth(ctx, provider, src, send)
	for sends := 1; err != nil && retry != nil && sends < MaxSends; sends++ {
		resend, replaced := retry(err)
		if !resend {
			var zero T
			return zero, replaced
		}
		out, err = Reauth(ctx, provider, src, send)
	}
	return out, err
}
