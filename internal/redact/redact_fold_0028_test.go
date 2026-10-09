package redact

import (
	"bytes"
	"slices"
	"strings"
	"testing"
)

// 0028-PLAN D10: a replacement function matched its regex again on the match
// alone, which can fail where (?i) folds the long s (U+017F) or the Kelvin
// sign (U+212A) into ASCII and \b, ASCII-only, does not; and the same letters
// hid secrets from the passes.

// foldedCases hold the long s or the Kelvin sign where a name or label is;
// each secret must be masked, and nothing may panic.
var foldedCases = []struct{ in, secret string }{
	{"key api_Key=abcd1234efgh", "abcd1234efgh"},
	{"code x_Key: abcd1234efgh", "abcd1234efgh"},
	{"key {\"x_Key\":\"abcd1234efgh\"}", "abcd1234efgh"},
	{"Key=abcd1234efgh key", "abcd1234efgh"},
	{"the ſecret=abcd1234efgh here", "abcd1234efgh"},
	{"x_ſecret=abcd1234efgh", "abcd1234efgh"},
	{"token x_Key=abcd1234efgh", "abcd1234efgh"},
	{"x_ſet-cookie: a=b0c1d2e3", "b0c1d2e3"},
	{"Authorization: Bearer abcd1234efgh K", "abcd1234efgh"},
}

// TestRedact_FoldedLetters: each case's secret is masked, and the text around
// it is kept, with the two letters shown as s and k.
func TestRedact_FoldedLetters(t *testing.T) {
	for _, c := range foldedCases {
		got := String(c.in)
		if strings.Contains(got, c.secret) {
			t.Errorf("String(%q) = %q; want %q masked", c.in, got, c.secret)
		}
	}
	if got, want := String("x_ſet-cookie: a=b"), "x_set-cookie: a=[REDACTED]"; got != want {
		t.Errorf("cookie: %q; want %q", got, want)
	}
	if got, want := String("ſome Kind of text"), "some kind of text"; got != want {
		t.Errorf("no secret: %q; want %q", got, want)
	}
}

// TestRedact_FoldLeavesTheCallersBuffer: folding works on a copy.
func TestRedact_FoldLeavesTheCallersBuffer(t *testing.T) {
	in := []byte("Key=abcd1234efgh key")
	keep := slices.Clone(in)
	_ = Redact(in)
	if !bytes.Equal(in, keep) {
		t.Errorf("Redact changed its input: %q; want %q", in, keep)
	}
}

// TestReplaceSubmatches_InContext: the groups are read where the match
// stands, so the unfolded Kelvin case, whose match does not match reKVLong on
// its own, is masked without a panic.
func TestReplaceSubmatches_InContext(t *testing.T) {
	in := []byte("key api_Key=abcd1234efgh")
	if reKVLong.FindSubmatch(reKVLong.Find(in)) != nil {
		t.Fatal("the match matches on its own; the case no longer shows the fault")
	}
	if got, want := string(replaceSubmatches(reKVLong, in, redactKVLong)), "key api_Key=[REDACTED]"; got != want {
		t.Errorf("replaceSubmatches = %q; want %q", got, want)
	}
	if got := replaceSubmatches(reKVLong, []byte("nothing here"), redactKVLong); string(got) != "nothing here" {
		t.Errorf("no match: %q", got)
	}
}

// FuzzRedact_NeverPanics: Redact returns for any text, and never changes its
// input.
func FuzzRedact_NeverPanics(f *testing.F) {
	for _, c := range foldedCases {
		f.Add(c.in)
	}
	for _, c := range equivCorpus() {
		f.Add(c.text)
	}
	f.Fuzz(func(t *testing.T, s string) {
		in := []byte(s)
		_ = Redact(in)
		if string(in) != s {
			t.Fatalf("Redact changed its input %q to %q", s, in)
		}
	})
}
