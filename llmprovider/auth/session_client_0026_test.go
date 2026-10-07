package auth

import (
	"net/http"
	"testing"
)

// TestOAuthSession_CallersClientReplacesDefault (0026-MADR F37, the PLAN's
// deviation D8): a provider's default client is the session's only until a
// caller's client arrives; a caller's client, given through a provider or
// set on the field, is never replaced.
func TestOAuthSession_CallersClientReplacesDefault(t *testing.T) {
	providerDefault, laterDefault := &http.Client{}, &http.Client{}
	callers, laterCallers := &http.Client{}, &http.Client{}

	s := &OAuthSession{}
	s.UseDefaultHTTPClient(providerDefault)
	s.UseDefaultHTTPClient(laterDefault)
	if s.HTTPClient != providerDefault {
		t.Fatal("a second default replaced the first")
	}
	s.UseHTTPClient(callers)
	if s.HTTPClient != callers {
		t.Fatal("a caller's client did not replace the provider's default")
	}
	s.UseDefaultHTTPClient(laterDefault)
	s.UseHTTPClient(laterCallers)
	if s.HTTPClient != callers {
		t.Fatal("a default or a later caller's client replaced the caller's")
	}

	own := &http.Client{}
	set := &OAuthSession{HTTPClient: own}
	set.UseHTTPClient(callers)
	set.UseDefaultHTTPClient(providerDefault)
	if set.HTTPClient != own {
		t.Fatal("a client set on the session's field was replaced")
	}
}
