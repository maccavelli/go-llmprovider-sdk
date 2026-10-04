package catalog

import (
	"net/http"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestConfigFor_SharesOneClientWithoutWithHTTPClient (0020-MADR F28, kept
// within 0016-MADR D8): catalog's own calls without a provider share one
// client, so connections are reused across listings; a caller's
// WithHTTPClient, which every provider passes (R32), still wins.
func TestConfigFor_SharesOneClientWithoutWithHTTPClient(t *testing.T) {
	first, err := configFor(llmprovider.ProviderTogether, nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := configFor(llmprovider.ProviderKilo, nil)
	if err != nil {
		t.Fatal(err)
	}
	if first.HTTPClient == nil || first.HTTPClient != second.HTTPClient || defaultConfig().HTTPClient != first.HTTPClient {
		t.Error("two listings without WithHTTPClient got different clients; want one shared")
	}
	own := &http.Client{}
	cfg, err := configFor(llmprovider.ProviderTogether, []llmprovider.Option{llmprovider.WithHTTPClient(own)})
	if err != nil || cfg.HTTPClient != own {
		t.Errorf("WithHTTPClient = %v, %v; want the caller's client", cfg.HTTPClient, err)
	}
}
