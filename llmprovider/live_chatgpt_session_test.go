//go:build live_gateways

package llmprovider_test

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// liveChatGPTSession borrows the Codex CLI's ChatGPT login read-only. It
// REQUIRES LLMPROVIDER_LIVE_CHATGPT=1 (every call spends the subscription) and
// $CODEX_HOME/auth.json (default ~/.codex). The session can never refresh:
// its refresh token is empty and its token URL unroutable, so the CLI's
// refresh token is never used or rotated (MADR 0012 §5). It skips when the
// access token has under ten minutes left.
func liveChatGPTSession(t *testing.T) *auth.OAuthSession {
	t.Helper()
	if os.Getenv("LLMPROVIDER_LIVE_CHATGPT") != "1" {
		t.Skip("LLMPROVIDER_LIVE_CHATGPT unset: live ChatGPT calls spend the subscription")
	}
	home := os.Getenv("CODEX_HOME")
	if home == "" {
		dir, err := os.UserHomeDir()
		if err != nil {
			t.Skip(err)
		}
		home = filepath.Join(dir, ".codex")
	}
	raw, err := os.ReadFile(filepath.Join(home, "auth.json"))
	if err != nil {
		t.Skipf("no Codex CLI login: %v", err)
	}
	var file struct {
		Tokens struct {
			Access    string `json:"access_token"`
			AccountID string `json:"account_id"`
		} `json:"tokens"`
	}
	if err := json.Unmarshal(raw, &file); err != nil || file.Tokens.Access == "" {
		t.Skipf("Codex CLI auth.json has no ChatGPT access token (%v)", err)
	}
	parts := strings.Split(file.Tokens.Access, ".")
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if len(parts) != 3 {
		t.Skip("access token is not a JWT")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || json.Unmarshal(payload, &claims) != nil {
		t.Skip("access token claims unreadable")
	}
	expiry := time.Unix(claims.Exp, 0)
	if time.Until(expiry) < 10*time.Minute {
		t.Skip("Codex CLI access token expires within ten minutes; run codex to refresh it")
	}
	return &auth.OAuthSession{
		Provider:  llmprovider.ProviderOpenAI,
		Access:    file.Tokens.Access,
		Expiry:    expiry,
		Issuer:    auth.DefaultOpenAIIssuer,
		ClientID:  auth.DefaultOpenAIClientID,
		AccountID: file.Tokens.AccountID,
		TokenURL:  "http://127.0.0.1:1/never-refresh",
	}
}
