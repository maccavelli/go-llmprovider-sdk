package ollama

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

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
	closed := httptest.NewServer(http.NotFoundHandler())
	unreachable := closed.URL
	closed.Close()
	for _, c := range []struct {
		name, base string
		kind       error
	}{
		{"503", status(http.StatusServiceUnavailable), llmprovider.ErrProviderUnavailable},
		{"401", status(http.StatusUnauthorized), llmprovider.ErrAuthFailure},
		{"unreachable", unreachable, llmprovider.ErrProviderUnavailable},
	} {
		p, err := New(llmprovider.WithBaseURL(c.base), llmprovider.WithModelProbes(false))
		if err != nil {
			t.Fatalf("%s: New: %v", c.name, err)
		}
		_, err = p.(llmprovider.ModelLister).ListModels(context.Background())
		if !errors.Is(err, c.kind) {
			t.Errorf("%s: ListModels = %v; want %v", c.name, err, c.kind)
		}
	}
}
