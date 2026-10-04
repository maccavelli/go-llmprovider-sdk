package catalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// countingTransport counts the requests that go through it.
type countingTransport struct{ n atomic.Int32 }

func (c *countingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	c.n.Add(1)
	return http.DefaultTransport.RoundTrip(r)
}

// TestValidateOllamaURLWith_UsesTheCallersClientAndIdentity (0020-MADR F37,
// Q5 a and its amendment): the check goes through the caller's client and
// names the caller's application.
func TestValidateOllamaURLWith_UsesTheCallersClientAndIdentity(t *testing.T) {
	var agent atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		agent.Store(r.UserAgent())
		_, _ = w.Write([]byte(`{"version":"0.12.0"}`))
	}))
	t.Cleanup(srv.Close)
	rt := &countingTransport{}
	err := ValidateOllamaURLWith(context.Background(), srv.URL,
		llmprovider.WithHTTPClient(&http.Client{Transport: rt}), llmprovider.WithClientInfo("wire-app", "9.9.9"))
	if err != nil {
		t.Fatal(err)
	}
	if rt.n.Load() != 1 {
		t.Errorf("the caller's client carried %d request(s), want 1", rt.n.Load())
	}
	if got, _ := agent.Load().(string); !strings.HasPrefix(got, "wire-app/9.9.9 ") {
		t.Errorf("User-Agent = %q, want it to start wire-app/9.9.9", got)
	}
	if err := ValidateOllamaURL(context.Background(), srv.URL); err != nil {
		t.Errorf("ValidateOllamaURL = %v, want nil", err)
	}
}
