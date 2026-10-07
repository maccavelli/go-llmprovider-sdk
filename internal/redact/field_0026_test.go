package redact

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// TestField (0026-MADR F18): a short field keeps its text, stripped of
// control characters; a long one is cut to FieldLimit bytes, on a rune
// boundary, so the result is valid UTF-8.
func TestField(t *testing.T) {
	for in, want := range map[string]string{
		"":                                "",
		"max_output_tokens":               "max_output_tokens",
		"\x1b]0;owned\x07SAFETY":          "]0;ownedSAFETY",
		strings.Repeat("r", FieldLimit):   strings.Repeat("r", FieldLimit),
		strings.Repeat("r", FieldLimit+1): strings.Repeat("r", FieldLimit),
	} {
		if got := Field(in); got != want {
			t.Errorf("Field(%d bytes) = %q; want %q", len(in), got, want)
		}
	}
	// "é" is two bytes; 127 ASCII bytes put the 128-byte cut inside it.
	long := strings.Repeat("r", FieldLimit-1) + strings.Repeat("é", 4)
	got := Field(long)
	if len(got) != FieldLimit-1 || !utf8.ValidString(got) {
		t.Errorf("Field across a rune: %d bytes, valid UTF-8 %t; want %d bytes, valid", len(got), utf8.ValidString(got), FieldLimit-1)
	}
}
