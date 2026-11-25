package main

import (
	"os"
	"path/filepath"
	"testing"
)

// docArgs is args after "-doc FILE", FILE holding runDoc: the document the
// test fetcher answered with before the command required -doc (0093-MADR D6,
// D12, MH29).
func docArgs(t *testing.T, args ...string) []string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "api.json")
	if err := os.WriteFile(path, []byte(runDoc), 0o600); err != nil {
		t.Fatal(err)
	}
	return append([]string{"-doc", path}, args...)
}
