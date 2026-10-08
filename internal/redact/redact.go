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

	// reDiagnosticCode is a bare code's diagnostic as well: two or more
	// letter-only words joined by - or _, in any case, such as Grok's
	// invalid-argument and Kilo's INVALID_TOKEN (0028-MADR A10b). A key or a
	// token keeps reDiagnostic alone: a random value without a digit can take
	// this shape, and a key's value is a secret far more often than a code's.
	reDiagnosticCode = regexp.MustCompile(`^[A-Za-z]+(?:[-_][A-Za-z]+)+$`)

	// reKiloToken matches Kilo's URL-prefixed token, "{backend URL}:{secret}"
	// (internal/kiloendpoint); the URL is kept, the secret redacted.
	reKiloToken = regexp.MustCompile(`(https?://[^\s:@/]+(?::\d+)?(?:/[^\s:@]*)?):[A-Za-z0-9._~=-]{16,}`)

	// reToken matches standalone, self-identifying secrets; the whole match is
	// redacted. Vendor prefixes stay case-sensitive (they are issued literals);
	// the legacy keyword prefixes are matched case-insensitively to preserve the
	// original behavior.
	reToken = regexp.MustCompile(strings.Join([]string{
		`eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]*`, // JWT, signed or not
		// Vendor shapes, each from 0028-PLAN D3's research (gitleaks,
		// betterleaks, vendor code and docs) and, where the owner holds
		// one, measured on a key. The generic sk- and xai- rows are
		// reTokenFallback's.
		`\bsk-[A-Za-z0-9]{20}T3BlbkFJ[A-Za-z0-9]{20}\b`,          // OpenAI, legacy (gitleaks; codex)
		`\bsk-(?:proj|svcacct|admin|None)-[A-Za-z0-9_-]{16,}`,    // OpenAI, typed (gitleaks; codex; OpenAPI spec)
		`\bsk-ant-[a-z]{2,8}[0-9]{2}-[A-Za-z0-9_-]{16,}`,         // Anthropic api03, admin01, oat01, ort01 (docs; gitleaks)
		`\bsk-or-v1-[0-9a-f]{64}\b`,                              // OpenRouter (docs)
		`\bsk-[A-Za-z0-9]{64}\b`,                                 // OpenCode Zen and Go (opencode key.ts)
		`\bxai-[A-Za-z0-9]{80}\b`,                                // xAI (trufflehog; measured)
		`\btgp_v1_[A-Za-z0-9_-]{43}`,                             // Together (betterleaks; measured)
		`\btgp_[A-Za-z0-9_-]{16,}`,                               // Together, any other
		`\bhf_[A-Za-z0-9]{34}\b`,                                 // Hugging Face (gitleaks; trufflehog; measured)
		`\bhf_[A-Za-z0-9]{16,}`,                                  // Hugging Face, any other length (0028-PLAN D7)
		`\bapi_org_[A-Za-z0-9]{34}\b`,                            // Hugging Face org (gitleaks)
		`\bAQ\.Ab[0-9A-Za-z_-]{30,}`,                             // Gemini auth key (Gemini docs; betterleaks)
		`\bya29\.[0-9A-Za-z_-]{20,}`,                             // Google OAuth access token (Google docs; Nosey Parker)
		`\b1//0[0-9A-Za-z_-]{40,}`,                               // Google OAuth refresh token (Google docs); 0 spares Python's 1//x
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

	// reTokenFallback is the generic sk- and xai- key, for a format no row of
	// reToken knows: 16 or more characters, today's floor, masked unless the
	// body is two or more lower-case words joined by -, so that
	// "sk-learn-tutorial-for-beginners" and
	// "xai-grok-login-device-code-flow-tests" are kept (0028-MADR A10c;
	// 0028-PLAN D6, D7).
	reTokenFallback = regexp.MustCompile(`\b(?:sk|xai)-[A-Za-z0-9_-]{16,}`)

	// reWordBody is a body of lower-case words joined by -, which
	// reTokenFallback keeps.
	reWordBody = regexp.MustCompile(`^[a-z]+(?:-[a-z]+)+$`)

	// reOllamaKey is Ollama's cloud key, 32 hex digits, a dot and 24
	// characters (betterleaks; Kingfisher), looked for only in text that
	// names Ollama, as those rules do.
	reOllamaKey = regexp.MustCompile(`\b[0-9a-f]{32}\.[A-Za-z0-9_-]{24}\b`)
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
	var seen anchorsSeen
	for i, pass := range passes {
		if !seen.any(lower, passAnchors[i]) {
			continue
		}
		if pass.apply != nil {
			p = pass.apply(p)
			continue
		}
		if !pass.re.Match(p) {
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
// apply, when set, does the whole pass in place of the regex's own search.
type pass struct {
	re      *regexp.Regexp
	repl    []byte
	replace func([]byte) []byte
	apply   func([]byte) []byte
	anchors []string
}

// passes run in order; each one's anchors are a literal its regex cannot
// match without.
var passes = []pass{
	{re: reAuth, replace: redactAuth, anchors: []string{"bearer", "basic", "token"}},
	{re: reCookie, replace: redactCookie, anchors: []string{"cookie"}},
	{re: reKV, apply: redactKVPass, anchors: []string{"pass", "pwd", "secret", "key", "signature", "token", "authorization", "code"}},
	{re: reKVLong, replace: redactKVLong, anchors: []string{"code", "key"}},
	{re: reKiloToken, repl: []byte("${1}:[REDACTED]"), anchors: []string{"http"}},
	{re: reDSN, repl: []byte("${1}[REDACTED]@"), anchors: []string{"://"}},
	{re: reToken, repl: []byte("[REDACTED]"), anchors: []string{
		"eyj", "sk-", "xai-", "tgp_", "hf_", "akia", "asia", "agpa", "aida", "aroa", "anpa", "anva",
		"ghp_", "gho_", "ghs_", "ghr_", "ghu_", "github_pat_", "xox", "sk_", "rk_", "pk_", "aiza",
		"-----begin", "token_", "key_", "secret_", "api_org_", "aq.ab", "ya29.", "1//0",
	}},
	{re: reTokenFallback, replace: redactTokenFallback, anchors: []string{"sk-", "xai-"}},
	{re: reOllamaKey, repl: []byte("[REDACTED]"), anchors: []string{"ollama"}},
}

// redactTokenFallback masks one reTokenFallback match, unless its body,
// after the prefix, is lower-case words joined by -.
func redactTokenFallback(m []byte) []byte {
	if reWordBody.Match(m[bytes.IndexByte(m, '-')+1:]) {
		return m
	}
	return redacted
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

// reKVAnchored is reKV anchored at its start, with the same groups: the pass
// tries it only where a match can start.
var reKVAnchored = regexp.MustCompile(`^(?:` + reKV.String() + `)`)

// kvKeywords are literals one of which every reKV name contains: the
// expression's keywords, with api_key's and the other keys' "key", and
// code_verifier's "verifier".
var kvKeywords = []string{"password", "passwd", "pwd", "secret", "key", "signature", "token", "authorization", "verifier", "code"}

// redactKVPass is the reKV pass, keyword first (0028-MADR D-A10). reKV's name
// begins with a repetition that an unanchored search tries from every word
// boundary, so 16 KiB of text cost milliseconds. Every match's name holds a
// keyword, and starts at a word boundary in the run of name bytes
// ([A-Za-z0-9_-]) that holds it; so the pass finds each keyword, and tries
// reKV anchored at each boundary in its run, in order, taking the first that
// matches. That is the match the unanchored search finds, leftmost first, and
// redactKV replaces it as before. Text with a byte outside ASCII takes the
// regex's own search: (?i) folds letters such as the Kelvin sign into ASCII
// ones, which a byte search for the keywords would miss.
func redactKVPass(p []byte) []byte {
	if !isASCII(p) {
		if !reKV.Match(p) {
			return p
		}
		return reKV.ReplaceAllFunc(p, redactKV)
	}
	var stack [4 << 10]byte
	lower := asciiLower(stack[:0], p)
	var out []byte
	last, pos := 0, 0
	for _, ks := range keywordStarts(lower) {
		if ks < pos {
			continue
		}
		run := ks
		for run > pos && isNameByte(p[run-1]) {
			run--
		}
		for s := run; s <= ks; s++ {
			// \b before a word character: the byte before is not one.
			if !isWordByte(p[s]) || (s > 0 && isWordByte(p[s-1])) {
				continue
			}
			loc := reKVAnchored.FindIndex(p[s:])
			if loc == nil {
				continue
			}
			end := s + loc[1]
			out = append(out, p[last:s]...)
			out = append(out, redactKV(p[s:end])...)
			last, pos = end, end
			break
		}
	}
	if out == nil {
		return p
	}
	return append(out, p[last:]...)
}

// keywordStarts are the offsets in lower of every kvKeywords occurrence, in
// order, each once.
func keywordStarts(lower []byte) []int {
	var starts []int
	for _, kw := range kvKeywords {
		for i := 0; ; {
			j := bytes.Index(lower[i:], []byte(kw))
			if j < 0 {
				break
			}
			starts = append(starts, i+j)
			i += j + 1
		}
	}
	slices.Sort(starts)
	return slices.Compact(starts)
}

// isWordByte is regexp's ASCII \w.
func isWordByte(c byte) bool {
	return 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' || c == '_'
}

// isNameByte is a byte reKV's name can hold.
func isNameByte(c byte) bool { return isWordByte(c) || c == '-' }

// isASCII reports whether p holds only ASCII bytes.
func isASCII(p []byte) bool {
	for _, c := range p {
		if c >= 0x80 {
			return false
		}
	}
	return true
}

// diagnosticCodeLimit bounds a code reDiagnosticCode keeps.
const diagnosticCodeLimit = 40

// redactKVLong redacts one reKVLong match's value, unless it is a diagnostic.
func redactKVLong(m []byte) []byte {
	sub := reKVLong.FindSubmatch(m)
	if reDiagnostic.Match(sub[2]) {
		return m
	}
	if len(sub[1]) >= 4 && bytes.EqualFold(sub[1][:4], []byte("code")) && len(sub[2]) <= diagnosticCodeLimit &&
		reDiagnosticCode.Match(sub[2]) {
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

// anchors are the distinct anchors of passes, and passAnchors[i] the
// indexes in anchors of passes[i]'s: an anchor two passes share, such as
// "token", is looked up once per Redact call (0028-PLAN Phase 4, step 7).
var anchors, passAnchors = indexAnchors(passes)

// anchorRare[j] is the offset in anchors[j] of its rarest byte in prose.
var anchorRare = rarestOffsets(anchors)

// byteCommonness ranks bytes from the most common in English prose and error
// messages; a byte not listed is rarer than any listed.
const byteCommonness = " etaoinsrhldcumfpgwyb.,v-k:\"x/0123456789jqz_"

// rarestOffsets returns, for each literal, the offset of its rarest byte.
func rarestOffsets(literals []string) []int {
	at := make([]int, len(literals))
	for i, l := range literals {
		best := -1
		for k := range len(l) {
			r := strings.IndexByte(byteCommonness, l[k])
			if r < 0 {
				r = len(byteCommonness)
			}
			if r > best {
				best, at[i] = r, k
			}
		}
	}
	return at
}

// anchorIn reports whether s holds lit, searching for lit's byte at offset
// rare and checking the whole of lit around each one found.
func anchorIn(s []byte, lit string, rare int) bool {
	c := lit[rare]
	for i := 0; i < len(s); {
		j := bytes.IndexByte(s[i:], c)
		if j < 0 {
			return false
		}
		j += i
		if start := j - rare; start >= 0 && start+len(lit) <= len(s) && string(s[start:start+len(lit)]) == lit {
			return true
		}
		i = j + 1
	}
	return false
}

// maxAnchors bounds anchors, so that anchorsSeen lives on the stack.
const maxAnchors = 64

// indexAnchors numbers the distinct anchors of ps.
func indexAnchors(ps []pass) ([]string, [][]int) {
	var distinct []string
	byPass := make([][]int, len(ps))
	for i, p := range ps {
		for _, a := range p.anchors {
			j := slices.Index(distinct, a)
			if j < 0 {
				j = len(distinct)
				distinct = append(distinct, a)
			}
			byPass[i] = append(byPass[i], j)
		}
	}
	if len(distinct) > maxAnchors {
		panic("redact: more than maxAnchors anchors")
	}
	return distinct, byPass
}

// anchorsSeen records, for one Redact call, what each anchor's lookup found.
type anchorsSeen [maxAnchors]uint8

// The states of an anchorsSeen entry.
const (
	anchorUnknown uint8 = iota
	anchorPresent
	anchorAbsent
)

// any reports whether lower holds any of the anchors indexed by idx, looking
// each up at most once per call.
func (seen *anchorsSeen) any(lower []byte, idx []int) bool {
	for _, j := range idx {
		if seen[j] == anchorUnknown {
			seen[j] = anchorAbsent
			if anchorIn(lower, anchors[j], anchorRare[j]) {
				seen[j] = anchorPresent
			}
		}
		if seen[j] == anchorPresent {
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
