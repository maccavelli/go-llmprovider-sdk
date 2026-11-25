package providers

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// extractFixture is llmprovider/catalog/testdata/extract-2026-10-12/api.json,
// parsed once: the extract the SDK embedded until 0093, its limits and
// OpenCode's routes, as a models.dev document (0093-MADR D12, MH55).
var extractFixture = sync.OnceValues(func() (catalog.Metadata, error) {
	f, err := os.Open(filepath.Join("..", "catalog", "testdata", "extract-2026-10-12", "api.json"))
	if err != nil {
		return catalog.Metadata{}, err
	}
	defer func() { _ = f.Close() }()
	return catalog.ParseMetadata(f)
})

// extractDoc hands the extract fixture in with catalog.WithMetadata: what a
// request read with no document before 0093. A test that read the extract
// without naming it hands it in (0093-MADR D12, MH55). It panics when the
// committed fixture cannot be read, which fails the package's run.
func extractDoc() llmprovider.Option {
	m, err := extractFixture()
	if err != nil {
		panic("the extract fixture: " + err.Error())
	}
	return catalog.WithMetadata(m)
}
