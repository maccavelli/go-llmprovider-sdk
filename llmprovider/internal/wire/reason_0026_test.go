package wire

import (
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestEmptyAnswer_ReasonIsBounded (0026-MADR F18): a finish reason is the
// service's text, so it reaches Error() stripped of control characters and
// bounded, as a code is (0021-MADR Z3).
func TestEmptyAnswer_ReasonIsBounded(t *testing.T) {
	raw := "\x1b]0;owned\x07" + strings.Repeat("r", 2048)
	err := EmptyAnswer("test", llmprovider.FinishReason(raw))
	msg := err.Error()
	if strings.ContainsAny(msg, "\x1b\x07") || len(msg) > 512 {
		t.Fatalf("Error() holds ESC=%t BEL=%t and is %d bytes; want neither, and a bounded reason",
			strings.Contains(msg, "\x1b"), strings.Contains(msg, "\x07"), len(msg))
	}
}
