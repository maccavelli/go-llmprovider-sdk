package responses

import (
	"strings"
	"testing"
)

// TestIncomplete_ReasonIsBounded (0026-MADR F18): the service's
// incomplete_details.reason reaches Error() stripped of control characters
// and bounded.
func TestIncomplete_ReasonIsBounded(t *testing.T) {
	msg := incomplete("\x1b[2J\x07" + strings.Repeat("r", 2048)).Error()
	if strings.ContainsAny(msg, "\x1b\x07") || len(msg) > 512 {
		t.Fatalf("Error() holds ESC=%t BEL=%t and is %d bytes; want neither, and a bounded reason",
			strings.Contains(msg, "\x1b"), strings.Contains(msg, "\x07"), len(msg))
	}
}
