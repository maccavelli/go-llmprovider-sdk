package auth

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestOAuthRefresh_EveryFailureHasAKind: the refresh failures that carried no
// kind now match one (R25; 0026-MADR F43). A failure to reach the issuer is
// unavailable; a 200 that is not a usable token reply is incomplete.
func TestOAuthRefresh_EveryFailureHasAKind(t *testing.T) {
	t.Run("transport failure", func(t *testing.T) {
		srv, _ := refreshServer(t, func(_ int32, w http.ResponseWriter, _ *http.Request) {})
		session := expiredSession(srv, DefaultOpenAIIssuer)
		srv.Close()
		if _, err := session.Token(context.Background()); !errors.Is(err, llmprovider.ErrProviderUnavailable) {
			t.Fatalf("Token = %v, want ErrProviderUnavailable", err)
		}
	})
	for _, c := range []struct{ name, body string }{
		{"a 200 that does not decode", `{"access_token":"a-new"`},
		{"a 200 with no access token", `{"refresh_token":"rt-new","expires_in":3600}`},
	} {
		t.Run(c.name, func(t *testing.T) {
			srv, _ := refreshServer(t, func(_ int32, w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, c.body) })
			if _, err := expiredSession(srv, DefaultOpenAIIssuer).Token(context.Background()); !errors.Is(err, llmprovider.ErrIncomplete) {
				t.Fatalf("Token = %v, want ErrIncomplete", err)
			}
		})
	}
}

// TestGrokDevice_DeniedAndExpiredAreAuthFailures: a device login the user
// denied, or let expire, is a failed sign-in (R25; 0026-MADR F43).
func TestGrokDevice_DeniedAndExpiredAreAuthFailures(t *testing.T) {
	for _, code := range []string{"access_denied", "expired_token"} {
		t.Run(code, func(t *testing.T) {
			mux := http.NewServeMux()
			srv := httptest.NewServer(mux)
			t.Cleanup(srv.Close)
			ti := newTestIssuer(mux, srv.URL)
			mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
				writeTestJSON(t, w, ti.discovery(t, map[string]string{"device_authorization_endpoint": srv.URL + "/device", "token_endpoint": srv.URL + "/token"}))
			})
			mux.HandleFunc("/device", func(w http.ResponseWriter, _ *http.Request) {
				writeTestJSON(t, w, `{"device_code":"device-secret","user_code":"ABCD-EFGH","verification_uri":"https://example.test/activate","expires_in":60,"interval":1}`)
			})
			mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				writeTestJSON(t, w, `{"error":"`+code+`"}`)
			})
			login, err := StartDeviceOAuth(context.Background(), llmprovider.ProviderGrok, grokDeviceOptions(srv, newFakeOAuthClock()))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := login.Wait(context.Background()); !errors.Is(err, llmprovider.ErrAuthFailure) {
				t.Fatalf("Wait = %v, want ErrAuthFailure", err)
			}
		})
	}
}

// TestKiloDevice_DeniedAndExpiredAreAuthFailures: the same for Kilo's 403
// and 410 (R25; 0026-MADR F43).
func TestKiloDevice_DeniedAndExpiredAreAuthFailures(t *testing.T) {
	for name, polls := range map[string][]int{
		"denied":  {http.StatusForbidden},
		"expired": {http.StatusAccepted, http.StatusGone},
	} {
		t.Run(name, func(t *testing.T) {
			srv, _ := kiloDeviceServer(t, http.StatusOK, polls...)
			login, err := StartDeviceOAuth(context.Background(), llmprovider.ProviderKilo, kiloOptions(srv, newFakeOAuthClock()))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := login.Wait(context.Background()); !errors.Is(err, llmprovider.ErrAuthFailure) {
				t.Fatalf("Wait = %v, want ErrAuthFailure", err)
			}
		})
	}
}
