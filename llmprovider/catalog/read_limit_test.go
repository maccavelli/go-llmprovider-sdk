package catalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// paddedServer answers every request with a valid JSON object of about size
// bytes, which every lister and the metadata decoder would otherwise accept.
func paddedServer(t *testing.T, size int) *httptest.Server {
	t.Helper()
	body := `{"pad":"` + strings.Repeat("a", size) + `"}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestListing_PageLimit (0021-MADR C3): a 9 MiB listing page is an error
// naming the 8 MiB limit, from every lister.
func TestListing_PageLimit(t *testing.T) {
	srv := paddedServer(t, 9<<20)
	cfg := testConfig(t, llmprovider.WithBaseURL(srv.URL))
	token := llmprovider.Token{Value: "k"}
	ctx := context.Background()
	for name, list := range map[string]func() error{
		"gemini":      func() error { _, err := fetchGeminiUsable(ctx, token, cfg); return err },
		"claude":      func() error { _, err := fetchClaudeUsable(ctx, token, cfg); return err },
		"ollama":      func() error { _, err := fetchOllamaNames(ctx, token, cfg); return err },
		"openai":      func() error { _, err := fetchOpenAIUsable(ctx, token, cfg); return err },
		"huggingface": func() error { _, err := fetchHuggingFaceUsable(ctx, token, cfg); return err },
		"together":    func() error { _, err := fetchTogetherUsable(ctx, token, cfg); return err },
		"kilo":        func() error { _, err := fetchKiloCatalog(ctx, token, cfg); return err },
	} {
		t.Run(name, func(t *testing.T) {
			if err := list(); err == nil || !strings.Contains(err.Error(), "8 MiB") {
				t.Errorf("a 9 MiB page: err = %v, want an error naming the 8 MiB limit", err)
			}
		})
	}
}

// TestModelMetadata_DocumentLimit (0021-MADR C3): a 33 MiB metadata document
// is an error naming the 32 MiB limit.
func TestModelMetadata_DocumentLimit(t *testing.T) {
	enableModelMetadata(t)
	srv := paddedServer(t, 33<<20)
	cfg := testConfig(t, llmprovider.WithModelMetadataURL(srv.URL))
	if _, err := loadModelMetadata(context.Background(), cfg); err == nil || !strings.Contains(err.Error(), "32 MiB") {
		t.Errorf("a 33 MiB document: err = %v, want an error naming the 32 MiB limit", err)
	}
}
