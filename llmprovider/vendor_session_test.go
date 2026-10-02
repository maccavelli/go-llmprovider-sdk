package llmprovider

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeVendorFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func codexAuthJSON(t *testing.T, exp time.Time, accountID string) (string, string) {
	t.Helper()
	access := openAITestJWT(t, map[string]any{"exp": exp.Unix()})
	return `{"tokens":{"access_token":"` + access + `","refresh_token":"rt-cli","account_id":"` + accountID + `"}}`, access
}

// TestVendorCLISession_ReadsThrough: each Token call re-reads the file, so a
// refresh the CLI writes is picked up with no request from this module.
func TestVendorCLISession_ReadsThrough(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	s := &VendorCLISession{Provider: ProviderOpenAI, Path: path}
	for _, account := range []string{"acct_1", "acct_2"} {
		content, access := codexAuthJSON(t, time.Now().Add(time.Hour), account)
		writeVendorFile(t, path, content)
		tok, err := s.Token(context.Background())
		if err != nil || tok.Value != access {
			t.Fatalf("Token = %q, %v; want the file's current token", tok.Value, err)
		}
		if got, _ := s.Account(); got != account {
			t.Fatalf("account = %q, want %q", got, account)
		}
	}
}

// TestVendorCLISession_ExpiredIsAuthFailure: an expired token is never
// refreshed here; the error says to run the CLI.
func TestVendorCLISession_ExpiredIsAuthFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	content, _ := codexAuthJSON(t, time.Now().Add(-time.Minute), "acct")
	writeVendorFile(t, path, content)
	_, err := (&VendorCLISession{Provider: ProviderOpenAI, Path: path}).Token(context.Background())
	if !errors.Is(err, ErrAuthFailure) || !strings.Contains(err.Error(), "run codex") {
		t.Fatalf("err = %v, want ErrAuthFailure advising to run codex", err)
	}
}

// TestVendorCLISession_GrokExactScope: the default login's scope is chosen
// every time, never another entry of the map (50 runs defeat map order).
func TestVendorCLISession_GrokExactScope(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	writeVendorFile(t, path, `{
  "xai::api_key": {"key": "xai-api-key", "auth_mode": "api_key"},
  "https://auth.x.ai::other-client": {"key": "grok-other", "auth_mode": "oidc", "expires_at": "2099-01-01T00:00:00Z"},
  "https://auth.x.ai::b1a00492-073a-47ea-816f-4c329264a828": {"key": "grok-default", "auth_mode": "oidc", "expires_at": "2099-01-01T00:00:00Z"}
}`)
	s := &VendorCLISession{Provider: ProviderGrok, Path: path}
	for range 50 {
		tok, err := s.Token(context.Background())
		if err != nil || tok.Value != "grok-default" {
			t.Fatalf("Token = %q, %v; want grok-default", tok.Value, err)
		}
	}
}
