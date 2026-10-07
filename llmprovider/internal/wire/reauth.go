package wire

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Reauth fetches a token from src and runs send with it, and once more with a
// fresh token after a refused credential when src can be told its token was refused
// (llmprovider.InvalidatingSource): a CommandToken reruns its command and an
// OAuthSession refreshes. A source that is also a
// llmprovider.TokenInvalidator is told which token was refused, so a late 401
// on a token already replaced keeps the fresh one (0021-MADR D5). A source
// that cannot renew, such as a Kilo device-login session, answers the second
// fetch with ErrAuthFailure, which is returned (0020-MADR F2; 0017-MADR D2,
// D3). provider labels a failure to fetch the token.
//
// A refused credential is an *APIError of kind ErrAuthFailure, whatever its
// status: Gemini refuses a key with HTTP 400 API_KEY_INVALID (0026-MADR F6).
// The kind is compared, not matched with errors.Is, since a not-permitted 403
// also matches ErrAuthFailure through its legacy status sentinel. An error
// with no kind is refused when its status is 401.
func Reauth[T any](ctx context.Context, provider string, src llmprovider.TokenSource, send func(llmprovider.Token) (T, error)) (T, error) {
	var zero T
	token, err := src.Token(ctx)
	if err != nil {
		return zero, fmt.Errorf("llmprovider: %s: acquire token: %w", provider, err)
	}
	out, err := send(token)
	if !credentialRefused(err) {
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
	refused := err
	if token, err = src.Token(ctx); err != nil {
		// The refusal's *APIError stays in the error, its status and kind
		// with it, beside the renewal's failure (R24; 0026-MADR F57, D9).
		return zero, errors.Join(refused, fmt.Errorf("llmprovider: %s: acquire token: %w", provider, err))
	}
	return send(token)
}

// credentialRefused reports whether err says the credential was refused, and
// so a renewed one may succeed.
func credentialRefused(err error) bool {
	apiErr, ok := errors.AsType[*llmprovider.APIError](err)
	switch {
	case !ok:
		return false
	case apiErr.Kind == nil:
		return apiErr.Status == http.StatusUnauthorized
	default:
		return apiErr.Kind == llmprovider.ErrAuthFailure //nolint:errorlint // the kind itself, not what it wraps
	}
}
