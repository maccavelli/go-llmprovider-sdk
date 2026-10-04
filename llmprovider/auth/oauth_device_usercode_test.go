package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestOpenAIDevice_RefusesControlCharactersInUserCode (0020-MADR F47): the
// user_code is printed to the user's terminal, so an issuer cannot slip a
// terminal escape into it; Grok's and Kilo's codes are already checked.
func TestOpenAIDevice_RefusesControlCharactersInUserCode(t *testing.T) {
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	ti := newTestIssuer(mux, srv.URL)
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		writeTestJSON(t, w, ti.discovery(t, nil))
	})
	mux.HandleFunc("/api/accounts/deviceauth/usercode", func(w http.ResponseWriter, _ *http.Request) {
		writeTestJSON(t, w, `{"device_auth_id":"device-auth-id","user_code":"OPEN\u001b[2J-AI","interval":"2"}`)
	})
	notified := false
	// Unchecked, the login would show the code and poll; a deadline ends it.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := LoginDeviceOAuth(ctx, llmprovider.ProviderOpenAI, OAuthFlowOptions{
		HTTPClient:   srv.Client(),
		ClientID:     "test-client",
		Issuer:       srv.URL,
		NotifyDevice: func(string, string) { notified = true },
	})
	if err == nil || !strings.Contains(err.Error(), "user_code") {
		t.Errorf("LoginDeviceOAuth = %v, want an invalid user_code error", err)
	}
	if notified {
		t.Error("the user_code with a terminal escape was shown to the user")
	}
}
