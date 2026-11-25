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
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/internal/wirecase"
)

// TestOpencode_RouteFromHandedDocument_0093 (MH20; D6): a model outside the
// route table takes the route its provider.npm names in the document given
// to New, at once; given no document, the name rule's.
func TestOpencode_RouteFromHandedDocument_0093(t *testing.T) {
	const model = "zz-doc-route-0093"
	meta, err := catalog.ParseMetadata(strings.NewReader(`{"opencode":{"models":{"` + model +
		`":{"limit":{"context":200000,"output":4000},"provider":{"npm":"@ai-sdk/anthropic"}}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	path := func(opts ...llmprovider.Option) string {
		var mu sync.Mutex
		var paths []string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			paths = append(paths, r.URL.Path)
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			if strings.HasSuffix(r.URL.Path, "/messages") {
				_, _ = io.WriteString(w, wirecase.MessagesReply)
				return
			}
			_, _ = io.WriteString(w, wirecase.ChatReply)
		}))
		defer srv.Close()
		p, err := NewZen(append([]llmprovider.Option{llmprovider.WithAPIKey("k"), llmprovider.WithModel(model),
			llmprovider.WithBaseURL(srv.URL), llmprovider.WithMaxTokens(64), testCache(t)}, opts...)...)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := p.Generate(context.Background(), text("hello")); err != nil {
			t.Fatalf("Generate: %v", err)
		}
		mu.Lock()
		defer mu.Unlock()
		if len(paths) != 1 {
			t.Fatalf("paths = %q; want one request", paths)
		}
		return paths[0]
	}
	if got := path(catalog.WithMetadata(meta)); !strings.HasSuffix(got, "/messages") {
		t.Errorf("with the document: path %q; want its provider.npm's /messages", got)
	}
	if got := path(); !strings.HasSuffix(got, "/chat/completions") {
		t.Errorf("with no document: path %q; want the name rule's /chat/completions", got)
	}
}
