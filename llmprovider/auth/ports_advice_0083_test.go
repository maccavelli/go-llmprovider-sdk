package auth

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestPortsAdvicePerMethod_0083 (ST27, ST28, ST34): with both registered ports
// held, the plan's error keeps the prefix, says to free the two ports it was
// given and to try again, and names no device-code login, which it does not
// have; the browser sign-in's keeps the prefix and says to free the two ports
// or use the device-code login. CI runs it on each platform.
func TestPortsAdvicePerMethod_0083(t *testing.T) {
	const prefix = "oauth: registered callback ports unavailable; "
	t.Run("the plan", func(t *testing.T) {
		pl := newPlanLogin(t)
		ports := holdPorts0074(t)
		_, err := LoginChatGPTPlan(ctx0083(t), pl.options())
		want := fmt.Sprintf("free ports %d and %d and try again", ports[0], ports[1])
		if err == nil || !strings.HasPrefix(err.Error(), prefix+want) || strings.Contains(err.Error(), "device-code") {
			t.Errorf("LoginChatGPTPlan error = %v; want %q%q and no device-code login", err, prefix, want)
		}
	})
	t.Run("the browser sign-in", func(t *testing.T) {
		mux := http.NewServeMux()
		srv := httptest.NewServer(mux)
		t.Cleanup(srv.Close)
		ti := newTestIssuer(mux, srv.URL)
		mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
			writeTestJSON(t, w, ti.discovery(t, nil))
		})
		ports := holdPorts0074(t)
		_, err := LoginBrowserOAuth(ctx0083(t), llmprovider.ProviderOpenAI,
			OAuthFlowOptions{HTTPClient: srv.Client(), ClientID: "test-client", Issuer: srv.URL})
		want := fmt.Sprintf("free ports %d and %d, or use device-code login", ports[0], ports[1])
		if err == nil || !strings.HasPrefix(err.Error(), prefix+want) {
			t.Errorf("LoginBrowserOAuth error = %v; want %q%q", err, prefix, want)
		}
	})
}
