package wire

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// failingRenewal refuses to renew: it hands out one token, then fails.
type failingRenewal struct{ invalidated bool }

func (s *failingRenewal) Token(context.Context) (llmprovider.Token, error) {
	if s.invalidated {
		return llmprovider.Token{}, errors.New("oauth: no refresh token")
	}
	return llmprovider.Token{Value: "t"}, nil
}

func (s *failingRenewal) Invalidate() { s.invalidated = true }

// TestReauth_FailedRenewalKeepsTheRefusal (0026-MADR F57, the PLAN's
// deviation D9): when renewing a refused credential fails, the error still
// holds the refusal's *APIError, with its status and kind (R24), and the
// renewal's failure beside it.
func TestReauth_FailedRenewalKeepsTheRefusal(t *testing.T) {
	refusal := &llmprovider.APIError{Provider: "p", Status: http.StatusUnauthorized, Kind: llmprovider.ErrAuthFailure}
	_, err := Reauth(context.Background(), "p", &failingRenewal{}, func(llmprovider.Token) (int, error) {
		return 0, refusal
	})
	apiErr, ok := errors.AsType[*llmprovider.APIError](err)
	if !ok || apiErr != refusal || !errors.Is(err, llmprovider.ErrAuthFailure) {
		t.Fatalf("Reauth = %v; want the refusal's *APIError, matching ErrAuthFailure", err)
	}
	if err == nil || !strings.Contains(err.Error(), "no refresh token") {
		t.Fatalf("Reauth = %v; want the renewal's failure too", err)
	}
}
