package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// Grok's sign-ins name this module in referrer, as the CLI names itself
// (0002-MADR §5; 0002-PLAN Phase 8 step 0). Both are offline: the live login
// tests check the same value against the real issuer.

const wantReferrer = "go-llmprovider-sdk"

// TestGrokAuthorizeURL_CarriesTheReferrer: the browser login's authorize URL.
func TestGrokAuthorizeURL_CarriesTheReferrer(t *testing.T) {
	t.Parallel()
	config := oauthFlowConfig{provider: llmprovider.ProviderGrok, clientID: DefaultGrokOAuthClientID}
	rawURL, err := buildAuthorizeURL(config, DefaultGrokOAuthIssuer+"/oauth2/auth",
		"http://127.0.0.1:56121/callback", "challenge", "state", "nonce")
	if err != nil {
		t.Fatalf("buildAuthorizeURL: %v", err)
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	if got := u.Query()["referrer"]; len(got) != 1 || got[0] != wantReferrer {
		t.Errorf("referrer = %q, want [%q]", got, wantReferrer)
	}
}

// TestGrokDeviceRequest_CarriesTheReferrer: the device-code request's form.
func TestGrokDeviceRequest_CarriesTheReferrer(t *testing.T) {
	var mu sync.Mutex
	var form url.Values
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	ti := newTestIssuer(mux, srv.URL)
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		writeTestJSON(t, w, ti.discovery(t, map[string]string{"device_authorization_endpoint": srv.URL + "/device", "token_endpoint": srv.URL + "/token"}))
	})
	mux.HandleFunc("/device", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse device form: %v", err)
		}
		mu.Lock()
		form = r.PostForm
		mu.Unlock()
		writeTestJSON(t, w, `{"device_code":"device-secret","user_code":"ABCD-EFGH","verification_uri":"https://example.test/activate","expires_in":60,"interval":1}`)
	})
	login, err := StartDeviceOAuth(context.Background(), llmprovider.ProviderGrok, grokDeviceOptions(srv, nil))
	if err != nil {
		t.Fatalf("StartDeviceOAuth: %v", err)
	}
	login.Cancel()
	mu.Lock()
	defer mu.Unlock()
	if got := form["referrer"]; len(got) != 1 || got[0] != wantReferrer {
		t.Errorf("device form referrer = %q, want [%q]", got, wantReferrer)
	}
}
