package llmprovider

import (
	"reflect"
	"testing"
)

// TestFillOAuthEndpoints: a field discovery left empty takes the built-in
// value, one it gave is kept, and the signing algorithms are filled only when
// discovery listed none (0016-MADR D7; 0015-PLAN S8, commit 2).
func TestFillOAuthEndpoints(t *testing.T) {
	builtin := oauthEndpoints{Authorization: "b-auth", Token: "b-token", Device: "b-device", JWKS: "b-jwks",
		SigningAlgs: []string{"RS256"}}
	discovered := oauthEndpoints{Token: "d-token", JWKS: "d-jwks"}
	fillOAuthEndpoints(&discovered, builtin)
	want := oauthEndpoints{Authorization: "b-auth", Token: "d-token", Device: "b-device", JWKS: "d-jwks",
		SigningAlgs: []string{"RS256"}}
	if !reflect.DeepEqual(discovered, want) {
		t.Errorf("filled = %+v, want %+v", discovered, want)
	}
	own := oauthEndpoints{SigningAlgs: []string{"ES256"}}
	fillOAuthEndpoints(&own, builtin)
	if !reflect.DeepEqual(own.SigningAlgs, []string{"ES256"}) {
		t.Errorf("SigningAlgs = %v, want discovery's [ES256] kept", own.SigningAlgs)
	}
}

// TestRefreshTokenURL: a session's own token URL wins; otherwise OpenAI's
// issuer refreshes at /oauth/token, and any other issuer at Grok's endpoint.
func TestRefreshTokenURL(t *testing.T) {
	for _, tc := range []struct {
		name  string
		state oauthSessionState
		want  string
	}{
		{"own token URL", oauthSessionState{tokenURL: "https://auth.example.test/token", issuer: DefaultOpenAIIssuer},
			"https://auth.example.test/token"},
		{"OpenAI", oauthSessionState{issuer: DefaultOpenAIIssuer + "/"}, DefaultOpenAIIssuer + "/oauth/token"},
		{"Grok", oauthSessionState{issuer: DefaultGrokOAuthIssuer}, defaultGrokOAuthRefreshURL},
	} {
		if got := refreshTokenURL(tc.state); got != tc.want {
			t.Errorf("%s: refreshTokenURL = %q, want %q", tc.name, got, tc.want)
		}
	}
}
