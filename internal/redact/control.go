package redact

import (
	"strings"
	"unicode/utf8"
)

// FieldLimit bounds a short field a service sends, such as an error code or
// a finish reason, where it reaches an error's text.
const FieldLimit = 128

// Field strips s of control characters and trims it to FieldLimit bytes on a
// rune boundary: a service's code or reason reaches Error() and so a terminal
// (0021-MADR Z3, 0026-MADR F18).
func Field(s string) string {
	s = StripControl(s)
	if len(s) <= FieldLimit {
		return s
	}
	cut := FieldLimit
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}

// StripControl removes the characters a terminal acts on: C0 controls but
// tab, DEL, and the C1 controls U+0080 to U+009F. A newline or carriage
// return becomes a space, so the words on either side stay apart. Text from
// a server or an IdP passes through it before it reaches a terminal, an
// error or a log, so it cannot move the cursor, clear the screen or set the
// window title (0021-MADR Z3).
func StripControl(s string) string {
	if !strings.ContainsFunc(s, isControl) {
		return s
	}
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\n' || r == '\r':
			return ' '
		case isControl(r):
			return -1
		}
		return r
	}, s)
}

// isControl reports a C0 control other than tab, DEL, or a C1 control.
func isControl(r rune) bool {
	return r < 0x20 && r != '\t' || r == 0x7f || r >= 0x80 && r <= 0x9f
}
