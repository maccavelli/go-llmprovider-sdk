package ollama

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/internal/nettest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestOllama_ListModelsKinds (0026-MADR F35): a failed listing has a kind
// (R25): 503 is ErrProviderUnavailable, 401 ErrAuthFailure, and an
// unreachable server ErrProviderUnavailable.
func TestOllama_ListModelsKinds(t *testing.T) {
	status := func(code int) string {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(code) }))
		t.Cleanup(srv.Close)
		return srv.URL
	}
	for _, c := range []struct {
		name, base string
		client     *http.Client
		kind       error
	}{
		{"503", status(http.StatusServiceUnavailable), nil, llmprovider.ErrProviderUnavailable},
		{"401", status(http.StatusUnauthorized), nil, llmprovider.ErrAuthFailure},
		// Refused without a freed port to meet (0045-MADR D1).
		{"unreachable", nettest.RefusedURL, nettest.Refusing(), llmprovider.ErrProviderUnavailable},
	} {
		opts := []llmprovider.Option{llmprovider.WithBaseURL(c.base), llmprovider.WithModelProbes(false)}
		if c.client != nil {
			opts = append(opts, llmprovider.WithHTTPClient(c.client))
		}
		p, err := New(opts...)
		if err != nil {
			t.Fatalf("%s: New: %v", c.name, err)
		}
		_, err = p.(llmprovider.ModelLister).ListModels(context.Background())
		if !errors.Is(err, c.kind) {
			t.Errorf("%s: ListModels = %v; want %v", c.name, err, c.kind)
		}
	}
}
