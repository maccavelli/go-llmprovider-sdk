package wire

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/internal/nettest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestPost_RefusedDialIsOffline_0061 (0061-MADR D2): a generation whose dial
// is refused matches ErrOffline and ErrProviderUnavailable; a 503 matches
// neither offline nor a send failure.
func TestPost_RefusedDialIsOffline_0061(t *testing.T) {
	decode := func(io.Reader) (string, error) { return "", nil }
	call := Call{Provider: "test", Client: nettest.Refusing(), URL: nettest.RefusedURL, Body: map[string]any{},
		Prepare: func(*http.Request, llmprovider.Token) {}}
	_, err := Post(context.Background(), call, llmprovider.Token{}, decode)
	if !errors.Is(err, llmprovider.ErrOffline) || !errors.Is(err, llmprovider.ErrProviderUnavailable) {
		t.Errorf("refused dial: %v; want ErrOffline and ErrProviderUnavailable", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)
	call.Client, call.URL = srv.Client(), srv.URL
	_, err = Post(context.Background(), call, llmprovider.Token{}, decode)
	if errors.Is(err, llmprovider.ErrOffline) || !errors.Is(err, llmprovider.ErrProviderUnavailable) {
		t.Errorf("503: %v; want ErrProviderUnavailable and not ErrOffline", err)
	}
}
