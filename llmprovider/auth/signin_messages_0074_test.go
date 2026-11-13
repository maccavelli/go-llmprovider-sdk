package auth

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// The six sites that return a denial or an expiry (0074-MADR D8, C42 to
// C47), run against fake issuers. This file calls only API that exists
// before 0074 Phase 3, so that it is seen green at the base commit: its
// literals are the old code's.

// signInRun0074 runs one of the six sites and gives its error.
type signInRun0074 struct {
	name string
	// message is the error's text, byte for byte, as it was before 0074.
	message string
	run     func(t *testing.T) error
}

// deviceWait0074 starts provider's device sign-in against a fake issuer
// with routes and waits on a clock that does not sleep.
func deviceWait0074(t *testing.T, provider llmprovider.ProviderID, routes func(f *fake0048) map[string]http.HandlerFunc) error {
	t.Helper()
	f := newFake0048(t, provider, routes)
	login, err := StartDeviceOAuth(context.Background(), provider, f.options(newFakeOAuthClock()))
	if err != nil {
		t.Fatalf("StartDeviceOAuth: %v", err)
	}
	session, err := login.Wait(context.Background())
	if session != nil {
		t.Errorf("Wait returned a session with its error")
	}
	return err
}

// signInRuns0074 are the six sites of C42 to C47.
func signInRuns0074() []signInRun0074 {
	return []signInRun0074{
		{
			name:    "browser denial",
			message: "llmprovider: authentication failed: oauth: authorization failed: access_denied",
			run: func(t *testing.T) error {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				f := newFake0048(t, llmprovider.ProviderGrok, nil)
				return browserLogin0048(t, ctx, f, OAuthFlowOptions{OpenURL: visit0048(f.ti, "error=access_denied")})
			},
		},
		{
			name:    "Grok device denial",
			message: "llmprovider: authentication failed: oauth: Grok device authorization denied",
			run: func(t *testing.T) error {
				return deviceWait0074(t, llmprovider.ProviderGrok, reply0048(t, "/token", http.StatusBadRequest, `{"error":"access_denied"}`))
			},
		},
		{
			name:    "Grok device expiry",
			message: "llmprovider: authentication failed: oauth: Grok device code expired",
			run: func(t *testing.T) error {
				return deviceWait0074(t, llmprovider.ProviderGrok, reply0048(t, "/token", http.StatusBadRequest, `{"error":"expired_token"}`))
			},
		},
		{
			name:    "shared device expiry",
			message: "llmprovider: authentication failed: oauth: device code expired",
			run: func(t *testing.T) error {
				return deviceWait0074(t, llmprovider.ProviderGrok, func(*fake0048) map[string]http.HandlerFunc {
					return map[string]http.HandlerFunc{
						"/device": answer0048(t, http.StatusOK,
							`{"device_code":"device-secret","user_code":"ABCD-EFGH","verification_uri":"https://example.test/activate","expires_in":1,"interval":1}`),
						"/token": answer0048(t, http.StatusBadRequest, `{"error":"authorization_pending"}`),
					}
				})
			},
		},
		{
			name:    "Kilo denial",
			message: "llmprovider: authentication failed: oauth: Kilo device authorization denied",
			run: func(t *testing.T) error {
				return deviceWait0074(t, llmprovider.ProviderKilo, reply0048(t, kiloPoll0048, http.StatusForbidden, ""))
			},
		},
		{
			name:    "Kilo expiry",
			message: "llmprovider: authentication failed: oauth: Kilo device code expired",
			run: func(t *testing.T) error {
				return deviceWait0074(t, llmprovider.ProviderKilo, reply0048(t, kiloPoll0048, http.StatusGone, ""))
			},
		},
	}
}

// TestSignInMessagesUnchanged_0074: each of the six errors' text is the
// literal it had before 0074 (D8: the new sentinels add no text).
func TestSignInMessagesUnchanged_0074(t *testing.T) {
	for _, tc := range signInRuns0074() {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run(t)
			if err == nil {
				t.Fatal("err = nil, want the sign-in's error")
			}
			if got := err.Error(); got != tc.message {
				t.Errorf("Error() = %q, want %q", got, tc.message)
			}
		})
	}
}
