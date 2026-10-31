package wizard

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0066-MADR D2: the wizard takes a client that adds request headers as
// Options.HTTPClient, built by wrapping llmprovider.NewHTTPClient's
// transport.

// headerKey0066 is the context key the wrapper reads its header from.
type headerKey0066 struct{}

// headers0066 sets X-Test from the request's context.
type headers0066 struct{ base http.RoundTripper }

func (h headers0066) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	if v, ok := r.Context().Value(headerKey0066{}).(string); ok {
		r.Header.Set("X-Test", v)
	}
	return h.base.RoundTrip(r)
}

// TestOptions_HTTPClientHeaders_0066: a Chat Completions server listed
// through the wizard sees the header the run's client sets from the
// context ResolveCredential was given.
func TestOptions_HTTPClientHeaders_0066(t *testing.T) {
	var mu sync.Mutex
	var listings []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		mu.Lock()
		listings = append(listings, r.Header.Get("X-Test"))
		mu.Unlock()
		_, _ = io.WriteString(w, `{"object":"list","data":[{"id":"local-model-00","object":"model"}]}`)
	}))
	t.Cleanup(srv.Close)
	c := llmprovider.NewHTTPClient()
	c.Transport = headers0066{base: c.Transport}
	o := compatOptions0066(t)
	o.HTTPClient = c
	ctx := context.WithValue(context.Background(), headerKey0066{}, "from-context")
	f := newFake(t, fakePrompter{inputs: []string{srv.URL + "/v1"}})
	_, cat, err := ResolveCredential(ctx, f, o, compatID0066)
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if !cat.Live || len(listings) == 0 || strings.Join(listings, ",") != strings.Repeat("from-context,", len(listings)-1)+"from-context" {
		t.Errorf("Live %v, listings' X-Test %q; want a live listing, each request carrying the header", cat.Live, listings)
	}
}
