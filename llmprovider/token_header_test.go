package llmprovider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestTokenHeader is R16's rule (0016-MADR D2, A6): no Header keeps the
// service's header and scheme; a Header replaces both, with "Bearer" for a
// TokenBearer and nothing for any other Type.
func TestTokenHeader(t *testing.T) {
	for _, tc := range []struct {
		name           string
		tok            Token
		header, scheme string
		wantName       string
		wantValue      string
	}{
		{"own header and scheme", Token{Value: "v", Type: TokenBearer}, "Authorization", "Bearer", "Authorization", "Bearer v"},
		{"own header, no scheme", Token{Value: "v", Type: TokenBearer}, "x-api-key", "", "x-api-key", "v"},
		{"override, bearer", Token{Value: "v", Type: TokenBearer, Header: "X-Custom"}, "x-api-key", "", "X-Custom", "Bearer v"},
		{"override, api key", Token{Value: "v", Type: TokenAPIKey, Header: "X-Custom"}, "Authorization", "Bearer", "X-Custom", "v"},
		{"override, no type", Token{Value: "v", Header: "X-Custom"}, "Authorization", "Bearer", "X-Custom", "v"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://example.invalid/", http.NoBody)
			tc.tok.Apply(req, tc.header, tc.scheme)
			if got := req.Header.Get(tc.wantName); got != tc.wantValue {
				t.Errorf("%s = %q, want %q", tc.wantName, got, tc.wantValue)
			}
			if len(req.Header) != 1 {
				t.Errorf("headers %v, want only %s", req.Header, tc.wantName)
			}
		})
	}
}

// TestTokenSources_ReportOnlyWhatIsSet (0016-MADR A6): a key source is a
// TokenAPIKey that names a Header only when its caller set one. The session
// rows moved to auth's TestSessions_ReportOnlyWhatIsSet (0015-PLAN S8c).
func TestTokenSources_ReportOnlyWhatIsSet(t *testing.T) {
	for _, tc := range []struct {
		name       string
		src        TokenSource
		wantType   TokenType
		wantHeader string
	}{
		{"static key", NewStaticToken("k"), TokenAPIKey, ""},
		{"static key, header set", &StaticToken{Value: "k", Header: "X-Custom"}, TokenAPIKey, "X-Custom"},
		{"command", NewCommandToken(helperArgv(t, "print", "k")...), TokenAPIKey, ""},
		{"command, header set", &CommandToken{Argv: helperArgv(t, "print", "k"), Header: "X-Custom"}, TokenAPIKey, "X-Custom"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tok, err := tc.src.Token(context.Background())
			if err != nil {
				t.Fatalf("Token: %v", err)
			}
			if tok.Type != tc.wantType || tok.Header != tc.wantHeader {
				t.Errorf("Type %q, Header %q; want %q, %q", tok.Type, tok.Header, tc.wantType, tc.wantHeader)
			}
		})
	}
}
