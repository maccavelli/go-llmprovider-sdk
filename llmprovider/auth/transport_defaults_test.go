package auth

import (
	"net/http"
	"testing"
)

// TestUseHTTPClient_KeepsTheSessionsOwn was TestShareHTTPClient_KeepsTheSessionsOwn
// (0015-MADR amendment "what `auth` holds"): a session with no client takes
// the provider's; one that already has a client keeps it; a nil client
// changes nothing.
func TestUseHTTPClient_KeepsTheSessionsOwn(t *testing.T) {
	own, provider := &http.Client{}, &http.Client{}
	session := &OAuthSession{HTTPClient: own}
	session.UseHTTPClient(provider)
	if session.HTTPClient != own {
		t.Error("a session's own client was replaced")
	}
	bare := &OAuthSession{}
	bare.UseHTTPClient(nil)
	if bare.HTTPClient != nil {
		t.Error("a nil client was shared")
	}
	bare.UseHTTPClient(provider)
	if bare.HTTPClient != provider {
		t.Error("a session with no client did not take the provider's")
	}
}
