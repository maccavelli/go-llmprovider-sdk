package redact

import (
	"slices"
	"strings"
	"testing"
)

// authCorpus is the label, case, spacing and overlap shapes the keyword-first
// reAuth pass must read as the expression does (0068-MADR D5).
var authCorpus = []string{
	"Authorization: Bearer abcdefgh1234",
	"AUTHORIZATION=basic dXNlcjpwYXNzd29yZA==",
	"authorization :  token 12345678abcdef",
	"authorization: authorization: bearer abcdefgh1234",
	"Authorization: Bearer",
	"x-bearer abcd and y-basic efgh",
	"Bearer abc",
	"token abcdefg",
	"tokentoken token abcdefghij",
	"bearertoken abcdefghijkl",
	"Bearer eyJhbGciOi.payload.sig then Bearer second-token-value",
	"Invalid bearer token",
	"Basic authentication is not supported",
	"the token was too long: token abcdefgh, then authorization=Token zyxwvuts98",
	"BEARER\tABCDEFGH and bearer\n" + "ijklmnop",
	"Authorization: Basic dXNlcjpwYXNz" + strings.Repeat(" padding", 600) + " Bearer tail-credential-1234",
	"Authorization: Bearer invalid",
	"Bearer basic abcdefgh",
	"a Kelvin sign: toKen abcd1234efgh, Authorization: Bearer invalid",
}

// FuzzRedactAuthPass_MatchesExpression_0068: on any input, the keyword-first
// pass redacts exactly what reAuth's own search finds, each match replaced by
// redactAuth.
func FuzzRedactAuthPass_MatchesExpression_0068(f *testing.F) {
	for _, c := range equivCorpus() {
		f.Add(c.text)
	}
	for _, s := range authCorpus {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		p := []byte(s)
		want := s
		if reAuth.Match(p) {
			want = string(replaceSubmatches(reAuth, slices.Clone(p), redactAuth))
		}
		if got := string(redactAuthPass(slices.Clone(p))); got != want {
			t.Fatalf("redactAuthPass(%q)\n  = %q\n  want %q (the expression's own search)", s, got, want)
		}
	})
}
