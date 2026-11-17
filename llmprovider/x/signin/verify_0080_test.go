package signin_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/x/signin"
)

// kilo0080 is an in-process stand-in for Kilo's profile (0080-MADR, "Live
// probes of 2026-11-16", L13 to L16, and the owner's run of 2026-11-17): 401
// "Unauthorized" with no credential, 401 "Invalid token" with one it does not
// know, 200 with a profile for the fake key it accepts, or one fixed status
// when status is set. It counts the requests it sees.
type kilo0080 struct {
	accepts  string
	status   int
	requests atomic.Int32
}

const (
	// A fixed fake key the stand-in accepts, and one it refuses.
	kiloGood0080 = "fake-kilo-good-key-0080"
	kiloBad0080  = "fake-kilo-bad-key-0080"
)

func (k *kilo0080) RoundTrip(r *http.Request) (*http.Response, error) {
	k.requests.Add(1)
	answer := func(status int, body string) (*http.Response, error) {
		return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}},
			Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	}
	token, _ := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	switch {
	case !strings.HasSuffix(r.URL.Path, "/api/profile"):
		return answer(http.StatusNotFound, `{}`)
	case k.status != 0:
		return answer(k.status, `{"error":"SERVICE-MESSAGE-0080"}`)
	case token == "":
		return answer(http.StatusUnauthorized, `{"error":"Unauthorized"}`)
	case token != k.accepts:
		return answer(http.StatusUnauthorized, `{"error":"Invalid token (req-0080)"}`)
	}
	return answer(http.StatusOK, `{"organizations":[],"hasPersonalAccount":true}`)
}

// TestVerifyKilo_0080 (0080-MADR D10): VerifyKilo is nil when the profile
// answers, when an organization is given (its listing checks, so nothing is
// requested) and when the profile cannot say (a 403, a 503); it matches
// llmprovider.ErrAuthFailure, with ErrorKindOf "auth" and the profile's
// *APIError, when Kilo refuses the key; and it matches ctx's error when ctx
// has ended. It never shows the key.
func TestVerifyKilo_0080(t *testing.T) {
	verify := func(ctx context.Context, k *kilo0080, key, organization string) error {
		return signin.VerifyKilo(ctx, key, organization, llmprovider.WithHTTPClient(&http.Client{Transport: k}))
	}
	t.Run("the profile answers", func(t *testing.T) {
		k := &kilo0080{accepts: kiloGood0080}
		if err := verify(context.Background(), k, kiloGood0080, ""); err != nil || k.requests.Load() != 1 {
			t.Errorf("error %v, %d requests; want nil, one request", err, k.requests.Load())
		}
	})
	t.Run("refused", func(t *testing.T) {
		k := &kilo0080{accepts: kiloGood0080}
		err := verify(context.Background(), k, kiloBad0080, "")
		apiErr, isAPI := errors.AsType[*llmprovider.APIError](err)
		if !errors.Is(err, llmprovider.ErrAuthFailure) || errors.Is(err, llmprovider.ErrNotPermitted) || !isAPI || apiErr.Status != http.StatusUnauthorized {
			t.Fatalf("error %v; want the profile's 401 *APIError matching ErrAuthFailure", err)
		}
		if kind := signin.ErrorKindOf(err); kind != authcontract.FailureAuth {
			t.Errorf("ErrorKindOf = %q; want %q", kind, authcontract.FailureAuth)
		}
		if strings.Contains(err.Error(), kiloBad0080) {
			t.Errorf("error text shows the key")
		}
	})
	t.Run("no credential", func(t *testing.T) {
		k := &kilo0080{accepts: kiloGood0080}
		if err := verify(context.Background(), k, "", ""); !errors.Is(err, llmprovider.ErrAuthFailure) {
			t.Errorf("error %v; want a refusal matching ErrAuthFailure", err)
		}
	})
	for _, status := range []int{http.StatusForbidden, http.StatusInternalServerError, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			k := &kilo0080{accepts: kiloGood0080, status: status}
			if err := verify(context.Background(), k, kiloGood0080, ""); err != nil || k.requests.Load() != 1 {
				t.Errorf("error %v, %d requests; want nil (the profile cannot say), one request", err, k.requests.Load())
			}
		})
	}
	t.Run("an organization", func(t *testing.T) {
		k := &kilo0080{accepts: kiloGood0080}
		if err := verify(context.Background(), k, kiloBad0080, "org-0080"); err != nil || k.requests.Load() != 0 {
			t.Errorf("error %v, %d requests; want nil and no request", err, k.requests.Load())
		}
	})
	t.Run("ctx ended", func(t *testing.T) {
		k := &kilo0080{accepts: kiloGood0080}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := verify(ctx, k, kiloGood0080, ""); !errors.Is(err, context.Canceled) {
			t.Errorf("error %v; want one matching context.Canceled", err)
		}
	})
}
