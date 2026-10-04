package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// serveCallback sends one request to the callback handler and returns its
// status.
func serveCallback(handler http.Handler, method, target string) int {
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(method, target, http.NoBody))
	return recorder.Code
}

// TestOAuthCallback_StrayRequestKeepsWaiting (0020-MADR F44, Q7 a; it
// replaces 0008-MADR D5's state-mismatch branch): a request with no state or
// the wrong state, from any page the user has open, gets 400 and does not end
// the login; the real callback still completes it.
func TestOAuthCallback_StrayRequestKeepsWaiting(t *testing.T) {
	t.Parallel()
	result := make(chan oauthCallbackResult, 1)
	handler := oauthCallbackHandler("/callback", "expected-state", result, "")
	for _, target := range []string{
		"http://127.0.0.1/callback",
		"http://127.0.0.1/callback?code=planted&state=wrong-state",
		"http://127.0.0.1/callback?error=access_denied",
	} {
		if status := serveCallback(handler, http.MethodGet, target); status != http.StatusBadRequest {
			t.Errorf("GET %s = %d, want 400", target, status)
		}
		select {
		case got := <-result:
			t.Fatalf("GET %s ended the login with %#v; a stray request must not", target, got)
		default:
		}
	}
	if status := serveCallback(handler, http.MethodGet, "http://127.0.0.1/callback?code=real&state=expected-state"); status != http.StatusOK {
		t.Fatalf("the real callback = %d, want 200", status)
	}
	select {
	case got := <-result:
		if got.err != nil || got.code != "real" {
			t.Fatalf("the real callback gave %#v, want code real", got)
		}
	default:
		t.Fatal("the real callback did not complete the login")
	}
}

// TestOAuthCallback_RefusesOtherMethods (0020-MADR F44): only GET, and
// OPTIONS for Grok's preflight, reach the callback.
func TestOAuthCallback_RefusesOtherMethods(t *testing.T) {
	t.Parallel()
	result := make(chan oauthCallbackResult, 1)
	handler := oauthCallbackHandler("/callback", "expected-state", result, "")
	for _, method := range []string{http.MethodDelete, http.MethodPost, http.MethodPut} {
		status := serveCallback(handler, method, "http://127.0.0.1/callback?code=abc&state=expected-state")
		if status != http.StatusMethodNotAllowed {
			t.Errorf("%s = %d, want 405", method, status)
		}
	}
	select {
	case got := <-result:
		t.Fatalf("a refused method ended the login with %#v", got)
	default:
	}
}

// TestParseOAuthInput_PastedURLNeedsState (0020-MADR F44): a pasted callback
// URL must carry the matching state; a bare code still skips the check.
func TestParseOAuthInput_PastedURLNeedsState(t *testing.T) {
	if code, err := parseOAuthInput("http://127.0.0.1:1455/auth/callback?code=attacker-code", "real-state"); err == nil ||
		!strings.Contains(err.Error(), "state mismatch") {
		t.Errorf("a pasted URL without state = %q, %v; want a state-mismatch error", code, err)
	}
	if code, err := parseOAuthInput("bare-code", "real-state"); err != nil || code != "bare-code" {
		t.Errorf("a bare code = %q, %v; want it accepted", code, err)
	}
}
