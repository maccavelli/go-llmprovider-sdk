package auth

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// grokBrowserLogin runs a Grok browser login against a stub issuer whose
// token endpoint answers with tokenBody(issuer), and returns its error.
func grokBrowserLogin(t *testing.T, tokenBody func(ti *testIssuer) string) error {
	t.Helper()
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	ti := newTestIssuer(mux, srv.URL)
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		writeTestJSON(t, w, ti.discovery(t, map[string]string{"authorization_endpoint": srv.URL + "/authorize", "token_endpoint": srv.URL + "/token"}))
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) { writeTestJSON(t, w, tokenBody(ti)) })
	openURL := func(rawURL string) error {
		ti.captureNonce(rawURL)
		u, err := url.Parse(rawURL)
		if err != nil {
			return err
		}
		resp, err := http.Get(u.Query().Get("redirect_uri") + "?code=c&state=" + url.QueryEscape(u.Query().Get("state")))
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := LoginBrowserOAuth(ctx, llmprovider.ProviderGrok, OAuthFlowOptions{HTTPClient: srv.Client(), OpenURL: openURL,
		ClientID: "test-client", Issuer: srv.URL})
	return err
}

// TestLoginBrowserOAuth_RejectsUnverifiedIDToken (0016-MADR D7, A3): a login
// whose id_token fails verification, or is missing, fails; a valid one passes.
func TestLoginBrowserOAuth_RejectsUnverifiedIDToken(t *testing.T) {
	session := func(idToken string) string {
		return `{"access_token":"access","refresh_token":"refresh","expires_in":3600,"id_token":"` + idToken + `"}`
	}
	claims := func(ti *testIssuer, nonce string) map[string]any {
		c := map[string]any{"iss": ti.base, "aud": "test-client", "exp": time.Now().Add(time.Hour).Unix()}
		if nonce != "" {
			c["nonce"] = nonce
		}
		return c
	}
	sentNonce := func(ti *testIssuer) string {
		ti.mu.Lock()
		defer ti.mu.Unlock()
		return ti.nonce
	}
	for _, tc := range []struct {
		name    string
		body    func(ti *testIssuer) string
		wantErr bool
	}{
		{"valid", func(ti *testIssuer) string { return ti.tokenResponse(t, "ES256", "test-client", "access", "refresh") }, false},
		{"nonce mismatch", func(ti *testIssuer) string {
			return session(signTestJWT(t, "ES256", "ec-1", claims(ti, "not-the-sent-nonce")))
		}, true},
		{"tampered", func(ti *testIssuer) string {
			parts := strings.Split(signTestJWT(t, "ES256", "ec-1", claims(ti, sentNonce(ti))), ".")
			return session(parts[0] + "." + b64([]byte(`{"iss":"`+ti.base+`","aud":"test-client","exp":9999999999,"sub":"x"}`)) + "." + parts[2])
		}, true},
		{"alg none", func(ti *testIssuer) string { return session(signTestJWT(t, "none", "ec-1", claims(ti, sentNonce(ti)))) }, true},
		{"no id_token", func(*testIssuer) string {
			return `{"access_token":"access","refresh_token":"refresh","expires_in":3600}`
		}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := grokBrowserLogin(t, tc.body)
			if (err != nil) != tc.wantErr {
				t.Errorf("login error = %v, want error %v", err, tc.wantErr)
			}
		})
	}
}

// TestStartDeviceOAuth_CallerIssuerDiscoveryFailureFails (0016-MADR D7): a
// caller's issuer whose discovery returns 500 fails the login, rather than
// falling back to the built-in endpoints. The transport never reaches the
// network: discovery answers 500, and any other request, such as one to a
// fallback endpoint, a valid device code, so a fallback would succeed.
func TestStartDeviceOAuth_CallerIssuerDiscoveryFailureFails(t *testing.T) {
	var reached []string
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		reached = append(reached, r.URL.String())
		status, body := http.StatusOK, `{"device_code":"d","user_code":"ABCD-EFGH","verification_uri":"https://example.test/activate","expires_in":60,"interval":1}`
		if r.URL.Path == "/.well-known/openid-configuration" {
			status, body = http.StatusInternalServerError, "down"
		}
		return &http.Response{StatusCode: status, Status: http.StatusText(status), Header: http.Header{},
			Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})}
	if _, err := StartDeviceOAuth(context.Background(), llmprovider.ProviderGrok, OAuthFlowOptions{
		HTTPClient: client, ClientID: "test-client", Issuer: "https://issuer.example",
	}); err == nil {
		t.Fatalf("StartDeviceOAuth succeeded against an issuer whose discovery fails (requests: %v)", reached)
	}
}
