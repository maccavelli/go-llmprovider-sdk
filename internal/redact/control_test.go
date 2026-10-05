package redact

import "testing"

// TestStripControl (0021-MADR Z3): C0 controls but tab, DEL and C1 controls
// go; a newline becomes a space; clean text is returned as it is.
func TestStripControl(t *testing.T) {
	for in, want := range map[string]string{
		"plain text":                       "plain text",
		"tab\tkept":                        "tab\tkept",
		"\x1b]0;owned\x07\x1b[2Jhello":     "]0;owned[2Jhello",
		"del\x7f and c1\u009b and nul\x00": "del and c1 and nul",
		"two\nlines\r\nhere":               "two lines  here",
		"ünïcödé stays":                    "ünïcödé stays",
		"":                                 "",
	} {
		if got := StripControl(in); got != want {
			t.Errorf("StripControl(%q) = %q, want %q", in, got, want)
		}
	}
}
