package catalog_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/internal/nettest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
)

// 0048-MADR D4: the errors the docs of ValidateOllamaURLWith and
// Metadata.Encode name.

// sentinels0048 are llmprovider's error kinds; a plain error matches none.
var sentinels0048 = []error{llmprovider.ErrRateLimited, llmprovider.ErrProviderUnavailable, llmprovider.ErrAuthFailure,
	llmprovider.ErrInvalidRequest, llmprovider.ErrQuotaExhausted, llmprovider.ErrNotPermitted,
	llmprovider.ErrContextOverflow, llmprovider.ErrIncomplete, llmprovider.ErrUnsupported, llmprovider.ErrInvalidProvider}

// plain0048 fails t unless err is an error matching none of the sentinels,
// whose message holds want.
func plain0048(t *testing.T, what string, err error, want string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("%s: %v; want an error saying %q", what, err, want)
		return
	}
	for _, s := range sentinels0048 {
		if errors.Is(err, s) {
			t.Errorf("%s: %v matches %v; want a plain error", what, err, s)
		}
	}
}

// TestValidateOllamaURLWith_Errors_0048: a refused option is ErrInvalidRequest;
// a malformed URL is a plain error; since 0049-MADR D2c, an unreachable server
// is ErrProviderUnavailable and a status other than 200 the listing's
// *APIError; the transport's error is wrapped, so a context that ends matches.
func TestValidateOllamaURLWith_Errors_0048(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)
	ctx := context.Background()

	foreign := llmprovider.ScopedOption("not-ollama", "test.foreign0048", 1)
	if err := catalog.ValidateOllamaURLWith(ctx, srv.URL, foreign); !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Errorf("a foreign option: %v; want ErrInvalidRequest", err)
	}
	// Since 0049-MADR D2c (and A1), classified as the listing's status is.
	status := catalog.ValidateOllamaURLWith(ctx, srv.URL)
	if api, ok := errors.AsType[*llmprovider.APIError](status); !ok || api.Status != http.StatusServiceUnavailable ||
		!errors.Is(status, llmprovider.ErrProviderUnavailable) || !strings.Contains(status.Error(), "ollama HTTP 503") {
		t.Errorf("HTTP 503: %v; want the listing's *APIError, ErrProviderUnavailable", status)
	}
	plain0048(t, "a malformed URL", catalog.ValidateOllamaURLWith(ctx, "http://[::1"), "failed to create request")

	err := catalog.ValidateOllamaURLWith(ctx, "http://127.0.0.1:1", llmprovider.WithHTTPClient(nettest.Refusing()))
	if !errors.Is(err, llmprovider.ErrProviderUnavailable) || !strings.Contains(err.Error(), "could not reach Ollama") {
		t.Errorf("a refused connection: %v; want ErrProviderUnavailable, as the listing's (0049-MADR D2c)", err)
	}
	if _, ok := errors.AsType[*url.Error](err); !ok {
		t.Errorf("a refused connection: %v; want the transport's *url.Error wrapped", err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if err := catalog.ValidateOllamaURLWith(canceled, srv.URL); !errors.Is(err, context.Canceled) {
		t.Errorf("a canceled context: %v; want context.Canceled", err)
	}
}

// failingWriter0048 refuses every write.
type failingWriter0048 struct{ err error }

func (w failingWriter0048) Write([]byte) (int, error) { return 0, w.err }

// TestMetadataEncode_WriterError_0048: a failed write is returned wrapped, so
// the writer's error matches.
func TestMetadataEncode_WriterError_0048(t *testing.T) {
	m, err := catalog.ParseMetadata(strings.NewReader(`{"anthropic":{"models":{"claude-x":{"name":"X"}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	full := errors.New("disk full")
	err = m.Encode(failingWriter0048{full})
	if !errors.Is(err, full) || !strings.Contains(err.Error(), "model metadata: encode") {
		t.Errorf("Encode = %v; want the writer's error, wrapped", err)
	}
}
