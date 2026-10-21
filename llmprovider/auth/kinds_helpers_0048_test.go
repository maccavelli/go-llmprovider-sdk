package auth

import (
	"context"
	"errors"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// kindSentinels0048 are llmprovider's error kinds. A plain error matches
// none of them (0048-MADR D4).
var kindSentinels0048 = []error{
	llmprovider.ErrRateLimited, llmprovider.ErrProviderUnavailable, llmprovider.ErrAuthFailure,
	llmprovider.ErrInvalidRequest, llmprovider.ErrQuotaExhausted, llmprovider.ErrNotPermitted,
	llmprovider.ErrContextOverflow, llmprovider.ErrIncomplete, llmprovider.ErrUnsupported,
	llmprovider.ErrInvalidProvider,
}

// assertPlain0048 fails unless err is non-nil, matches none of llmprovider's
// kinds, and names step, as the code's message does.
func assertPlain0048(t *testing.T, err error, step string) {
	t.Helper()
	if err == nil {
		t.Fatalf("err = nil, want a plain error naming %q", step)
	}
	for _, kind := range kindSentinels0048 {
		if errors.Is(err, kind) {
			t.Fatalf("err = %v matches %v; want a plain error", err, kind)
		}
	}
	if !strings.Contains(err.Error(), step) {
		t.Fatalf("err = %v, want it to name %q", err, step)
	}
}

// answer0048 is a handler that replies with status and body.
func answer0048(t *testing.T, status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if _, err := io.WriteString(w, body); err != nil {
			t.Errorf("write response: %v", err)
		}
	}
}

const (
	openAICode0048 = `{"device_auth_id":"dev-1","user_code":"WXYZ-1234","interval":"1"}`
	grokCode0048   = `{"device_code":"device-secret","user_code":"ABCD-EFGH","verification_uri":"https://example.test/activate","expires_in":60,"interval":1}`
	kiloCode0048   = `{"code":"KILO-1234","verificationUrl":"https://app.kilo.ai/device","expiresIn":60}`

	openAIPoll0048   = "/api/accounts/deviceauth/token"
	kiloCodes0048    = "POST /api/device-auth/codes"
	kiloPoll0048     = "GET /api/device-auth/codes/{code}"
	kiloPollPath0048 = "/api/device-auth/codes/KILO-1234"
)

// fake0048 is a fake issuer for one provider's logins.
type fake0048 struct {
	srv *httptest.Server
	ti  *testIssuer
}

// newFake0048 serves provider's discovery and device-code endpoints with the
// answers a login that succeeds gets, then routes, which replace any default
// of the same pattern. Grok's discovery names its authorize, device and token
// endpoints, less those drop names. Kilo has no discovery.
func newFake0048(t *testing.T, provider llmprovider.ProviderID, routes func(f *fake0048) map[string]http.HandlerFunc, drop ...string) *fake0048 {
	t.Helper()
	mux := http.NewServeMux()
	f := &fake0048{srv: httptest.NewServer(mux)}
	t.Cleanup(f.srv.Close)
	f.ti = newTestIssuer(mux, f.srv.URL)
	all := map[string]http.HandlerFunc{}
	switch provider {
	case llmprovider.ProviderOpenAI:
		all["/.well-known/openid-configuration"] = func(w http.ResponseWriter, _ *http.Request) {
			writeTestJSON(t, w, f.ti.discovery(t, nil))
		}
		all["/api/accounts/deviceauth/usercode"] = answer0048(t, http.StatusOK, openAICode0048)
	case llmprovider.ProviderGrok:
		fields := map[string]string{"authorization_endpoint": f.srv.URL + "/authorize",
			"device_authorization_endpoint": f.srv.URL + "/device", "token_endpoint": f.srv.URL + "/token"}
		for _, name := range drop {
			delete(fields, name)
		}
		all["/.well-known/openid-configuration"] = func(w http.ResponseWriter, _ *http.Request) {
			writeTestJSON(t, w, f.ti.discovery(t, fields))
		}
		all["/device"] = answer0048(t, http.StatusOK, grokCode0048)
	case llmprovider.ProviderKilo:
		all[kiloCodes0048] = answer0048(t, http.StatusOK, kiloCode0048)
	}
	if routes != nil {
		maps.Copy(all, routes(f))
	}
	for pattern, handler := range all {
		mux.HandleFunc(pattern, handler)
	}
	return f
}

// options are the flow options for this fake, with clock's time when set.
func (f *fake0048) options(clock *fakeOAuthClock) OAuthFlowOptions {
	opts := OAuthFlowOptions{HTTPClient: f.srv.Client(), ClientID: "test-client", Issuer: f.srv.URL}
	if clock != nil {
		opts.now, opts.sleep = clock.now, clock.sleep
	}
	return opts
}

// cancelOn0048 is a client that sends through f, except that a request for
// path first cancels the login's context, as a caller giving up while the
// request is in flight, and then fails with the request context's error.
func (f *fake0048) cancelOn0048(path string, cancel context.CancelFunc) *http.Client {
	base := f.srv.Client().Transport
	return &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == path {
			cancel()
			<-r.Context().Done()
			return nil, r.Context().Err()
		}
		return base.RoundTrip(r)
	})}
}

// visit0048 is an OpenURL that brings the browser back to the login's
// loopback callback with query and the login's state.
func visit0048(ti *testIssuer, query string) func(string) error {
	return func(rawURL string) error {
		ti.captureNonce(rawURL)
		u, err := url.Parse(rawURL)
		if err != nil {
			return err
		}
		q := u.Query()
		resp, err := http.Get(q.Get("redirect_uri") + "?" + query + "&state=" + url.QueryEscape(q.Get("state")))
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
		return err
	}
}
