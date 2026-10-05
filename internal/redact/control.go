package redact

import "strings"

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
