package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"
)

// plantedSecret is in every secret field below; no formatted form may show it.
const plantedSecret = "PLANTED-SECRET-7f3a"

// TestSessionRedacts is the session row of llmprovider's
// TestSecretBearingTypesRedact (0016-MADR D5), moved with the session
// (0015-PLAN S8c): a session, formatted by fmt, slog and encoding/json, hides
// its tokens and keeps its non-secret fields.
func TestSessionRedacts(t *testing.T) {
	expiry := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	session := &OAuthSession{Access: "at-" + plantedSecret, Refresh: "rt-" + plantedSecret,
		Expiry: expiry, Issuer: DefaultGrokOAuthIssuer, ClientID: "client-visible", AccountID: "acct-visible"}
	forms := map[string]string{}
	for _, verb := range []string{"%v", "%+v", "%#v", "%s"} {
		forms[verb] = fmt.Sprintf(verb, session)
	}
	var jsonOut, textOut bytes.Buffer
	slog.New(slog.NewJSONHandler(&jsonOut, nil)).Info("m", "v", session)
	slog.New(slog.NewTextHandler(&textOut, nil)).Info("m", "v", session)
	forms["slog JSON"], forms["slog text"] = jsonOut.String(), textOut.String()
	raw, err := json.Marshal(session)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	forms["json.Marshal"] = string(raw)
	for form, out := range forms {
		if strings.Contains(out, plantedSecret) {
			t.Errorf("*OAuthSession via %s shows the secret: %s", form, out)
		}
		if !strings.Contains(out, "client-visible") {
			t.Errorf("*OAuthSession via %s lost the non-secret client id: %s", form, out)
		}
	}
}

// TestSecretText is llmprovider's TestSecretText for auth's own copies of the
// helpers, with the nil-session forms that moved with the session.
func TestSecretText(t *testing.T) {
	for in, want := range map[string]string{"": "", "x": redactedSecret, strings.Repeat("s", 200): redactedSecret} {
		if got := secretText(in); got != want {
			t.Errorf("secretText(%d runes) = %q, want %q", len(in), got, want)
		}
	}
	if got := (*OAuthSession)(nil).String(); got != "OAuthSession(nil)" {
		t.Errorf("nil session String = %q", got)
	}
	if got := (*OAuthSession)(nil).LogValue().String(); got != "OAuthSession(nil)" {
		t.Errorf("nil session LogValue = %q", got)
	}
	if raw, err := json.Marshal((*OAuthSession)(nil)); err != nil || string(raw) != "null" {
		t.Errorf("nil session JSON = %s, %v; want null", raw, err)
	}
	if got := expiryText(time.Time{}); got != "none" {
		t.Errorf("expiryText(zero) = %q, want none", got)
	}
}
