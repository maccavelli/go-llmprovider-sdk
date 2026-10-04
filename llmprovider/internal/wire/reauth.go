package wire

import (
	"errors"
	"net/http"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Reauth runs send, and once more after an HTTP 401 when src can be told its
// token was refused (llmprovider.InvalidatingSource): a CommandToken reruns
// its command and an OAuthSession refreshes. send must fetch its token from
// src on every call, so the second send carries a fresh one. A source that
// cannot renew, such as a Kilo device-login session, answers that second
// fetch with ErrAuthFailure, which is returned (0020-MADR F2; 0017-MADR D2,
// D3).
func Reauth[T any](src llmprovider.TokenSource, send func() (T, error)) (T, error) {
	out, err := send()
	var apiErr *llmprovider.APIError
	if err == nil || !errors.As(err, &apiErr) || apiErr.Status != http.StatusUnauthorized {
		return out, err
	}
	source, ok := src.(llmprovider.InvalidatingSource)
	if !ok {
		return out, err
	}
	source.Invalidate()
	return send()
}
