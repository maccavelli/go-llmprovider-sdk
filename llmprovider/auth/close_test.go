package auth

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// closeFails is a body whose Close fails.
type closeFails struct{ io.Reader }

func (closeFails) Close() error { return errors.New("close broke") }

// TestFetchJWKS_ReturnsACloseFailure (0015-PLAN S10): auth has no logger, so a
// failure to close the keys response is returned with the result.
func TestFetchJWKS_ReturnsACloseFailure(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Request: r,
			Body: closeFails{strings.NewReader(`{"keys":[]}`)}}, nil
	})}
	_, err := fetchJWKS(context.Background(), client, "https://issuer.test/jwks")
	if err == nil || !strings.Contains(err.Error(), "close response body: close broke") {
		t.Fatalf("err = %v, want the close failure returned", err)
	}
}
