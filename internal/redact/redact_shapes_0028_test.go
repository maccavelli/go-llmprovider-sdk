package redact

import (
	"strings"
	"testing"
)

// synthetic builds a value of n characters from charset, cycling through it:
// the shape of a credential, never one.
func synthetic(charset string, n int) string {
	var b strings.Builder
	for i := range n {
		b.WriteByte(charset[(i*7+3)%len(charset)])
	}
	return b.String()
}

// alnum and base64url are redact_test.go's.
const (
	lowerHex = "0123456789abcdef"
	letters  = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

// TestRedact_KeepsDiagnosticCodes (0028-MADR A10b): a code made of
// letter-only words joined by - or _ is a diagnostic, as Grok's and Kilo's
// refusals show (0028-PLAN Phase 1, T1), and is kept.
func TestRedact_KeepsDiagnosticCodes(t *testing.T) {
	for _, s := range []string{
		`{"code":"invalid-argument","error":"Incorrect API key provided."}`,
		`{"error":{"code":"INVALID_TOKEN","message":"Your authentication token is invalid."}}`,
		`{"code":"model-not-found"}`,
	} {
		if got := String(s); got != s {
			t.Errorf("String(%q) = %q; want the diagnostic code kept", s, got)
		}
	}
}

// TestRedact_StillMasksSecretShapedCodes: a code or key value with a digit, a
// longer one, and any bare key's value that is not today's diagnostic, stay
// masked.
func TestRedact_StillMasksSecretShapedCodes(t *testing.T) {
	for _, v := range []string{
		synthetic(lowerHex, 32),
		"sk-proj-" + synthetic(alnum, 40),
		synthetic(base64url, 43),
		"abcd-1234-efgh-5678",
		synthetic(letters, 20) + "-" + synthetic(letters, 20),
	} {
		s := `{"code":"` + v + `"}`
		if strings.Contains(String(s), v) {
			t.Errorf("code %q was kept; want it masked", v)
		}
	}
	// A bare key keeps today's rule: letter words with a hyphen are masked.
	if s := `{"key":"abc-DEF-ghij"}`; strings.Contains(String(s), "abc-DEF-ghij") {
		t.Errorf("String(%q) kept the key; want it masked", s)
	}
}

// keyShapes are synthetic values in each researched shape (0028-PLAN D3).
func keyShapes() map[string]string {
	return map[string]string{
		"OpenAI legacy":        "sk-" + synthetic(alnum, 20) + "T3BlbkFJ" + synthetic(alnum, 20),
		"OpenAI project":       "sk-proj-" + synthetic(base64url, 74) + "T3BlbkFJ" + synthetic(base64url, 74),
		"Anthropic":            "sk-ant-api03-" + synthetic(base64url, 93) + "AA",
		"Anthropic OAuth":      "sk-ant-oat01-" + synthetic(base64url, 60),
		"OpenRouter":           "sk-or-v1-" + synthetic(lowerHex, 64),
		"OpenCode":             "sk-" + synthetic(alnum, 64),
		"Google API key":       "AIza" + synthetic(base64url, 35),
		"Gemini auth key":      "AQ.Ab" + synthetic(base64url, 50),
		"Google access token":  "ya29." + synthetic(base64url, 60),
		"Google refresh token": "1//0" + synthetic(base64url, 60),
		"xAI":                  "xai-" + synthetic(alnum, 80),
		"Together":             "tgp_v1_" + synthetic(base64url, 43),
		"Hugging Face":         "hf_" + synthetic(letters, 34),
		"Hugging Face org":     "api_org_" + synthetic(letters, 34),
	}
}

// TestRedact_KeyShapes (0028-MADR A10c): each researched shape is masked
// whole, in a sentence and in JSON; Ollama's key where the text names Ollama.
func TestRedact_KeyShapes(t *testing.T) {
	for name, v := range keyShapes() {
		for _, s := range []string{"the credential " + v + " was refused", `{"value":"` + v + `"}`} {
			if got := String(s); strings.Contains(got, v[len(v)-12:]) {
				t.Errorf("%s: String(%q) = %q; want it masked", name, s, got)
			}
		}
	}
	ollama := synthetic(lowerHex, 32) + "." + synthetic(base64url, 24)
	if got := String("ollama key " + ollama + " refused"); strings.Contains(got, ollama[len(ollama)-12:]) {
		t.Errorf("Ollama: %q; want it masked", got)
	}
}

// TestRedact_NotKeys: ordinary text that today's redaction masked, and text
// that resembles a shape, is kept. "(id)hf_requiredCharacteristicTypes…" is
// not here: it has the shape a Hugging Face token could take, and stays
// masked, a known false positive (0028-PLAN D7).
func TestRedact_NotKeys(t *testing.T) {
	for _, s := range []string{
		"see sk-learn-tutorial-for-beginners for details",
		"crate xai-grok-login-device-code-flow-tests failed",
		"the FAQ.md file",
		"x = a 1//long_python_variable_name_over_forty_characters",
		"hash " + synthetic(lowerHex, 32) + "." + synthetic(letters, 24) + " here",
	} {
		if got := String(s); got != s {
			t.Errorf("String(%q) = %q; want it kept", s, got)
		}
	}
}
