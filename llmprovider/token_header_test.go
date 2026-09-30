package llmprovider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
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
			SetTokenHeader(req, tc.tok, tc.header, tc.scheme)
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
// TokenAPIKey and a session a TokenBearer; each names a Header only when its
// caller set one.
func TestTokenSources_ReportOnlyWhatIsSet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "refreshed", "expires_in": 3600})
	}))
	t.Cleanup(srv.Close)
	vendorPath := filepath.Join(t.TempDir(), "auth.json")
	content, _ := codexAuthJSON(t, time.Now().Add(time.Hour), "acct_1")
	writeVendorFile(t, vendorPath, content)

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
		{"oauth, current", &OAuthSession{Provider: ProviderOpenAI, Access: "a", Expiry: time.Now().Add(time.Hour)}, TokenBearer, ""},
		{"oauth, refreshed", &OAuthSession{Provider: ProviderOpenAI, Refresh: "r", ClientID: "client-test",
			TokenURL: srv.URL, HTTPClient: srv.Client()}, TokenBearer, ""},
		{"vendor cli", &VendorCLISession{Provider: ProviderOpenAI, Path: vendorPath}, TokenBearer, ""},
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
