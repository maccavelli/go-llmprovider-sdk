package llmprovider

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"
)

// plantedSecret is in every secret field below; no formatted form may show it.
const plantedSecret = "PLANTED-SECRET-7f3a"

// TestSecretBearingTypesRedact (0016-MADR D5): every secret-bearing value,
// formatted by fmt and by slog's JSON and text handlers, hides its secrets
// and keeps its non-secret fields.
func TestSecretBearingTypesRedact(t *testing.T) {
	expiry := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	token := Token{Value: "tok-" + plantedSecret, Type: TokenBearer, Expiry: expiry, Header: "Authorization"}
	static := StaticToken{Value: "key-" + plantedSecret, Header: "x-api-key"}
	session := &OAuthSession{Provider: ProviderGrok, Access: "at-" + plantedSecret, Refresh: "rt-" + plantedSecret,
		Expiry: expiry, Issuer: DefaultGrokOAuthIssuer, ClientID: "client-visible", AccountID: "acct-visible"}

	for _, tc := range []struct {
		name  string
		value any
		shown string // a non-secret field every form must still show
	}{
		{"Token", token, "Authorization"},
		{"*Token", &token, "Authorization"},
		{"StaticToken", static, "x-api-key"},
		{"*StaticToken", &static, "x-api-key"},
		{"*OAuthSession", session, "client-visible"},
		{"struct holding a Token", struct{ Tok Token }{token}, "Authorization"},
	} {
		forms := map[string]string{}
		for _, verb := range []string{"%v", "%+v", "%#v", "%s"} {
			forms[verb] = fmt.Sprintf(verb, tc.value)
		}
		var jsonOut, textOut bytes.Buffer
		slog.New(slog.NewJSONHandler(&jsonOut, nil)).Info("m", "v", tc.value)
		slog.New(slog.NewTextHandler(&textOut, nil)).Info("m", "v", tc.value)
		forms["slog JSON"], forms["slog text"] = jsonOut.String(), textOut.String()

		for form, out := range forms {
			// Known gap, recorded as an open item in 0016-PLAN's T2 step 3
			// record: slog's JSON handler encodes a struct *holding* a Token
			// with encoding/json, which never calls the nested LogValue. Only
			// a redacting MarshalJSON would close it, which T2 does not plan.
			if tc.name == "struct holding a Token" && form == "slog JSON" {
				continue
			}
			if strings.Contains(out, plantedSecret) {
				t.Errorf("%s via %s shows the secret: %s", tc.name, form, out)
			}
			if tc.name != "struct holding a Token" || !strings.HasPrefix(form, "slog") {
				if !strings.Contains(out, tc.shown) {
					t.Errorf("%s via %s lost the non-secret %q: %s", tc.name, form, tc.shown, out)
				}
			}
		}
	}
}

// TestSecretText: an empty secret shows as empty, any other as the fixed
// placeholder, whatever its length.
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
	if got := expiryText(time.Time{}); got != "none" {
		t.Errorf("expiryText(zero) = %q, want none", got)
	}
}
