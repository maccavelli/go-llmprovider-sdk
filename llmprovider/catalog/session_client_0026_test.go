package catalog

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// clientTakingSource is a token source that refreshes through a client it
// is given, as *auth.OAuthSession does.
type clientTakingSource struct {
	client *http.Client
}

func (s *clientTakingSource) Token(context.Context) (llmprovider.Token, error) {
	return llmprovider.Token{Value: "k"}, nil
}

func (s *clientTakingSource) UseDefaultHTTPClient(c *http.Client) {
	if s.client == nil {
		s.client = c
	}
}

// TestList_GivesASessionItsClient (0026-MADR F10): a listing hands its client
// to a source that refreshes with one and has none, so a refresh the listing
// causes goes through the caller's client. It is given as a default, which a
// caller's own client later replaces (0026-MADR F37).
func TestList_GivesASessionItsClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"data":[{"id":"grok-4.5"}]}`)
	}))
	defer srv.Close()
	client := &http.Client{}
	src := &clientTakingSource{}
	if _, err := List(context.Background(), llmprovider.ProviderGrok, src, llmprovider.WithBaseURL(srv.URL),
		llmprovider.WithHTTPClient(client)); err != nil {
		t.Fatal(err)
	}
	if src.client != client {
		t.Fatal("the source was not given the listing's client")
	}
}
