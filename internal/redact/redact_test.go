package redact

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"
)

// TestRedact_SecretClasses verifies each hardened pattern redacts its secret
// class while leaving the surrounding text intact.
func TestRedact_SecretClasses(t *testing.T) {
	// Assembled at runtime: a contiguous AIza-format literal in source trips
	// secret scanners (GitHub, gitleaks) even though the value is fake.
	googleKey := "AIza" + "SyA0123456789abcdefghijklmnopqrstuv"
	cases := []struct {
		name   string
		in     string
		secret string // substring that must be gone
	}{
		{"aws access key id", "key=AKIAIOSFODNN7EXAMPLE end", "AKIAIOSFODNN7EXAMPLE"},
		{"bare jwt", "tok eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.abcDEF123 end", "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.abcDEF123"},
		{"github token", "gh ghp_0123456789abcdefghijABCDEFGHIJ end", "ghp_0123456789abcdefghijABCDEFGHIJ"},
		{"github pat", "x github_pat_0123456789abcdefghij end", "github_pat_0123456789abcdefghij"},
		{"slack token", "s xoxb-2222-3333-abcdefghij end", "xoxb-2222-3333-abcdefghij"},
		{"stripe live", "p sk_live_0123456789abcdef0123 end", "sk_live_0123456789abcdef0123"},
		{"google api key", "g " + googleKey + " end", googleKey},
		{"pem header", "-----BEGIN RSA PRIVATE KEY----- rest", "BEGIN RSA PRIVATE KEY"},
		{"password kv", `cfg password=hunter2longvalue end`, "hunter2longvalue"},
		{"api_key json", `{"api_key":"abcd1234efgh"}`, "abcd1234efgh"},
		{"legacy token_", "connecting with token_abc123xyz end", "token_abc123xyz"},
		{"legacy short secret_", "had secret_xyz456 here", "secret_xyz456"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := String(tc.in)
			if strings.Contains(out, tc.secret) {
				t.Errorf("secret %q not redacted: %q", tc.secret, out)
			}
			if !strings.Contains(out, "[REDACTED]") {
				t.Errorf("expected [REDACTED] marker, got %q", out)
			}
		})
	}
}

// TestRedact_Refinements covers the follow-up additions: DSN credentials, and
// confirms schemeless Authorization / X-Api-Key are caught by the key=value pass.
func TestRedact_Refinements(t *testing.T) {
	cases := []struct {
		name   string
		in     string
		secret string
		keep   string // substring that must remain (readability)
	}{
		{"postgres dsn", "dsn postgres://admin:s3cr3tpw@db.host:5432/app end", "s3cr3tpw", "postgres://"},
		{"redis dsn", "redis://:hunter2pass@cache:6379 ok", "hunter2pass", "redis://"},
		{"schemeless authorization", "Authorization: ab39f0c2deadbeef0011 done", "ab39f0c2deadbeef0011", "Authorization"},
		{"x-api-key header", `X-Api-Key: abcd1234efghijk done`, "abcd1234efghijk", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := String(tc.in)
			if strings.Contains(out, tc.secret) {
				t.Errorf("secret %q not redacted: %q", tc.secret, out)
			}
			if !strings.Contains(out, "[REDACTED]") {
				t.Errorf("expected [REDACTED], got %q", out)
			}
			if tc.keep != "" && !strings.Contains(out, tc.keep) {
				t.Errorf("expected %q preserved, got %q", tc.keep, out)
			}
		})
	}
}

// TestRedact_SizeCap confirms an oversized input is truncated with a marker
// (bounding regex work + stored size) and the caller's buffer is not mutated.
func TestRedact_SizeCap(t *testing.T) {
	in := make([]byte, maxRedactBytes+5000)
	for i := range in {
		in[i] = 'a'
	}
	orig := append([]byte(nil), in...)
	out := Redact(in)
	if len(out) >= len(in) {
		t.Errorf("expected truncation, got len %d (input %d)", len(out), len(in))
	}
	if !strings.Contains(string(out), "truncated 5000 bytes") {
		t.Errorf("expected truncation marker, got tail %q", string(out[len(out)-40:]))
	}
	if string(in) != string(orig) {
		t.Error("Redact must not mutate the caller's buffer")
	}
}

