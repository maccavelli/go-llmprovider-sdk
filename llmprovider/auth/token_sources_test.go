package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestSessions_ReportOnlyWhatIsSet is the session half of llmprovider's
// TestTokenSources_ReportOnlyWhatIsSet (0016-MADR A6), moved with the
// sessions (0015-PLAN S8c): a session's token is a TokenBearer that names no
// Header.
func TestSessions_ReportOnlyWhatIsSet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "refreshed", "expires_in": 3600})
	}))
	t.Cleanup(srv.Close)
	vendorPath := filepath.Join(t.TempDir(), "auth.json")
	content, _ := codexAuthJSON(t, time.Now().Add(time.Hour), "acct_1")
	writeVendorFile(t, vendorPath, content)

	for _, tc := range []struct {
		name string
		src  llmprovider.TokenSource
	}{
		{"oauth, current", &OAuthSession{Provider: llmprovider.ProviderOpenAI, Access: "a", Expiry: time.Now().Add(time.Hour)}},
		{"oauth, refreshed", &OAuthSession{Provider: llmprovider.ProviderOpenAI, Refresh: "r", ClientID: "client-test",
			TokenURL: srv.URL, HTTPClient: srv.Client()}},
		{"vendor cli", &VendorCLISession{Provider: llmprovider.ProviderOpenAI, Path: vendorPath}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tok, err := tc.src.Token(context.Background())
			if err != nil {
				t.Fatalf("Token: %v", err)
			}
			if tok.Type != llmprovider.TokenBearer || tok.Header != "" {
				t.Errorf("Type %q, Header %q; want %q, none", tok.Type, tok.Header, llmprovider.TokenBearer)
			}
		})
	}
}
