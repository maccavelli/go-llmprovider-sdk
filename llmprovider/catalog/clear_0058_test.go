package catalog

import (
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestResetProcessStore_EmptiesEveryURLMap_0058 (0058-MADR, amendment of
// 2026-10-23): the tests that keep the process's store clear it through
// resetProcessStore, which leaves no map keyed by a URL behind: the
// listings and Ollama's show replies. The store holds no document
// (0093-MADR D2).
func TestResetProcessStore_EmptiesEveryURLMap_0058(t *testing.T) {
	clearProcessStore(t)
	key := listingKey{provider: llmprovider.ProviderOllama, baseURL: "http://127.0.0.1:1"}
	processStore.storeListing(key, map[string]ModelInfo{"m": {ID: "m"}}, false)
	processStore.storeShow(key, "m", ModelInfo{ID: "m"}, "sha256:0")
	resetProcessStore()
	processStore.listingMu.Lock()
	listings, shows := len(processStore.listing), len(processStore.ollamaShows)
	processStore.listingMu.Unlock()
	if listings+shows != 0 {
		t.Errorf("after resetProcessStore: %d listing and %d show entries; want none", listings, shows)
	}
}
