package wire

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Reauth fetches a token from src and runs send with it, and once more with a
// fresh token after an HTTP 401 when src can be told its token was refused
// (llmprovider.InvalidatingSource): a CommandToken reruns its command and an
// OAuthSession refreshes. A source that is also a
// llmprovider.TokenInvalidator is told which token was refused, so a late 401
// on a token already replaced keeps the fresh one (0021-MADR D5). A source
// that cannot renew, such as a Kilo device-login session, answers the second
// fetch with ErrAuthFailure, which is returned (0020-MADR F2; 0017-MADR D2,
// D3). provider labels a failure to fetch the token.
func Reauth[T any](ctx context.Context, provider string, src llmprovider.TokenSource, send func(llmprovider.Token) (T, error)) (T, error) {
	var zero T
	token, err := src.Token(ctx)
	if err != nil {
		return zero, fmt.Errorf("llmprovider: %s: acquire token: %w", provider, err)
	}
	out, err := send(token)
	var apiErr *llmprovider.APIError
	if err == nil || !errors.As(err, &apiErr) || apiErr.Status != http.StatusUnauthorized {
		return out, err
	}
	source, ok := src.(llmprovider.InvalidatingSource)
	if !ok {
		return out, err
	}
	if invalidator, ok := src.(llmprovider.TokenInvalidator); ok {
		invalidator.InvalidateToken(token)
	} else {
		source.Invalidate()
	}
	if token, err = src.Token(ctx); err != nil {
		return zero, fmt.Errorf("llmprovider: %s: acquire token: %w", provider, err)
	}
	return send(token)
}
