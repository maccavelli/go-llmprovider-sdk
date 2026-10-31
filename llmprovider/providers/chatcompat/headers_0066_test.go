package chatcompat

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// 0066-MADR D2: a caller adds request headers by wrapping
// llmprovider.NewHTTPClient's transport, and gives the client to
// WithHTTPClient.

// headerKey0066 is the context key the wrapper reads its header from.
type headerKey0066 struct{}

// errNoHeader0066 is the wrapper's refusal of a request whose context holds
// no header.
var errNoHeader0066 = errors.New("no X-Test for this request")

// headers0066 sets X-Test from the request's context, and refuses a request
// whose context holds none, sending nothing.
type headers0066 struct{ base http.RoundTripper }

func (h headers0066) RoundTrip(r *http.Request) (*http.Response, error) {
	v, ok := r.Context().Value(headerKey0066{}).(string)
	if !ok {
		return nil, errNoHeader0066
	}
	r = r.Clone(r.Context())
	r.Header.Set("X-Test", v)
	return h.base.RoundTrip(r)
}

// TestWrappedClient_Headers_0066: a generation and a listing each carry the
// header the wrapped client sets; a request it refuses ends with no request
// reaching the server.
func TestWrappedClient_Headers_0066(t *testing.T) {
	var mu sync.Mutex
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.URL.Path+" X-Test="+r.Header.Get("X-Test"))
		mu.Unlock()
		if strings.HasSuffix(r.URL.Path, "/models") {
			_, _ = io.WriteString(w, `{"object":"list","data":[{"id":"m","object":"model"}]}`)
			return
		}
		_, _ = io.WriteString(w, fxChat)
	}))
	t.Cleanup(srv.Close)
	c := llmprovider.NewHTTPClient()
	c.Transport = headers0066{base: c.Transport}
	p := build(t, srv.URL, llmprovider.WithHTTPClient(c))
	ctx := context.WithValue(context.Background(), headerKey0066{}, "from-context")
	req := &llmprovider.Request{Input: []llmprovider.Item{llmprovider.MessageItem{Role: llmprovider.RoleUser, Text: "hi"}}}

	if _, err := llmprovider.GenerateText(ctx, p, req); err != nil {
		t.Fatalf("generation: %v", err)
	}
	if _, err := p.(llmprovider.ModelLister).ListModels(ctx); err != nil {
		t.Fatalf("listing: %v", err)
	}
	mu.Lock()
	got := append([]string(nil), seen...)
	mu.Unlock()
	listed := false
	for _, s := range got {
		if !strings.HasSuffix(s, " X-Test=from-context") {
			t.Errorf("request %q; want X-Test from the context", s)
		}
		listed = listed || strings.Contains(s, "/models ")
	}
	if len(got) < 2 || !listed {
		t.Fatalf("requests %q; want a generation and a listing", got)
	}

	_, err := llmprovider.GenerateText(context.Background(), p, req)
	if !errors.Is(err, errNoHeader0066) {
		t.Errorf("err = %v; want the wrapper's refusal", err)
	}
	mu.Lock()
	after := len(seen)
	mu.Unlock()
	if after != len(got) {
		t.Errorf("%d requests reached the server after the refusal; want none", after-len(got))
	}
}
