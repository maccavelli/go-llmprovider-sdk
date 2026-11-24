package catalog

import "testing"

// ReplaceExtract replaces the embedded extract with raw for t: a test hook for
// the external TestOutputLimit_Fallback_0089 (0089-MADR DL51), which needs a
// route where no extract section states a limit.
func ReplaceExtract(t *testing.T, raw string) {
	t.Helper()
	withExtract(t, raw)
}
