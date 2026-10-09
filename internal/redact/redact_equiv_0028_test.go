package redact

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// 0028-MADR D-A10: the keyword-first reKV pass must redact exactly what the
// expression's own search did. The frozen reference (redact_ref_test.go) is
// Redact as it was at 0028-PLAN Phase 4's start.

// equivCorpus is a named corpus: hand-picked edges, then 200 strings
// generated with seed 28 from the shapes reKV and its neighbours see.
func equivCorpus() []struct{ name, text string } {
	cases := []struct{ name, text string }{
		{"key at the start", "api_key=abcd1234efgh more"},
		{"two keys", `{"api_key":"abcd1234efgh","refresh_token":"zyxw9876vuts"}`},
		{"camelCase", "accessToken: Zm9vYmFyYmF6cXV4 then"},
		{"double separator", "foo--key=abcd1234efgh"},
		{"keyword inside a word", "monkey=bananas123 and turkey: stuffing99"},
		{"keyword, no assignment", "This request used too many tokens. Please retry."},
		{"bare token diagnostic", "token: 128000 and token=model_not_found"},
		{"escaped JSON", `{\"client_secret\":\"s3cr3t-v4lue\"}`},
		{"single quotes", "password = 'hunter2 with spaces' done"},
		{"Kelvin sign", "api_Key=abcd1234efgh"},
		{"long s", "ſecret=abcd1234efgh"},
		{"form body", "grant_type=refresh_token&refresh_token=rt_abcdefgh12345&client_id=x"},
		{"after a match", "token=aaaabbbb1111token=ccccdddd2222"},
		{"prefix run", "x-my-secret: valuevaluevalue; y=1"},
		{"header", "Authorization: Bearer abc.def.ghi and x-api-key: 0123456789abcdef"},
		// A keyword mid-word, after a run that cannot start a name: reKV has
		// no \b before it, so nothing matches.
		{"double underscore", "a__token=abcd1234efgh"},
		{"mid-word secret", "x__client_secret: s3cr3tv4lue"},
		{"mid-word after digits", "9__password=hunter2hunter2"},
	}
	r := rand.New(rand.NewPCG(28, 0))
	names := []string{"api_key", "apiKey", "API-KEY", "token", "accessToken", "refresh_token", "id-token", "secret",
		"client_secret", "password", "passwd", "pwd", "signature", "authorization", "code_verifier", "device_code",
		"code", "key", "subscription_key", "db_password", "my-api_key", "monkey", "tokens", "keyboard"}
	seps := []string{"=", ": ", ":", "\":\"", "\": \"", " = ", "='", "\\\":\\\""}
	values := []string{"abcd1234efgh", "zyxw9876vuts", "model_not_found", "128000", "hunter2", "sk-not-a-key",
		"a b c d", "ok", "Zm9vYmFyYmF6cXV4", "x", "invalid-argument", "INVALID_TOKEN"}
	fillers := []string{" ", ", ", "&", "; ", "\n", " and ", "}{", " the ", "\t", "--", ""}
	for i := range 200 {
		var b strings.Builder
		for range 1 + r.IntN(5) {
			b.WriteString(fillers[r.IntN(len(fillers))])
			b.WriteString(names[r.IntN(len(names))])
			b.WriteString(seps[r.IntN(len(seps))])
			b.WriteString(values[r.IntN(len(values))])
			if r.IntN(3) == 0 {
				b.WriteString(`"`)
			}
		}
		cases = append(cases, struct{ name, text string }{fmt.Sprintf("gen-%03d", i), b.String()})
	}
	return cases
}

// changedBy0028 are the corpus cases whose redaction 0028's A10b and A10c
// change on purpose; every other case must match the reference byte for byte.
var changedBy0028 = map[string]bool{
	"gen-192":     true, // A10b: code INVALID_TOKEN is a diagnostic, kept
	"Kelvin sign": true, // D10: folded to k, the key is masked
	"long s":      true, // D10: folded to s, the secret is masked
}

// TestRedact_MatchesReference: Redact equals the frozen reference on the
// corpus, except exactly the cases changedBy0028 names.
func TestRedact_MatchesReference(t *testing.T) {
	var differ []string
	for _, c := range equivCorpus() {
		got, want := String(c.text), string(redactReference([]byte(c.text)))
		if got != want {
			differ = append(differ, c.name)
			if !changedBy0028[c.name] {
				t.Errorf("%s: %q\n  got  %q\n  want %q (the reference)", c.name, c.text, got, want)
			}
		}
	}
	for name := range changedBy0028 {
		if !slices.Contains(differ, name) {
			t.Errorf("%s is listed as changed by 0028, but matches the reference", name)
		}
	}
}

// existingCaseFiles hold the package's cases from before 0028.
var existingCaseFiles = []string{"redact_test.go", "redact_providers_test.go", "redact_0026_test.go"}

// existingCases returns every string literal of existingCaseFiles, and every
// sum of string literals, such as "AIza" + "SyA…", as written in the source.
func existingCases(t *testing.T) []string {
	t.Helper()
	var cases []string
	fset := token.NewFileSet()
	for _, name := range existingCaseFiles {
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if e, ok := n.(ast.Expr); ok {
				if s, ok := stringConstant(e); ok {
					cases = append(cases, s)
				}
			}
			return true
		})
	}
	return cases
}

// stringConstant evaluates e when it is a string literal or a sum of them.
func stringConstant(e ast.Expr) (string, bool) {
	switch e := e.(type) {
	case *ast.BasicLit:
		if e.Kind != token.STRING {
			return "", false
		}
		s, err := strconv.Unquote(e.Value)
		return s, err == nil
	case *ast.BinaryExpr:
		if e.Op != token.ADD {
			return "", false
		}
		x, ok := stringConstant(e.X)
		if !ok {
			return "", false
		}
		y, ok := stringConstant(e.Y)
		return x + y, ok
	case *ast.ParenExpr:
		return stringConstant(e.X)
	}
	return "", false
}

// TestRedact_MatchesReferenceOnExistingCases: Redact equals the frozen
// reference on every case the package's tests held before 0028 (0028-PLAN
// Phase 4, step 5). 0028's A10b and A10c change none of them.
func TestRedact_MatchesReferenceOnExistingCases(t *testing.T) {
	cases := existingCases(t)
	if len(cases) < 100 {
		t.Fatalf("%d cases read from %v; want the package's tables", len(cases), existingCaseFiles)
	}
	for _, c := range cases {
		if got, want := String(c), string(redactReference([]byte(c))); got != want {
			t.Errorf("%q\n  got  %q\n  want %q (the reference)", c, got, want)
		}
	}
}

// FuzzRedact_MatchesReference: the keyword-first reKV pass redacts exactly
// what the frozen expression's own search finds, with today's replacement
// read in context (0028-PLAN D10: the reference's re-matching is a fault).
// It isolates the rewritten pass, so A10b's and A10c's changes elsewhere do
// not touch it.
func FuzzRedact_MatchesReference(f *testing.F) {
	for _, c := range equivCorpus() {
		f.Add(c.text)
	}
	f.Fuzz(func(t *testing.T, s string) {
		p := []byte(s)
		want := s
		if refReKV.Match(p) {
			want = string(replaceSubmatches(refReKV, slices.Clone(p), redactKV))
		}
		if got := string(redactKVPass(slices.Clone(p))); got != want {
			t.Fatalf("redactKVPass(%q)\n  = %q\n  want %q (the expression's own search)", s, got, want)
		}
	})
}
