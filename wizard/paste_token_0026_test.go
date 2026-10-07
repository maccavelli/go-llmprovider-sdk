package wizard

import (
	"encoding/base64"
	"testing"
)

// TestPasteAccessToken_KeepsAccount (0026-MADR F48): a pasted ChatGPT access
// token carrying the account claims saves AccountID and FedRAMP, as a login
// takes them from its id_token, so requests carry the account id.
func TestPasteAccessToken_KeepsAccount(t *testing.T) {
	jwt := func(payload string) string {
		enc := base64.RawURLEncoding.EncodeToString
		return enc([]byte(`{"alg":"RS256"}`)) + "." + enc([]byte(payload)) + ".c2lnbmF0dXJl"
	}
	for _, c := range []struct {
		name, payload, account string
		fedramp                bool
	}{
		{"nested claims", `{"exp":4102444800,"https://api.openai.com/auth":{"chatgpt_account_id":"acct-123",` +
			`"chatgpt_account_is_fedramp":true}}`, "acct-123", true},
		{"top-level account", `{"exp":4102444800,"chatgpt_account_id":"acct-456"}`, "acct-456", false},
		{"no claims", `{"exp":4102444800}`, "", false},
	} {
		s := accessOnlyOpenAISession(jwt(c.payload))
		if s.AccountID != c.account || s.FedRAMP != c.fedramp {
			t.Errorf("%s: AccountID %q, FedRAMP %t; want %q, %t", c.name, s.AccountID, s.FedRAMP, c.account, c.fedramp)
		}
	}
}
