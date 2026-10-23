package opencode

import (
	"context"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// TestOpencode_ReusedPortRoutesByItsOwnDocument_0058 (0058-MADR D3): httptest
// reuses ports, and CI's Linux runners gave one test's server the port of an
// earlier one, whose document the process's cache still held. Here two
// subtests, as two tests, serve different documents on one port in turn; the
// second routes by its own.
func TestOpencode_ReusedPortRoutesByItsOwnDocument_0058(t *testing.T) {
	gw, model := llmprovider.ProviderOpencodeGo, "brand-new-model"
	warm := func(t *testing.T, url string) {
		t.Helper()
		if _, err := catalog.LookupMetadataWith(context.Background(), gw,
			llmprovider.WithModelMetadataURL(url+"/api.json"), testCache(t)); err != nil {
			t.Fatalf("warming the metadata cache: %v", err)
		}
	}
	var addr, firstURL string
	t.Run("first", func(t *testing.T) {
		srv, paths := routeServerAt(t, "", gw, model, "@ai-sdk/openai-compatible", true)
		warm(t, srv.URL)
		if got := generatePath(t, srv, paths, gw, model); got != "/chat/completions" {
			t.Fatalf("path = %q, want %q", got, "/chat/completions")
		}
		addr, firstURL = srv.Listener.Addr().String(), srv.URL
		srv.Close()
	})
	if addr == "" {
		t.Fatal("the first subtest gave no address")
	}
	t.Run("second", func(t *testing.T) {
		srv, paths := routeServerAt(t, addr, gw, model, "@ai-sdk/openai", true)
		if srv.URL != firstURL {
			t.Fatalf("the second server is at %s; want the first's %s", srv.URL, firstURL)
		}
		warm(t, srv.URL)
		if got := generatePath(t, srv, paths, gw, model); got != "/responses" {
			t.Errorf("path = %q, want %q", got, "/responses")
		}
	})
}
