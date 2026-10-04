package auth

import (
	"context"
	"net/http"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestLogin_SessionKeepsOnlyTheCallersClient (0020-MADR F20): a login keeps
// the client its caller gave. Without one it uses the default, but the
// session it returns has none, so the provider's UseHTTPClient gives it the
// provider's (0016-MADR D8).
func TestLogin_SessionKeepsOnlyTheCallersClient(t *testing.T) {
	for _, own := range []*http.Client{nil, {}} {
		config, err := resolveOAuthFlowConfig(llmprovider.ProviderOpenAI, OAuthFlowOptions{HTTPClient: own})
		if err != nil {
			t.Fatal(err)
		}
		session, err := oauthSessionFromResponse(config, "https://issuer.example/oauth/token", oauthTokenResponse{AccessToken: "access"})
		if err != nil {
			t.Fatal(err)
		}
		if session.HTTPClient != own {
			t.Errorf("caller's client %p: the session holds %p", own, session.HTTPClient)
		}
	}

	srv, _ := kiloDeviceServer(t, http.StatusOK, http.StatusOK)
	opts := kiloOptions(srv, newFakeOAuthClock())
	opts.HTTPClient = nil
	login, err := StartDeviceOAuth(context.Background(), llmprovider.ProviderKilo, opts)
	if err != nil {
		t.Fatal(err)
	}
	session, err := login.Wait(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	provider := &http.Client{}
	session.UseHTTPClient(provider)
	if session.HTTPClient != provider {
		t.Error("a Kilo login without a client: the session kept the default, not the provider's")
	}
}
