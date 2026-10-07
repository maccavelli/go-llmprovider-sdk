package catalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestValidateOllamaURL_TrailingSlash (0026-MADR F52): a base URL ending in
// "/" requests /api/version, not //api/version, as the listing trims it
// (0020-MADR F30). The server here does not clean paths, as some proxies do
// not.
func TestValidateOllamaURL_TrailingSlash(t *testing.T) {
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		if r.URL.Path != "/api/version" {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	for _, base := range []string{srv.URL + "/", srv.URL + "//"} {
		if err := ValidateOllamaURL(context.Background(), base); err != nil || path != "/api/version" {
			t.Errorf("ValidateOllamaURL(%q) = %v, requested %q; want /api/version", base, err, path)
		}
	}
}
