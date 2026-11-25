package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

// ExtractFixture is 0093's extract fixture, parsed: the extract's limits, and
// OpenCode's routes at the record's base, as a models.dev document. A test
// hook for the external TestOutputLimit_Fallback_0089, whose fresh cache
// read the extract (0093-MADR D12, MH55).
func ExtractFixture(t *testing.T) Metadata {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", "extract-2026-10-12", "api.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	m, err := ParseMetadata(f)
	if err != nil {
		t.Fatal(err)
	}
	return m
}
