package together

import (
	"net/http"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// servedDoc is the document the test's own server serves at url, read once
// and handed in with WithMetadata: the bytes the test served by URL before
// the SDK stopped fetching (0093-MADR D12, MH29).
func servedDoc(t *testing.T, url string) llmprovider.Option {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("read the served document: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	m, err := catalog.ParseMetadata(resp.Body)
	if err != nil {
		t.Fatalf("parse the served document: %v", err)
	}
	return catalog.WithMetadata(m)
}
