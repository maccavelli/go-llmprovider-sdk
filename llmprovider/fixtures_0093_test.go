package llmprovider_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// fixtureDoc is the committed models.dev-format fixture at rel, under
// llmprovider/catalog/testdata, handed in with catalog.WithMetadata
// (0093-MADR D12).
func fixtureDoc(t *testing.T, rel string) llmprovider.Option {
	t.Helper()
	f, err := os.Open(filepath.Join("catalog", "testdata", filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	m, err := catalog.ParseMetadata(f)
	if err != nil {
		t.Fatal(err)
	}
	return catalog.WithMetadata(m)
}

// extractDoc is the extract fixture: what a cold request read with no
// document before 0093, the embedded extract's limits and OpenCode's routes
// (0093-MADR D12, MH55). The live tests and their offline twins hand it in.
func extractDoc(t *testing.T) llmprovider.Option {
	return fixtureDoc(t, "extract-2026-10-12/api.json")
}