// TestRedact_NoOverRedaction confirms the word-anchored, min-length legacy prefix
// no longer clobbers short benign identifiers.
func TestRedact_NoOverRedaction(t *testing.T) {
	for _, benign := range []string{"secret_id", "token_v2", "next_secret_in_line", "key_map"} {
		if out := String("value " + benign + " here"); !strings.Contains(out, benign) {
			t.Errorf("benign identifier %q was over-redacted: %q", benign, out)
		}
	}
}

// TestRedact_BearerLeak is the B2 regression: the JWT after "Bearer" used to be
// emitted in cleartext because leftmost-match consumed only "Bearer".
func TestRedact_BearerLeak(t *testing.T) {
	in := "Authorization: Bearer eyJhbGciOi.eyJzdWIi.sigABC123 done"
	out := String(in)
	for _, leak := range []string{"eyJhbGciOi", "eyJzdWIi", "sigABC123"} {
		if strings.Contains(out, leak) {
			t.Errorf("JWT segment %q leaked: %q", leak, out)
		}
	}
	if !strings.Contains(out, "[REDACTED]") {
		t.Errorf("expected redaction, got %q", out)
	}
	// The "Authorization:" label is preserved for readability.
	if !strings.Contains(out, "Authorization:") {
		t.Errorf("expected Authorization label preserved, got %q", out)
	}
}

// TestRedact_PreservesBenign verifies clean text is returned byte-for-byte
// (same backing array) so logs stay readable and the io.Writer contract holds.
func TestRedact_PreservesBenign(t *testing.T) {
	in := []byte("connecting with the database and finishing up")
	out := Redact(in)
	if string(out) != string(in) {
		t.Errorf("benign text altered: %q", out)
	}
	if &out[0] != &in[0] {
		t.Error("benign input should be returned unmodified (same backing array)")
	}
}

// cleanText is n bytes of log-like prose with no secret and none of the
// words a secret pattern anchors on.
func cleanText(n int) []byte {
	const line = "upstream request failed while processing the completion; the model returned an unexpected response and the gateway will retry later. "
	return []byte(strings.Repeat(line, n/len(line)+1)[:n])
}

// BenchmarkRedact_Clean4KiB (0021-MADR Z1): the cost of redacting a clean
// 4 KiB message, the common case.
func BenchmarkRedact_Clean4KiB(b *testing.B) {
	in := cleanText(4 << 10)
	b.ReportAllocs()
	for b.Loop() {
		_ = Redact(in)
	}
}

// pemBlock is a fake PEM private key; its body line is what must go.
const pemBody = "MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC7fakefakefake"

func pemBlock(body string) string {
	return "-----BEGIN PRIVATE KEY-----\n" + body + "\n" + body + "\n-----END PRIVATE KEY-----"
}

// TestRedact_Coverage0021 (0021-MADR Z2): the secret shapes the v1.1 review
// found unredacted are redacted, and the diagnostics it found over-redacted
// are kept.
func TestRedact_Coverage0021(t *testing.T) {
	for _, c := range []struct {
		name, in string
		secrets  []string
	}{
		{"accessToken", `{"accessToken":"abcd1234efgh5678"}`, []string{"abcd1234efgh5678"}},
		{"refreshToken", `{"refreshToken":"abcd1234efgh5678"}`, []string{"abcd1234efgh5678"}},
		{"idToken", `{"idToken":"abcd1234efgh5678"}`, []string{"abcd1234efgh5678"}},
		{"sessionToken", `{"sessionToken":"abcd1234efgh5678"}`, []string{"abcd1234efgh5678"}},
		{"pem block", "key " + pemBlock(pemBody) + " end", []string{pemBody, "-----END"}},
		{"cookie pairs", "Cookie: a=secret1234; b=secret5678", []string{"secret1234", "secret5678"}},
		{"quoted password", `password="two words here"`, []string{"two", "words", "here"}},
		{"refresh grant form", "grant_type=refresh_token&refresh_token=abc%2Fdef12345", []string{"abc%2Fdef12345"}},
		{"signature", "signature=abcdef123456", []string{"abcdef123456"}},
		{"private_key", `{"private_key":"` + pemBody + `"}`, []string{pemBody}},
		// Assembled at runtime, as TestRedact_SecretClasses's key is.
		{"google key ending in -", "key " + "AIza" + strings.Repeat("Ab0", 11) + "Cd-" + " given", []string{"Ab0Ab0Ab0Ab0Ab0Ab0Ab0Ab0Ab0Ab0Ab0Cd-"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			out := String(c.in)
			for _, secret := range c.secrets {
				if strings.Contains(out, secret) {
					t.Errorf("String(%q) = %q; %q is not redacted", c.in, out, secret)
				}
			}
		})
	}
	for _, keep := range []string{
		`{"code":"context_length_exceeded"}`,
		`'code': 'model_not_found'`,
		"token: 128000",
	} {
		if out := String(keep); out != keep {
			t.Errorf("String(%q) = %q; a diagnostic must be kept", keep, out)
		}
	}
}

