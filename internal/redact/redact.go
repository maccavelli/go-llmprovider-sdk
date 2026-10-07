// Package redact removes and masks secrets. Redact and String hide a secret
// completely, so text is safe to log or to return in an error; MaskSecret shows
// a short suffix on purpose, so a person can tell which credential is meant.
package redact

import (
	"bytes"
	"fmt"
	"regexp"
	"slices"
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
// Each runs only when the text holds one of its literal anchors, compared
// case-insensitively: a regex pass over clean text is the cost, and a clean
// line contains none (0021-MADR Z1).
var (
	// reAuth matches "[Authorization:] (Bearer|Basic|Token) <credential>".
	// Groups 1 and 3 are the optional "Authorization:" label to preserve, and
	// groups 2 and 4 the credential. The greedy credential class is what fixes
	// the prior leftmost-match leak where the JWT following "Bearer" was
	// emitted in cleartext. redactAuth keeps an ordinary word (0026-MADR F21).
	reAuth = regexp.MustCompile(`(?i)((?:authorization\s*[:=]\s*)?)(?:bearer|basic)\s+([A-Za-z0-9._~+/=-]{4,})|((?:authorization\s*[:=]\s*)?)token\s+([A-Za-z0-9._~+/=-]{8,})`)

	// reKV matches key=value / "key": "value" secret assignments, in JSON
	// (escaped or not), form bodies and headers. Group 1 is the key name
	// (group 2) and separator to preserve; the value is redacted. Any name
	// ending in token counts, joined (refresh_token, id_token) or camelCase
	// (accessToken) (0020-MADR F8; 0021-MADR Z2), and so does any other secret
	// name with a snake_case, kebab-case or camelCase prefix (openai_api_key,
	// apiSecret, db_password), as does any prefixed key (subscription_key)
	// (0026-MADR F17). A quoted value runs to its closing quote, spaces and all
	// (groups 3-5 for ", 6-8 for '); an unquoted one to the first separator
	// (group 9).
	reKV = regexp.MustCompile(`(?i)(\b((?:[a-z0-9]+[_-]?)*(?:password|passwd|pwd|secret|api[_-]?key|access[_-]?key|client[_-]?secret|private[_-]?key|signature|token|authorization|code[_-]?verifier|device[_-]?code)|(?:[a-z0-9]+[_-])+key)\b\\?["']?\s*[:=]\s*)` +
		`(?:(\\?")([^"\\]{4,})(\\?")?|(')([^']{4,})(')?|([^"'\\\s,}&;]{4,}))`)

	// reKVLong is reKV for the bare names code and key, whose values are
	// redacted only from 8 characters, so a short diagnostic such as
	// "error code: 1102" stays readable (0020-MADR F8). Group 2 is the value.
	reKVLong = regexp.MustCompile(`(?i)(\b(?:code|key)\b\\?["']?\s*[:=]\s*\\?["']?)([^"'\\\s,}&]{8,})`)

	// reCookie matches a Cookie or Set-Cookie header's value; each of its
	// name=value pairs has its value redacted (0021-MADR Z2).
	reCookie = regexp.MustCompile(`(?i)(\b(?:set-)?cookie\s*:\s*)([^\r\n]+)`)
	// reCookiePair is one name=value pair; group 2 is the value.
	reCookiePair = regexp.MustCompile(`([^\s=;]+\s*=\s*)([^;]*)`)

	// reDiagnostic is a value that is a diagnostic, not a secret: a plain
	// number or a snake_case word, such as model_not_found. The bare names
	// code, key and token keep it (0021-MADR Z2).
	reDiagnostic = regexp.MustCompile(`^(?:\d+|[a-z]+(?:_[a-z]+)+)$`)

	// reKiloToken matches Kilo's URL-prefixed token, "{backend URL}:{secret}"
	// (internal/kiloendpoint); the URL is kept, the secret redacted.
	reKiloToken = regexp.MustCompile(`(https?://[^\s:@/]+(?::\d+)?(?:/[^\s:@]*)?):[A-Za-z0-9._~=-]{16,}`)

	// reToken matches standalone, self-identifying secrets; the whole match is
	// redacted. Vendor prefixes stay case-sensitive (they are issued literals);
	// the legacy keyword prefixes are matched case-insensitively to preserve the
	// original behavior.
	reToken = regexp.MustCompile(strings.Join([]string{
		`eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]*`,      // JWT, signed or not
		`\bsk-[A-Za-z0-9_-]{16,}`,                                // OpenAI, Anthropic (sk-ant-), OpenRouter (sk-or-)
		`\bxai-[A-Za-z0-9_-]{16,}`,                               // xAI
		`\btgp_[A-Za-z0-9_-]{16,}`,                               // Together
		`\bhf_[A-Za-z0-9]{16,}`,                                  // Hugging Face
		`\b(?:AKIA|ASIA|AGPA|AIDA|AROA|ANPA|ANVA)[0-9A-Z]{16}\b`, // AWS access key id
		`\bgh[posru]_[A-Za-z0-9]{20,}\b`,                         // GitHub token
		`\bgithub_pat_[A-Za-z0-9_]{20,}\b`,                       // GitHub fine-grained PAT
		`\bxox[baprs]-[A-Za-z0-9-]{10,}\b`,                       // Slack token
		`\b(?:sk|rk|pk)_(?:live|test)_[A-Za-z0-9]{16,}\b`,        // Stripe key
		`\bAIza[0-9A-Za-z_-]{35}`,                                // Google API key, which may end in - or _ (0021-MADR Z2)
		// PEM private key, from BEGIN to END; without an END, to the end of
		// its base64 body (0021-MADR Z2).
		`-----BEGIN (?:RSA |EC |OPENSSH |DSA |PGP |ENCRYPTED )?PRIVATE KEY-----(?:[\s\S]*?-----END (?:RSA |EC |OPENSSH |DSA |PGP |ENCRYPTED )?PRIVATE KEY-----|[A-Za-z0-9+/=\s]*)`,
		`\b(?i:token_|sk_|key_|secret_)[A-Za-z0-9._/+=-]{6,}`, // legacy keyword prefixes (word-anchored, min length to cut false positives)
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
	// The anchors are looked up in the text as it was before any pass: a
	// replacement only removes text and adds "[REDACTED]", so it never adds
	// an anchor, and a pass that looks needlessly costs time, not safety.
	// A message of up to 4 KiB, the common case, is lower-cased on the stack.
	var stack [4 << 10]byte
	lower := asciiLower(stack[:0], p)
	for _, pass := range passes {
		if !containsAny(lower, pass.anchors) || !pass.re.Match(p) {
			continue
		}
		if pass.replace != nil {
			p = pass.re.ReplaceAllFunc(p, pass.replace)
		} else {
			p = pass.re.ReplaceAll(p, pass.repl)
		}
	}
	return p
}

// pass is one redaction regex, its replacement (a template, or a function of
// the match), and the lower-case literals one of which every match contains.
type pass struct {
	re      *regexp.Regexp
	repl    []byte
	replace func([]byte) []byte
	anchors []string
}

// passes run in order; each one's anchors are a literal its regex cannot
// match without.
var passes = []pass{
	{re: reAuth, replace: redactAuth, anchors: []string{"bearer", "basic", "token"}},
	{re: reCookie, replace: redactCookie, anchors: []string{"cookie"}},
	{re: reKV, replace: redactKV, anchors: []string{"pass", "pwd", "secret", "key", "signature", "token", "authorization", "code"}},
	{re: reKVLong, replace: redactKVLong, anchors: []string{"code", "key"}},
	{re: reKiloToken, repl: []byte("${1}:[REDACTED]"), anchors: []string{"http"}},
	{re: reDSN, repl: []byte("${1}[REDACTED]@"), anchors: []string{"://"}},
	{re: reToken, repl: []byte("[REDACTED]"), anchors: []string{
		"eyj", "sk-", "xai-", "tgp_", "hf_", "akia", "asia", "agpa", "aida", "aroa", "anpa", "anva",
		"ghp_", "gho_", "ghs_", "ghr_", "ghu_", "github_pat_", "xox", "sk_", "rk_", "pk_", "aiza",
		"-----begin", "token_", "key_", "secret_",
	}},
}

// redacted is the marker a secret is replaced with.
var redacted = []byte("[REDACTED]")

// redactAuth redacts one reAuth match's scheme and credential, keeping its
// label. After an Authorization label the value is a credential, whatever its
// shape; without one, an ordinary word stays, so "Invalid bearer token" and
// "Basic authentication is not supported" keep their meaning (0026-MADR F21).
func redactAuth(m []byte) []byte {
	sub := reAuth.FindSubmatch(m)
	label, value := sub[1], sub[2]
	if value == nil {
		label, value = sub[3], sub[4]
	}
	if len(label) == 0 && !credentialShaped(value) {
		return m
	}
	return slices.Concat(label, redacted)
}

// credentialShaped reports whether v looks like a credential rather than a
// word: it holds a digit, token punctuation or a capital after its first
// letter, or it is longer than an ordinary word. A full stop that ends a
// sentence does not count.
func credentialShaped(v []byte) bool {
	v = bytes.TrimRight(v, ".")
	if len(v) >= 20 {
		return true
	}
	for i, c := range v {
		switch {
		case '0' <= c && c <= '9', strings.IndexByte("._~+/=-", c) >= 0, i > 0 && 'A' <= c && c <= 'Z':
			return true
		}
	}
	return false
}

// redactKV redacts one reKV match's value, keeping its quotes. A bare token
// key keeps a diagnostic value, such as "token: 128000".
func redactKV(m []byte) []byte {
	sub := reKV.FindSubmatch(m)
	value := sub[4]
	if value == nil {
		value = sub[7]
	}
	if value == nil {
		value = sub[9]
	}
	if bytes.EqualFold(sub[2], []byte("token")) && reDiagnostic.Match(value) {
		return m
	}
	return slices.Concat(sub[1], sub[3], sub[6], redacted, sub[5], sub[8])
}

// redactKVLong redacts one reKVLong match's value, unless it is a diagnostic.
func redactKVLong(m []byte) []byte {
	sub := reKVLong.FindSubmatch(m)
	if reDiagnostic.Match(sub[2]) {
		return m
	}
	return slices.Concat(sub[1], redacted)
}

// redactCookie redacts the value of each pair in one cookie header.
func redactCookie(m []byte) []byte {
	sub := reCookie.FindSubmatch(m)
	pairs := reCookiePair.ReplaceAll(sub[2], []byte("${1}[REDACTED]"))
	return slices.Concat(sub[1], pairs)
}

// asciiLower appends p to dst with ASCII letters lower-cased; other bytes are
// copied as they are, which suffices for the ASCII anchors.
func asciiLower(dst, p []byte) []byte {
	for _, c := range p {
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		dst = append(dst, c)
	}
	return dst
}

// containsAny reports whether s holds any of the literals.
func containsAny(s []byte, literals []string) bool {
	for _, l := range literals {
		if bytes.Contains(s, []byte(l)) {
			return true
		}
	}
	return false
}

// reDSN matches credentials embedded in connection strings / URIs
// (scheme://user:pass@host); the scheme is preserved, the user:pass redacted.
var reDSN = regexp.MustCompile(`(?i)\b([a-z][a-z0-9+.-]*://)[^:@/\s]*:[^@/\s]+@`)

// String is the string convenience wrapper around Redact.
func String(s string) string {
	return string(Redact([]byte(s)))
}
