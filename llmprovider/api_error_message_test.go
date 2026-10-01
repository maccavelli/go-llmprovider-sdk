package llmprovider

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestProviders_ErrorCarriesServiceMessage: every provider's error keeps the
// service's own explanation instead of discarding the body (MADR 0012 §1.1,
// 0013 B3), and still matches the sentinel its status always mapped to.
func TestProviders_ErrorCarriesServiceMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"model is unavailable"}}`))
	}))
	t.Cleanup(srv.Close)
	base := WithBaseURL(srv.URL)

	build := map[string]func() (LegacyProvider, error){
		"huggingface": func() (LegacyProvider, error) { return NewHuggingFace("k", "org/model", base) },
		"ollama":      func() (LegacyProvider, error) { return NewOllama("", "llama3", base) },
	}
	for name, newProvider := range build {
		t.Run(name, func(t *testing.T) {
			p, err := newProvider()
			if err != nil {
				t.Fatalf("construct: %v", err)
			}
			_, err = p.Generate(context.Background(), "hello")
			if err == nil || !strings.Contains(err.Error(), "model is unavailable") {
				t.Fatalf("error = %v, want it to carry the service's message", err)
			}
			if !errors.Is(err, ErrInvalidRequest) {
				t.Fatalf("error = %v, want it to still match ErrInvalidRequest", err)
			}
		})
	}
}
