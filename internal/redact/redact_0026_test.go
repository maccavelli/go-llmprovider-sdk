package redact

import (
	"strings"
	"testing"
)

// TestRedact_PrefixedKeys (0026-MADR F17): a secret name with a snake_case,
// kebab-case or camelCase prefix is still a secret name, so its value is
// redacted.
func TestRedact_PrefixedKeys(t *testing.T) {
	const secret = "s3cr3tValue9"
	for _, in := range []string{
		`openai_api_key=` + secret,
		`{"x_api_key": "` + secret + `"}`,
		`app_secret: ` + secret,
		`{"apiSecret":"` + secret + `"}`,
		`db_password=` + secret,
		`{"subscription_key": "` + secret + `"}`,
		`x-client-secret: ` + secret,
	} {
		if got := String(in); strings.Contains(got, secret) {
			t.Errorf("String(%q) = %q; the value is not redacted", in, got)
		}
	}
	// A count whose name holds "token" but does not end in it is not a secret.
	for _, in := range []string{`{"max_tokens": 128000}`, `prompt_token_count: 4096`} {
		if got := String(in); got != in {
			t.Errorf("String(%q) = %q; want it unchanged", in, got)
		}
	}
}

// TestRedact_KeepsOrdinaryWords (0026-MADR F21): a word after "bearer",
// "basic" or "token" is redacted only when it looks like a credential, so a
// service's diagnostic stays readable; after an Authorization label the value
// is always redacted, short or not.
func TestRedact_KeepsOrdinaryWords(t *testing.T) {
	for _, in := range []string{
		"Invalid bearer token",
		"Basic authentication is not supported",
		"The token provided is invalid",
		"bearer authorization required",
		"Missing bearer token.",
	} {
		if got := String(in); got != in {
			t.Errorf("String(%q) = %q; want it unchanged", in, got)
		}
	}
	for in, secret := range map[string]string{
		"Authorization: Bearer abc12 end":             "abc12",
		"Authorization: Basic dXNlcjpwYXNz":           "dXNlcjpwYXNz",
		"sent bearer sk_live_shortish9 to the server": "sk_live_shortish9",
		"retry with token Zm9vYmFyYmF6cXV4MTIz":       "Zm9vYmFyYmF6cXV4MTIz",
		"bearer abcdefghijklmnopqrstuvwxyzABCDEF":     "abcdefghijklmnopqrstuvwxyzABCDEF",
	} {
		if got := String(in); strings.Contains(got, secret) {
			t.Errorf("String(%q) = %q; the credential is not redacted", in, got)
		}
	}
}
