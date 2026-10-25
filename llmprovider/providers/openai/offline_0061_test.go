package openai

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/internal/nettest"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// TestChatGPTListing_RefusedDialIsOffline_0061 (0061-MADR D2): the ChatGPT
// catalog's request, its dial refused, matches ErrOffline.
func TestChatGPTListing_RefusedDialIsOffline_0061(t *testing.T) {
	p := &provider{client: nettest.Refusing(), logger: slog.New(slog.DiscardHandler)}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, nettest.RefusedURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.fetchCatalog(req, "chatgpt"); !errors.Is(err, llmprovider.ErrOffline) {
		t.Errorf("fetchCatalog = %v; want ErrOffline", err)
	}
}
