package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// assertKind0049 fails t unless err matches kind and names step.
func assertKind0049(t *testing.T, err, kind error, step string) {
	t.Helper()
	if !errors.Is(err, kind) || !strings.Contains(err.Error(), step) {
		t.Fatalf("err = %v; want one matching %v that names %q", err, kind, step)
	}
}

// TestDeviceCode_OwnExpiryIsAuthFailure_0049 (0049-MADR D2b): a device code
// that passes its own expiry matches ErrAuthFailure, as the service's
// expired_token does, and still says it expired.
func TestDeviceCode_OwnExpiryIsAuthFailure_0049(t *testing.T) {
	clock := newFakeOAuthClock()
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		writeTestJSON(t, w, fmt.Sprintf(`{"device_authorization_endpoint":%q,"token_endpoint":%q}`, srv.URL+"/device", srv.URL+"/token"))
	})
	mux.HandleFunc("/device", func(w http.ResponseWriter, _ *http.Request) {
		writeTestJSON(t, w, `{"device_code":"device-secret","user_code":"ABCD-EFGH","verification_uri":"https://example.test/activate","expires_in":1,"interval":1}`)
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		writeTestJSON(t, w, `{"error":"authorization_pending"}`)
	})
	_, err := LoginDeviceOAuth(context.Background(), llmprovider.ProviderGrok, OAuthFlowOptions{
		HTTPClient: srv.Client(), ClientID: "test-client", Issuer: srv.URL, now: clock.now, sleep: clock.sleep,
	})
	assertKind0049(t, err, llmprovider.ErrAuthFailure, "device code expired")
}

// TestLoginBrowserOAuth_CodexDenialIsAuthFailure_0049 (0049-MADR D2a): the
// workspace-without-Codex access_denied matches ErrAuthFailure too, and keeps
// its explanation.
func TestLoginBrowserOAuth_CodexDenialIsAuthFailure_0049(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	f := newFake0048(t, llmprovider.ProviderGrok, nil)
	err := browserLogin0048(t, ctx, f, OAuthFlowOptions{
		OpenURL: visit0048(f.ti, "error=access_denied&error_description=missing_codex_entitlement")})
	assertKind0049(t, err, llmprovider.ErrAuthFailure, "Codex is not enabled for your workspace")
}
