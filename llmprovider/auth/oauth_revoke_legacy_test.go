package auth

import (
	"context"
	"net/http"
	"net/url"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestRevokeOAuthSession_LegacyGrokSession: an older Grok session with no
// issuer refreshes against xAI's issuer (refreshTokenURL), so it revokes
// there too, rather than running discovery on an empty issuer (0026-MADR
// F45). The client sends every request to the test issuer, and records the
// host each was meant for.
func TestRevokeOAuthSession_LegacyGrokSession(t *testing.T) {
	srv, c := revokeIssuer(t, true)
	target, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var hosts []string
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		mu.Lock()
		hosts = append(hosts, r.URL.Host)
		mu.Unlock()
		r = r.Clone(r.Context())
		r.URL.Scheme, r.URL.Host = target.Scheme, target.Host
		return srv.Client().Transport.RoundTrip(r)
	})}
	err = RevokeOAuthSession(context.Background(), &OAuthSession{Provider: llmprovider.ProviderGrok, Access: "a", Refresh: "rt",
		ClientID: "grok-client", HTTPClient: client})
	wantHost, _ := url.Parse(DefaultGrokOAuthIssuer)
	if err != nil || len(hosts) == 0 || hosts[0] != wantHost.Host || c.path != "/oauth2/revoke" || c.form.Get("token") != "rt" {
		t.Fatalf("err = %v, hosts %v, revoke %s %v; want discovery at %s, then the form at /oauth2/revoke",
			err, hosts, c.path, c.form, wantHost.Host)
	}
}
