package redact

import (
	"strings"
	"testing"
)

// BenchmarkRedact_16KiBTokens (0028-MADR D-A10): 16 KiB of text that says
// "tokens", so the passes anchored on "token" run over all of it.
func BenchmarkRedact_16KiBTokens(b *testing.B) {
	text := []byte("This request used too many tokens. " + strings.Repeat("detail ", 2300))
	b.SetBytes(int64(len(text)))
	b.ReportAllocs()
	for b.Loop() {
		_ = Redact(text)
	}
}
