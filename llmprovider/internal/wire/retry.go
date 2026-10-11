package wire

import (
	"context"
	"io"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Retry decides, when a send has failed after Reauth, whether its request is
// built again and sent once more. It returns true to resend, or false and the
// error to return in place of err. It is never consulted on success, nor after
// a stream's first event, nor on a resend (0034-MADR D3).
type Retry func(err error) (resend bool, replaced error)

// Send builds a request with build, posts it through Reauth, and decodes the
// reply with decode. When it fails and retry says to, the request is built
// again, so that it reflects what retry learned, and sent once more, through
// Reauth again. A nil retry sends once. provider labels a failure to fetch a
// token.
func Send[T any](ctx context.Context, provider string, src llmprovider.TokenSource, build func(llmprovider.Token) Call,
	decode func(io.Reader) (T, error), retry Retry) (T, error) {
	send := func(token llmprovider.Token) (T, error) { return Post(ctx, build(token), token, decode) }
	out, err := Reauth(ctx, provider, src, send)
	if err == nil || retry == nil {
		return out, err
	}
	resend, err := retry(err)
	if !resend {
		var zero T
		return zero, err
	}
	return Reauth(ctx, provider, src, send)
}