// plantedRand is the seeded source of TestRedact_PlantedSecrets.
type plantedRand struct{ *rand.Rand }

// word is n random characters from alphabet.
func (r plantedRand) word(n int, alphabet string) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = alphabet[r.IntN(len(alphabet))]
	}
	return string(b)
}

const (
	alnum     = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	base64url = alnum + "-_"
)

// value is a random secret value: never a number, never a snake_case word.
func (r plantedRand) value() string { return "v" + r.word(12+r.IntN(20), alnum) + "Z9" }

// TestRedact_PlantedSecrets (0021-MADR Z2): 1,000 seeded cases, each planting
// one secret of every class in a random JSON, form, header or prose context.
// No secret survives.
func TestRedact_PlantedSecrets(t *testing.T) {
	r := plantedRand{rand.New(rand.NewPCG(1, 2))}
	googleKey := func() string { return "AIza" + r.word(35, base64url) }
	standalone := map[string]func() string{
		"openai":    func() string { return "sk-" + r.word(24, base64url) },
		"anthropic": func() string { return "sk-ant-api03-" + r.word(24, base64url) },
		"xai":       func() string { return "xai-" + r.word(24, base64url) },
		"together":  func() string { return "tgp_v1_" + r.word(24, base64url) },
		"hf":        func() string { return "hf_" + r.word(24, alnum) },
		"google":    googleKey,
		"github":    func() string { return "ghp_" + r.word(30, alnum) },
		"jwt": func() string {
			return "eyJ" + r.word(16, base64url) + "." + r.word(20, base64url) + "." + r.word(20, base64url)
		},
	}
	keyed := []string{"password", "secret", "api_key", "apiKey", "client_secret", "clientSecret", "access_token",
		"refresh_token", "accessToken", "refreshToken", "idToken", "sessionToken", "signature", "private_key", "privateKey"}
	contexts := []func(key, value string) string{
		func(key, value string) string { return fmt.Sprintf(`{"model":"m","%s":"%s","n":3}`, key, value) },
		func(key, value string) string { return fmt.Sprintf("a=1&%s=%s&b=2", key, value) },
		func(key, value string) string { return fmt.Sprintf("X-Trace: 1\r\n%s: %s\r\n", key, value) },
		func(key, value string) string { return fmt.Sprintf("the request had %s=%s and failed", key, value) },
	}
	note := []func(secret string) string{
		func(secret string) string { return fmt.Sprintf(`{"message":"bad credential %s given"}`, secret) },
		func(secret string) string { return "a=1&note=" + secret + "&b=2" },
		func(secret string) string { return "X-Note: " + secret + "\r\n" },
		func(secret string) string { return "the key " + secret + " was refused" },
	}
	for i := range 1000 {
		var b strings.Builder
		var secrets []string
		for _, class := range []string{"openai", "anthropic", "xai", "together", "hf", "google", "github", "jwt"} {
			secret := standalone[class]()
			secrets = append(secrets, secret)
			b.WriteString(note[r.IntN(len(note))](secret) + "\n")
		}
		for _, key := range keyed {
			secret := r.value()
			secrets = append(secrets, secret)
			b.WriteString(contexts[r.IntN(len(contexts))](key, secret) + "\n")
		}
		bearer, c1, c2 := r.value(), r.value(), r.value()
		pem := r.word(48, alnum)
		secrets = append(secrets, bearer, c1, c2, pem)
		b.WriteString("Authorization: Bearer " + bearer + "\n")
		b.WriteString("Cookie: s=" + c1 + "; t=" + c2 + "\n")
		b.WriteString(pemBlock(pem) + "\n")
		out := String(b.String())
		for _, secret := range secrets {
			if strings.Contains(out, secret) {
				t.Fatalf("case %d: %q survived in\n%s", i, secret, out)
			}
		}
	}
}
