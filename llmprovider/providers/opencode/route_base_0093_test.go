package opencode

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
)

// otherHost0093 passes requests to its own server, and answers every other
// one, and any request for an api.json, with 404, counting it.
type otherHost0093 struct {
	host string
	mu   sync.Mutex
	seen []string
}

func (o *otherHost0093) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Host != o.host || strings.HasSuffix(req.URL.Path, "api.json") {
		o.mu.Lock()
		o.seen = append(o.seen, req.URL.String())
		o.mu.Unlock()
		return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("")),
			Header: http.Header{}, Request: req}, nil
	}
	return http.DefaultTransport.RoundTrip(req)
}

// TestOpencode_RouteNoWait_0093 (MH20; D6): a model outside the route table,
// on a provider given no document and no pinned route, takes the name rule's
// route at once: no request for a document is made, and none is waited for.
func TestOpencode_RouteNoWait_0093(t *testing.T) {
	var paths []string
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, wirecase.ChatReply)
	}))
	defer srv.Close()
	counter := &otherHost0093{host: strings.TrimPrefix(srv.URL, "http://")}
	p, err := NewZen(llmprovider.WithAPIKey("k"), llmprovider.WithModel("zz-unrouted-0093"), llmprovider.WithBaseURL(srv.URL),
		llmprovider.WithHTTPClient(&http.Client{Transport: counter}), llmprovider.WithMaxTokens(64), testCache(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Generate(context.Background(), text("hello")); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	counter.mu.Lock()
	seen := append([]string(nil), counter.seen...)
	counter.mu.Unlock()
	if len(seen) != 0 {
		t.Errorf("requests for a document: %q; want none", seen)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(paths) != 1 || !strings.HasSuffix(paths[0], "/chat/completions") {
		t.Errorf("paths = %q; want one /chat/completions, the name rule's route", paths)
	}
}
