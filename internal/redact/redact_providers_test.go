package redact

import (
	"strings"
	"testing"
)

// fake assembles a fake secret at runtime, so no contiguous literal trips a
// secret scanner.
func fake(prefix string) string { return prefix + "FAKE" + "abcdefghijklmnop0123" }

// TestRedact_ProviderKeysAndTokenFields (0020-MADR F8): the key formats of
// this module's own providers, every *_token field, the OAuth form fields,
// cookies, Kilo's URL-prefixed token, an unsigned JWT, escaped JSON and a
// short bearer are all redacted.
func TestRedact_ProviderKeysAndTokenFields(t *testing.T) {
	jwtNone := "eyJhbGciOiJub25lIn0" + ".eyJzdWIiOiIxIn0."
	cases := []struct {
		name, in, secret string
	}{
		{"openai key", "Incorrect API key " + fake("sk-") + " given", fake("sk-")},
		{"openai project key", "key " + fake("sk-proj-") + " given", fake("sk-proj-")},
		{"anthropic key", "key " + fake("sk-ant-api03-") + " given", fake("sk-ant-api03-")},
		{"openrouter key", "key " + fake("sk-or-v1-") + " given", fake("sk-or-v1-")},
		{"xai key", "key " + fake("xai-") + " given", fake("xai-")},
		{"together key", "key " + fake("tgp_v1_") + " given", fake("tgp_v1_")},
		{"hugging face token", "invalid credential " + fake("hf_") + " given", fake("hf_")},
		{"refresh_token json", `{"refresh_token":"` + fake("rt-") + `"}`, fake("rt-")},
		{"access_token json", `{"access_token":"` + fake("at-") + `"}`, fake("at-")},
		{"id_token json", `{"id_token":"` + fake("notajwt") + `"}`, fake("notajwt")},
		{"device_code json", `{"device_code":"` + fake("dc-") + `"}`, fake("dc-")},
		{"refresh_token form", "grant_type=refresh_token&refresh_token=" + fake("rt-") + "&client_id=app", fake("rt-")},
		{"code form", "code=" + fake("ac-") + "&state=s", fake("ac-")},
		{"code_verifier form", "code_verifier=" + fake("cv-") + "&x=1", fake("cv-")},
		{"key query", "GET /models?key=" + fake("k") + " failed", fake("k")},
		{"cookie header", "Cookie: session=" + fake("sess") + " end", fake("sess")},
		{"kilo url token", "token https://api.kilo.ai:" + fake("kilo") + " refused", fake("kilo")},
		{"unsigned jwt", "value " + jwtNone + " refused", jwtNone},
		{"escaped json", `{\"password\":\"` + fake("pw") + `\"}`, fake("pw")},
		{"short bearer", "Authorization: Bearer abc12 end", "abc12"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if out := String(tc.in); strings.Contains(out, tc.secret) || !strings.Contains(out, "[REDACTED]") {
				t.Errorf("String(%q) = %q; want %q redacted", tc.in, out, tc.secret)
			}
		})
	}
}

// TestRedact_KeepsDiagnostics (0020-MADR F8): short codes, ports and error
// names stay readable.
func TestRedact_KeepsDiagnostics(t *testing.T) {
	for _, keep := range []string{
		"error code: 1102",
		"status code=400",
		`{"error":"invalid_grant"}`,
		"http://localhost:11434/v1/chat/completions",
		"https://api.kilo.ai:443/api/gateway",
		"the key_map and token_v2 fields",
	} {
		if out := String(keep); out != keep {
			t.Errorf("String(%q) = %q; want it unchanged", keep, out)
		}
	}
}
