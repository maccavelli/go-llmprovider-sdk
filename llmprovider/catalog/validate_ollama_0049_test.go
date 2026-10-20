package catalog_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/internal/nettest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// TestValidateOllamaURL_ClassifiesAsTheListing_0049 (0049-MADR D2c):
// ValidateOllamaURL, with no options, classifies as the listing does: an
// unreachable server matches ErrProviderUnavailable, a status other than 200
// is an *APIError.
func TestValidateOllamaURL_ClassifiesAsTheListing_0049(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	ctx := context.Background()
	if err := catalog.ValidateOllamaURL(ctx, nettest.RefusedURL); !errors.Is(err, llmprovider.ErrProviderUnavailable) {
		t.Errorf("an unreachable server: %v; want ErrProviderUnavailable", err)
	}
	err := catalog.ValidateOllamaURL(ctx, srv.URL)
	if api, ok := errors.AsType[*llmprovider.APIError](err); !ok || api.Status != http.StatusInternalServerError {
		t.Errorf("HTTP 500: %v; want an *APIError with status 500", err)
	}
}
