// Package redact removes and masks secrets. Redact and String hide a secret
// completely, so text is safe to log or to return in an error; MaskSecret shows
// a short suffix on purpose, so a person can tell which credential is meant.
package redact

import (
	"fmt"
	"regexp"
	"strings"
)

// maxRedactBytes bounds the work (and stored size) for a single redaction call.
// A log line larger than this is almost always an accidental payload dump, so it
// is truncated with a marker before the regex passes run.
const maxRedactBytes = 256 * 1024

// This file is the single source of truth for secret redaction in this module.
// Provider error bodies and OAuth error responses funnel through String, so the
// patterns can never drift apart.
//
// Patterns are split into three precompiled regexes by replacement style:
//   - reAuth / reKV preserve a human-readable label (capture group 1) and
//     redact only the credential, e.g. "Authorization: [REDACTED]".
//   - reToken redacts the whole match for self-identifying standalone secrets.
//
// Each is guarded by a cheap Match() so clean lines incur no allocation.
var (
	// reAuth matches "[Authorization:] (Bearer|Basic|Token) <credential>".
	// Group 1 is the optional "Authorization:" label to preserve. The greedy
	// credential class is what fixes the prior leftmost-match leak where the JWT
	// following "Bearer" was emitted in cleartext.
	reAuth = regexp.MustCompile(`(?i)((?:authorization\s*[:=]\s*)?)(?:bearer|basic)\s+[A-Za-z0-9._~+/=-]{4,}|((?:authorization\s*[:=]\s*)?)token\s+[A-Za-z0-9._~+/=-]{8,}`)

	// reKV matches key=value / "key": "value" secret assignments, in JSON
	// (escaped or not), form bodies and headers. Group 1 is the key name +
	// separator (+ optional quote) to preserve; the value is redacted. Any
	// name ending in token (refresh_token, id_token) counts (0020-MADR F8).
	reKV = regexp.MustCompile(`(?i)(\b(?:password|passwd|pwd|secret|api[_-]?key|access[_-]?key|client[_-]?secret|(?:[a-z]+[_-])*token|authorization|cookie|code[_-]?verifier|device[_-]?code)\b\\?["']?\s*[:=]\s*\\?["']?)[^"'\\\s,}&]{4,}`)

	// reKVLong is reKV for the bare names code and key, whose values are
	// redacted only from 8 characters, so a short diagnostic such as
	// "error code: 1102" stays readable (0020-MADR F8).
	reKVLong = regexp.MustCompile(`(?i)(\b(?:code|key)\b\\?["']?\s*[:=]\s*\\?["']?)[^"'\\\s,}&]{8,}`)

	// reKiloToken matches Kilo's URL-prefixed token, "{backend URL}:{secret}"
	// (internal/kiloendpoint); the URL is kept, the secret redacted.
	reKiloToken = regexp.MustCompile(`(https?://[^\s:@/]+(?::\d+)?(?:/[^\s:@]*)?):[A-Za-z0-9._~=-]{16,}`)

	// reToken matches standalone, self-identifying secrets; the whole match is
	// redacted. Vendor prefixes stay case-sensitive (they are issued literals);
	// the legacy keyword prefixes are matched case-insensitively to preserve the
	// original behavior.
	reToken = regexp.MustCompile(strings.Join([]string{
		`eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]*`,                      // JWT, signed or not
		`\bsk-[A-Za-z0-9_-]{16,}`,                                                // OpenAI, Anthropic (sk-ant-), OpenRouter (sk-or-)
		`\bxai-[A-Za-z0-9_-]{16,}`,                                               // xAI
		`\btgp_[A-Za-z0-9_-]{16,}`,                                               // Together
		`\bhf_[A-Za-z0-9]{16,}`,                                                  // Hugging Face
		`\b(?:AKIA|ASIA|AGPA|AIDA|AROA|ANPA|ANVA)[0-9A-Z]{16}\b`,                 // AWS access key id
		`\bgh[posru]_[A-Za-z0-9]{20,}\b`,                                         // GitHub token
		`\bgithub_pat_[A-Za-z0-9_]{20,}\b`,                                       // GitHub fine-grained PAT
		`\bxox[baprs]-[A-Za-z0-9-]{10,}\b`,                                       // Slack token
		`\b(?:sk|rk|pk)_(?:live|test)_[A-Za-z0-9]{16,}\b`,                        // Stripe key
		`\bAIza[0-9A-Za-z_-]{35}\b`,                                              // Google API key
		`-----BEGIN (?:RSA |EC |OPENSSH |DSA |PGP |ENCRYPTED )?PRIVATE KEY-----`, // PEM private key header
		`\b(?i:token_|sk_|key_|secret_)[A-Za-z0-9._/+=-]{6,}`,                    // legacy keyword prefixes (word-anchored, min length to cut false positives)
	}, "|"))
)

// Redact removes common secret formats from p, returning p unchanged (same
// backing array) when nothing matches. Safe for nil/empty input.
func Redact(p []byte) []byte {
	if len(p) > maxRedactBytes {
		// 3-index slice caps capacity so append allocates a fresh array and never
		// mutates the caller's buffer.
		marker := fmt.Sprintf("…[truncated %d bytes]\n", len(p)-maxRedactBytes)
		p = append(p[:maxRedactBytes:maxRedactBytes], marker...)
	}
	if reAuth.Match(p) {
		p = reAuth.ReplaceAll(p, []byte("${1}${2}[REDACTED]"))
	}
	if reKV.Match(p) {
		p = reKV.ReplaceAll(p, []byte("${1}[REDACTED]"))
	}
	if reKVLong.Match(p) {
		p = reKVLong.ReplaceAll(p, []byte("${1}[REDACTED]"))
	}
	if reKiloToken.Match(p) {
		p = reKiloToken.ReplaceAll(p, []byte("${1}:[REDACTED]"))
	}
	if reDSN.Match(p) {
		p = reDSN.ReplaceAll(p, []byte("${1}[REDACTED]@"))
	}
	if reToken.Match(p) {
		p = reToken.ReplaceAll(p, []byte("[REDACTED]"))
	}
	return p
}

// reDSN matches credentials embedded in connection strings / URIs
// (scheme://user:pass@host); the scheme is preserved, the user:pass redacted.
var reDSN = regexp.MustCompile(`(?i)\b([a-z][a-z0-9+.-]*://)[^:@/\s]*:[^@/\s]+@`)

// String is the string convenience wrapper around Redact.
func String(s string) string {
	return string(Redact([]byte(s)))
}
