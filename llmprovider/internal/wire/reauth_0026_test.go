package wire

import (
	"context"
	"net/http"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestReauth_RerunsOnAuthFailureKind (0026-MADR F6, Q3 a): a refused key is
// rerun by its kind, not only by a 401: Gemini refuses one with HTTP 400
// API_KEY_INVALID, of kind ErrAuthFailure. A 403 of kind ErrNotPermitted is
// a valid key refused something, and is not rerun.
func TestReauth_RerunsOnAuthFailureKind(t *testing.T) {
	for _, c := range []struct {
		name        string
		first       error
		wantSends   int
		invalidates int
	}{
		{"gemini 400 API_KEY_INVALID", &llmprovider.APIError{Status: http.StatusBadRequest, Kind: llmprovider.ErrAuthFailure,
			Code: "API_KEY_INVALID"}, 2, 1},
		{"403 auth failure", &llmprovider.APIError{Status: http.StatusForbidden, Kind: llmprovider.ErrAuthFailure}, 2, 1},
		{"403 not permitted", &llmprovider.APIError{Status: http.StatusForbidden, Kind: llmprovider.ErrNotPermitted}, 1, 0},
		{"401 quota exhausted", &llmprovider.APIError{Status: http.StatusUnauthorized, Kind: llmprovider.ErrQuotaExhausted}, 1, 0},
		{"400 invalid request", &llmprovider.APIError{Status: http.StatusBadRequest, Kind: llmprovider.ErrInvalidRequest}, 1, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			src := &countingSource{}
			sends := 0
			_, _ = Reauth(context.Background(), "p", src, func(llmprovider.Token) (int, error) {
				sends++
				if sends == 1 {
					return 0, c.first
				}
				return sends, nil
			})
			if sends != c.wantSends || src.invalidated != c.invalidates {
				t.Errorf("sends = %d, invalidations = %d; want %d and %d", sends, src.invalidated, c.wantSends, c.invalidates)
			}
		})
	}
}
