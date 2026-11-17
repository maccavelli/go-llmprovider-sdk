package signinflow

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// kiloAnswer0080 is the error Kilo's profile gives for status, as
// auth.KiloProfile returns it: the module's own classification of the
// answer (0080-MADR C8, C10, C11), over a body that holds a fixed marker
// where a service's text would be.
func kiloAnswer0080(t *testing.T, status int) error {
	t.Helper()
	resp := &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}},
		Body: io.NopCloser(strings.NewReader(`{"error":"SERVICE-MESSAGE-0080"}`)), Request: &http.Request{}}
	err := llmprovider.ClassifyHTTPError(string(llmprovider.ProviderKilo), resp)
	if err == nil {
		t.Fatalf("status %d classified as no error", status)
	}
	return err
}

// TestVerifyKilo_Kinds_0080 (0080-MADR D1, D5, D6, D9, D10): the check's two
// results. A profile that answers is neither; a 401 is the refusal, the
// profile's own *APIError, matching ErrAuthFailure; a 403 (every Kilo 403 is
// ErrNotPermitted), a 5xx, a machine offline and a body that does not decode
// are not a refusal, and come back as the cause of an unverified result; with
// an organization, no request is made at all.
func TestVerifyKilo_Kinds_0080(t *testing.T) {
	const fakeKey = "fake-kilo-key-0080"
	decodeErr := errors.New("kilo: decode profile: invalid character '<' looking for beginning of value")
	offlineErr := fmt.Errorf("kilo: profile: %w", llmprovider.ErrOffline)
	for _, tc := range []struct {
		name         string
		organization string
		read         error
		refused      error // a sentinel the refusal must match; nil for none
		status       int   // the refusal's or the cause's HTTP status; 0 for none
		unverified   error // a sentinel the cause must match; nil for none
		reads        int
	}{
		{name: "the profile answers", reads: 1},
		{name: "401", read: kiloAnswer0080(t, http.StatusUnauthorized), refused: llmprovider.ErrAuthFailure,
			status: http.StatusUnauthorized, reads: 1},
		{name: "403", read: kiloAnswer0080(t, http.StatusForbidden), unverified: llmprovider.ErrNotPermitted,
			status: http.StatusForbidden, reads: 1},
		{name: "500", read: kiloAnswer0080(t, http.StatusInternalServerError), unverified: llmprovider.ErrProviderUnavailable,
			status: http.StatusInternalServerError, reads: 1},
		{name: "503", read: kiloAnswer0080(t, http.StatusServiceUnavailable), unverified: llmprovider.ErrProviderUnavailable,
			status: http.StatusServiceUnavailable, reads: 1},
		{name: "offline", read: offlineErr, unverified: llmprovider.ErrOffline, reads: 1},
		{name: "a body that does not decode", read: decodeErr, unverified: decodeErr, reads: 1},
		{name: "an organization, whose listing checks", organization: "org-0080", read: kiloAnswer0080(t, http.StatusUnauthorized), reads: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reads := 0
			var gotToken string
			var gotOpts int
			read := func(_ context.Context, token string, opts ...llmprovider.Option) (auth.KiloAccount, error) {
				reads++
				gotToken, gotOpts = token, len(opts)
				return auth.KiloAccount{}, tc.read
			}
			refused, unverified := VerifyKilo(context.Background(), read, fakeKey, tc.organization,
				llmprovider.WithBaseURL("https://kilo.example.test/api/gateway"))
			if reads != tc.reads {
				t.Fatalf("%d profile reads; want %d", reads, tc.reads)
			}
			if reads == 1 && (gotToken != fakeKey || gotOpts != 1) {
				t.Errorf("read given the key %t and %d options; want the key and the one option", gotToken == fakeKey, gotOpts)
			}
			switch {
			case tc.refused != nil:
				apiErr, isAPI := errors.AsType[*llmprovider.APIError](refused)
				if !errors.Is(refused, tc.refused) || errors.Is(refused, llmprovider.ErrNotPermitted) || !isAPI || apiErr.Status != tc.status {
					t.Errorf("refused %v; want the profile's *APIError, status %d, matching %v", refused, tc.status, tc.refused)
				}
				if unverified != nil {
					t.Errorf("unverified %v; want none beside a refusal", unverified)
				}
			case tc.unverified != nil:
				if refused != nil {
					t.Errorf("refused %v; want no refusal, only a cause", refused)
				}
				if !errors.Is(unverified, tc.unverified) {
					t.Errorf("unverified %v; want the cause, matching %v", unverified, tc.unverified)
				}
				if apiErr, isAPI := errors.AsType[*llmprovider.APIError](unverified); tc.status != 0 && (!isAPI || apiErr.Status != tc.status) {
					t.Errorf("unverified %v; want the profile's *APIError, status %d", unverified, tc.status)
				}
			default:
				if refused != nil || unverified != nil {
					t.Errorf("refused %v, unverified %v; want neither", refused, unverified)
				}
			}
		})
	}
}
